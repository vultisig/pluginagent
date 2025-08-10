package main

import (
	"fmt"
	"net"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/pluginagent/api"
	"github.com/vultisig/pluginagent/config"
	"github.com/vultisig/pluginagent/storage"
	"github.com/vultisig/verifier/vault"
)

func main() {
	cfg, err := config.LoadServerConfig()
	if err != nil {
		panic(err)
	}
	logger := logrus.New()

	redisStorage, err := storage.NewRedisStorage(storage.WithConfig(storage.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		User:     cfg.Redis.User,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}))
	if err != nil {
		panic(err)
	}

	redisOptions := asynq.RedisClientOpt{
		Addr:     net.JoinHostPort(cfg.Redis.Host, cfg.Redis.Port),
		Username: cfg.Redis.User,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	client := asynq.NewClient(redisOptions)
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Println("fail to close asynq client,", err)
		}
	}()

	inspector := asynq.NewInspector(redisOptions)

	vaultStorage, err := vault.NewBlockStorageImp(cfg.BlockStorage)
	if err != nil {
		panic(err)
	}

	db, err := storage.NewDatabaseStorage(storage.StorageConfig{
		Type: storage.StorageTypePostgreSQL,
		DSN:  cfg.Database.DSN,
	})
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	server := api.NewServer(
		cfg.Server,
		db,
		redisStorage,
		vaultStorage,
		client,
		inspector,
	)

	if err := server.StartServer(); err != nil {
		panic(err)
	}
}
