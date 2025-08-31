package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/plugin/service"
	"github.com/vultisig/pluginagent/config"
	"github.com/vultisig/pluginagent/policy"
	"github.com/vultisig/pluginagent/storage"
	"github.com/vultisig/pluginagent/storage/interfaces"
	"github.com/vultisig/pluginagent/types"
	vcommon "github.com/vultisig/verifier/common"
	vv "github.com/vultisig/verifier/common/vultisig_validator"
	"github.com/vultisig/verifier/plugin/tasks"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"
)

type Server struct {
	cfg           config.ServerConfig
	pluginCfg     config.PluginConfig
	db            interfaces.DatabaseStorage
	redis         *storage.RedisStorage
	vaultStorage  vault.Storage
	client        *asynq.Client
	inspector     *asynq.Inspector
	sdClient      *statsd.Client
	policyService service.Policy
	logger        *logrus.Logger
	mode          string
}

// NewServer returns a new server.
func NewServer(
	cfg config.ServerConfig,
	pluginCfg config.PluginConfig,
	db interfaces.DatabaseStorage,
	redis *storage.RedisStorage,
	vaultStorage vault.Storage,
	client *asynq.Client,
	inspector *asynq.Inspector,
) *Server {
	logger := logrus.WithField("service", "plugin").Logger

	policyService, err := policy.NewPolicyService(db, logger.WithField("service", "policy").Logger)
	if err != nil {
		logger.Fatalf("Failed to initialize policy service: %v", err)
	}

	return &Server{
		cfg:           cfg,
		pluginCfg:     pluginCfg,
		redis:         redis,
		client:        client,
		inspector:     inspector,
		vaultStorage:  vaultStorage,
		db:            db,
		logger:        logger,
		policyService: policyService,
	}
}

func (s *Server) StartServer() error {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("2M")) // set maximum allowed size for a request body to 2M
	e.Use(s.statsdMiddleware)
	e.Use(middleware.CORS())
	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(
		middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 30, ExpiresIn: 5 * time.Minute},
	)
	e.Use(middleware.RateLimiter(limiterStore))

	e.Validator = &vv.VultisigValidator{Validator: validator.New()}

	e.GET("/ping", s.Ping)

	grp := e.Group("/vault")
	grp.POST("/reshare", s.ReshareVault)
	grp.GET("/get/:pluginId/:publicKeyECDSA", s.GetVault)     // Get Vault Data
	grp.GET("/exist/:pluginId/:publicKeyECDSA", s.ExistVault) // Check if Vault exists
	grp.POST("/sign", s.SignMessages)                         // Sign messages
	grp.GET("/sign/response/:taskId", s.GetKeysignResult)     // Get keysign result
	grp.DELETE("/:pluginId/:publicKeyECDSA", s.DeleteVault)   // Delete Vault

	pluginGroup := e.Group("/plugin")

	// policy mode is always available since it is used by both verifier server and plugin server
	pluginGroup.POST("/policy", s.CreatePluginPolicy)
	pluginGroup.PUT("/policy", s.UpdatePluginPolicyById)
	pluginGroup.GET("/recipe-specification", s.GetRecipeSpecification)
	pluginGroup.DELETE("/policy/:policyId", s.DeletePluginPolicyById)

	return e.Start(fmt.Sprintf(":%d", s.cfg.Port))
}

func (s *Server) Ping(c echo.Context) error {
	return c.String(http.StatusOK, "Plugin agent is running")
}

