package main

import (
	"fmt"
	"net"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/pluginagent/config"
	"github.com/vultisig/verifier/plugin/tasks"
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
	mux.HandleFunc(tasks.TypeKeyGenerationDKLS, vaultMgmService.HandleKeyGenerationDKLS)
	mux.HandleFunc(tasks.TypeKeySignDKLS, vaultMgmService.HandleKeySignDKLS)
	mux.HandleFunc(tasks.TypeReshareDKLS, vaultMgmService.HandleReshareDKLS)

	if err := srv.Run(mux); err != nil {
		panic(fmt.Errorf("could not run server: %w", err))
	}
}
