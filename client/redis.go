package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Password        string `yaml:"password"`
	DB              int    `yaml:"db"`
	PoolSize        int    `yaml:"pool_size"`
	MinIdleConns    int    `yaml:"min_idle_conns"`
	MaxConnAge      int    `yaml:"max_conn_age"`
	PoolTimeout     int    `yaml:"pool_timeout"`
	IdleTimeout     int    `yaml:"idle_timeout"`
	IdleCheckFreq   int    `yaml:"idle_check_freq"`
}

type RedisClient struct {
	client *redis.Client
}

var (
	redisClients map[string]*RedisClient
	redisMutex   sync.RWMutex
)

func init() {
	redisClients = make(map[string]*RedisClient)
}

func InitRedis(name string, config RedisConfig) error {
	redisMutex.Lock()
	defer redisMutex.Unlock()

	if _, exists := redisClients[name]; exists {
		return fmt.Errorf("Redis client '%s' already exists", name)
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	options := &redis.Options{
		Addr:     addr,
		Password: config.Password,
		DB:       config.DB,
	}

	setConnectionDefaults := func() {
		if config.PoolSize > 0 {
			options.PoolSize = config.PoolSize
		} else {
			options.PoolSize = 10
		}

		if config.MinIdleConns > 0 {
			options.MinIdleConns = config.MinIdleConns
		} else {
			options.MinIdleConns = 2
		}

		if config.MaxConnAge > 0 {
			options.MaxConnAge = time.Duration(config.MaxConnAge) * time.Second
		} else {
			options.MaxConnAge = time.Hour
		}

		if config.PoolTimeout > 0 {
			options.PoolTimeout = time.Duration(config.PoolTimeout) * time.Second
		} else {
			options.PoolTimeout = 4 * time.Second
		}

		if config.IdleTimeout > 0 {
			options.IdleTimeout = time.Duration(config.IdleTimeout) * time.Second
		} else {
			options.IdleTimeout = 5 * time.Minute
		}

		if config.IdleCheckFreq > 0 {
			options.IdleCheckFrequency = time.Duration(config.IdleCheckFreq) * time.Second
		} else {
			options.IdleCheckFrequency = time.Minute
		}
	}

	setConnectionDefaults()

	rdb := redis.NewClient(options)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return fmt.Errorf("failed to ping Redis server for '%s': %v", name, err)
	}

	redisClients[name] = &RedisClient{client: rdb}
	return nil
}

func GetRedis(name string) (*RedisClient, error) {
	redisMutex.RLock()
	defer redisMutex.RUnlock()

	client, exists := redisClients[name]
	if !exists {
		return nil, fmt.Errorf("Redis client '%s' not found", name)
	}

	return client, nil
}

func GetDefaultRedis() (*RedisClient, error) {
	return GetRedis("default")
}

func (c *RedisClient) Client() *redis.Client {
	return c.client
}

func (c *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return c.client.Set(ctx, key, value, expiration)
}

func (c *RedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return c.client.Get(ctx, key)
}

func (c *RedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Del(ctx, keys...)
}

func (c *RedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Exists(ctx, keys...)
}

func (c *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return c.client.Expire(ctx, key, expiration)
}

func (c *RedisClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.HSet(ctx, key, values...)
}

func (c *RedisClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return c.client.HGet(ctx, key, field)
}

func (c *RedisClient) HGetAll(ctx context.Context, key string) *redis.StringStringMapCmd {
	return c.client.HGetAll(ctx, key)
}

func (c *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.LPush(ctx, key, values...)
}

func (c *RedisClient) RPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.RPop(ctx, key)
}

func (c *RedisClient) Close() error {
	return c.client.Close()
}

func CloseAllRedis() {
	redisMutex.Lock()
	defer redisMutex.Unlock()

	for name, client := range redisClients {
		if err := client.Close(); err != nil {
			fmt.Printf("Error closing Redis client '%s': %v\n", name, err)
		}
	}

	redisClients = make(map[string]*RedisClient)
}