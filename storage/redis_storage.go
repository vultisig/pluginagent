package storage

import (
	"context"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vultisig/vultiserver/contexthelper"
)

// RedisConfig holds the configuration parameters for connecting to a Redis instance.
// Fields:
// - Host: The hostname or IP address of the Redis server.
// - Port: The port number on which the Redis server is listening.
// - User: The username for authentication (if required by the Redis server).
// - Password: The password for authentication (if required by the Redis server).
// - DB: The Redis database number to use (default is 0).
type RedisConfig struct {
	Host     string `mapstructure:"host" json:"host,omitempty"`
	Port     string `mapstructure:"port" json:"port,omitempty"`
	User     string `mapstructure:"user" json:"user,omitempty"`
	Password string `mapstructure:"password" json:"password,omitempty"`
	DB       int    `mapstructure:"db" json:"db,omitempty"`
}

type RedisStorage struct {
	cfg    RedisConfig
	client *redis.Client
}

type RedisStorageOption func(*RedisStorage)

func WithHost(host string) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.cfg.Host = host
	}
}

func WithPort(port string) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.cfg.Port = port
	}
}

func WithUser(user string) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.cfg.User = user
	}
}

func WithPassword(password string) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.cfg.Password = password
	}
}

func WithDB(db int) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.cfg.DB = db
	}
}

func WithConfig(cfg RedisConfig) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.cfg = cfg
	}
}

func NewRedisStorage(opts ...RedisStorageOption) (*RedisStorage, error) {
	rs := &RedisStorage{
		cfg: RedisConfig{
			Host: "localhost",
			Port: "6379",
			DB:   0,
		},
	}

	for _, opt := range opts {
		opt(rs)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(rs.cfg.Host, rs.cfg.Port),
		Username: rs.cfg.User,
		Password: rs.cfg.Password,
		DB:       rs.cfg.DB,
	})
	status := client.Ping(context.Background())
	if status.Err() != nil {
		return nil, status.Err()
	}
	rs.client = client
	return rs, nil
}

func (r *RedisStorage) Get(ctx context.Context, key string) (string, error) {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return "", err
	}
	return r.client.Get(ctx, key).Result()
}

func (r *RedisStorage) Set(ctx context.Context, key string, value string, expiry time.Duration) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Set(ctx, key, value, expiry).Err()
}

func (r *RedisStorage) Expire(ctx context.Context, key string, expiry time.Duration) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Expire(ctx, key, expiry).Err()
}

func (r *RedisStorage) Delete(ctx context.Context, key string) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Del(ctx, key).Err()
}

func (r *RedisStorage) Close() error {
	return r.client.Close()
}
