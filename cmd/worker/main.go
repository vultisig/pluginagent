package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/pluginagent/config"
	"github.com/vultisig/pluginagent/storage"
	"github.com/vultisig/pluginagent/storage/interfaces"
	"github.com/vultisig/pluginagent/types"
	"github.com/vultisig/verifier/plugin/tasks"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"
)

func main() {
	cfg, err := config.LoadWorkerConfig()
	if err != nil {
		panic(err)
	}

	redisCfg := cfg.Redis
	redisOptions := asynq.RedisClientOpt{
		Addr:     net.JoinHostPort(redisCfg.Host, redisCfg.Port),
		Username: redisCfg.User,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	}

	db, err := storage.NewDatabaseStorage(storage.StorageConfig{
		Type: storage.StorageTypePostgreSQL,
		DSN:  cfg.Database.DSN,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize redis storage: %v", err))
	}

	logger := logrus.StandardLogger()
	client := asynq.NewClient(redisOptions)
	vaultStorage, err := vault.NewBlockStorageImp(cfg.BlockStorage)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize vault storage: %v", err))
	}

	srv := asynq.NewServer(
		redisOptions,
		asynq.Config{
			Logger:      logger,
			Concurrency: 10,
			Queues: map[string]int{
				tasks.QUEUE_NAME: 10,
			},
		},
	)

	vaultMgmService, err := vault.NewManagementService(
		cfg.VaultService,
		client,
		nil,
		vaultStorage,
		nil,
	)

	if err != nil {
		panic(fmt.Sprintf("failed to initialize vault management service: %v", err))
	}

	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeKeyGenerationDKLS, resultWriter(db, vaultMgmService.HandleKeyGenerationDKLS))
	mux.HandleFunc(tasks.TypeKeySignDKLS, resultWriter(db, vaultMgmService.HandleKeySignDKLS))
	mux.HandleFunc(tasks.TypeReshareDKLS, resultWriter(db, vaultMgmService.HandleReshareDKLS))

	if err := srv.Run(mux); err != nil {
		panic(fmt.Errorf("could not run server: %w", err))
	}
}

func resultWriter(db interfaces.DatabaseStorage, handler asynq.HandlerFunc) asynq.HandlerFunc {
	return func(ctx context.Context, task *asynq.Task) error {
		err := handler(ctx, task)
		if err != nil {
			return err
		}

		switch task.Type() {
		case tasks.TypeReshareDKLS:
			var taskData vtypes.ReshareRequest
			if err := json.Unmarshal(task.Payload(), &taskData); err != nil {
				return err
			}

			// Record vault resharing event
			event := &types.SystemEvent{
				PublicKey: &taskData.PublicKey,
				PolicyID:  nil,
				EventType: types.SystemEventTypeVaultReshared,
				EventData: task.Payload(),
			}
			_, err = db.InsertEvent(ctx, event)
			if err != nil {
				fmt.Printf("failed to insert event: %v", err)
				return nil
			}
		}

		return nil
	}
}
