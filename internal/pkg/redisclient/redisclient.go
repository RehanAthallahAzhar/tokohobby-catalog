package redisclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient struct yang membungkus *redis.Client
type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient() (*RedisClient, error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default Redis address
		log.Printf("REDIS_ADDR not set, using default: %s", redisAddr)
	}

	redisPassword := os.Getenv("REDIS_PASSWORD") // Default kosong jika tidak ada password
	redisDBStr := os.Getenv("REDIS_DB")
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		redisDB = 0 // Default DB 0 jika tidak diset atau salah format
		log.Printf("REDIS_DB not set or invalid, using default DB: %d", redisDB)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPassword,
		DB:           redisDB,
		PoolSize:     10, // Number of connections in the pool
		MinIdleConns: 5,  // Number of connections in idle minimum
	})

	// Ping Redis to verify the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}
	log.Printf("Redis connected: %s", pong)

	return &RedisClient{Client: rdb}, nil
}

func (rc *RedisClient) Close() {
	if rc.Client != nil {
		log.Println("Menutup koneksi Redis...")
		err := rc.Client.Close()
		if err != nil {
			log.Printf("Gagal menutup koneksi Redis: %v", err)
		}
	}
}

// Anda dapat menambahkan metode utilitas Redis umum di sini jika diperlukan,
// seperti Get, Set, Delete, dll.
// func (rc *RedisClient) Get(ctx context.Context, key string) (string, error) {
// 	return rc.Client.Get(ctx, key).Result()
// }
// func (rc *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
// 	return rc.Client.Set(ctx, key, value, expiration).Err()
// }