// ReshareVault is a handler to reshare a vault
func (s *Server) ReshareVault(c echo.Context) error {
	var req vtypes.ReshareRequest
	if err := c.Bind(&req); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}
	if err := req.IsValid(); err != nil {
		return fmt.Errorf("invalid request, err: %w", err)
	}
	buf, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("fail to marshal to json, err: %w", err)
	}
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 5*time.Minute); err != nil {
		s.logger.Errorf("fail to set session, err: %v", err)
	}
	_, err = s.client.Enqueue(asynq.NewTask(tasks.TypeReshareDKLS, buf),
		asynq.MaxRetry(-1),
		asynq.Timeout(7*time.Minute),
		asynq.Retention(10*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))
	if err != nil {
		return fmt.Errorf("fail to enqueue task, err: %w", err)
	}

	// Record vault resharing event
	// TODO: move this to post-reshare
	event := &types.SystemEvent{
		PublicKey: &req.PublicKey,
		PolicyID:  nil,
		EventType: types.SystemEventTypeVaultReshared,
		EventData: buf,
	}
	_, err = s.db.InsertEvent(c.Request().Context(), event)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error()))
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) GetVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	if publicKeyECDSA == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("public key is required"))
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("pluginId is required"))
	}

	filePathName := vcommon.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	content, err := s.vaultStorage.GetVault(filePathName)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to read file in GetVault, err: %w", err)
		s.logger.Error(wrappedErr)
		return wrappedErr
	}

	v, err := vcommon.DecryptVaultFromBackup(s.cfg.EncryptionSecret, content)
	if err != nil {
		s.logger.WithError(err).Error("fail to decrypt vault")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("fail to get vault"))
	}

	return c.JSON(http.StatusOK, vtypes.VaultGetResponse{
		Name:           v.Name,
		PublicKeyEcdsa: v.PublicKeyEcdsa,
		PublicKeyEddsa: v.PublicKeyEddsa,
		HexChainCode:   v.HexChainCode,
		LocalPartyId:   v.LocalPartyId,
	})
}

// SignMessages is a handler to process Keysing request
func (s *Server) SignMessages(c echo.Context) error {
	s.logger.Debug("VERIFIER SERVER: SIGN MESSAGES")
	var req vtypes.KeysignRequest
	if err := c.Bind(&req); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}
	if err := req.IsValid(); err != nil {
		return fmt.Errorf("invalid request, err: %w", err)
	}
	if !s.isValidHash(req.PublicKey) {
		return c.NoContent(http.StatusBadRequest)
	}
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 30*time.Minute); err != nil {
		s.logger.Errorf("fail to set session, err: %v", err)
	}

	filePathName := vcommon.GetVaultBackupFilename(req.PublicKey, req.PluginID)
	content, err := s.vaultStorage.GetVault(filePathName)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to read file in SignMessages, err: %w", err)
		s.logger.Infof("fail to read file in SignMessages, err: %v", err)
		s.logger.Error(wrappedErr)
		return wrappedErr
	}

	_, err = vcommon.DecryptVaultFromBackup(s.cfg.EncryptionSecret, content)
	if err != nil {
		return fmt.Errorf("fail to decrypt vault from the backup, err: %w", err)
	}
	buf, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("fail to marshal to json, err: %w", err)
	}

	ti, err := s.client.EnqueueContext(c.Request().Context(),
		asynq.NewTask(tasks.TypeKeySignDKLS, buf),
		asynq.MaxRetry(-1),
		asynq.Timeout(2*time.Minute),
		asynq.Retention(5*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))

	if err != nil {
		return fmt.Errorf("fail to enqueue task, err: %w", err)
	}

	return c.JSON(http.StatusOK, ti.ID)

}

// GetKeysignResult is a handler to get the keysign response
func (s *Server) GetKeysignResult(c echo.Context) error {
	taskID := c.Param("taskId")
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	result, err := tasks.GetTaskResult(s.inspector, taskID)
	if err != nil {
		if err.Error() == "task is still in progress" {
			return c.JSON(http.StatusOK, "Task is still in progress")
		}
		return err
	}

	return c.JSON(http.StatusOK, result)
}
func (s *Server) DeleteVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	pluginId := c.Param("pluginId")

	if publicKeyECDSA == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("public key is required"))
	}

	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}

	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("pluginId is required"))
	}

	fileName := vcommon.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	if err := s.vaultStorage.DeleteFile(fileName); err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error()))
	}

	// Record vault deletion event
	event := &types.SystemEvent{
		PublicKey: &publicKeyECDSA,
		PolicyID:  nil,
		EventType: types.SystemEventTypeVaultDeleted,
		EventData: nil,
	}
	_, err := s.db.InsertEvent(c.Request().Context(), event)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error()))
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) isValidHash(hash string) bool {
	if len(hash) != 66 {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil
}

func (s *Server) ExistVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	if publicKeyECDSA == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("public key is required"))
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("plugin id is required"))
	}

	filePathName := vcommon.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	exist, err := s.vaultStorage.Exist(filePathName)
	if err != nil || !exist {
		return c.NoContent(http.StatusBadRequest)
	}
	return c.NoContent(http.StatusOK)
}
