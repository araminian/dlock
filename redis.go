package dlock

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrRedisClientNotConnected = errors.New("Can't connect to redis server")
)

type RedisLocker struct {
	host   string
	port   string
	client *redis.Client
}

type RedisLockerOption func(*RedisLocker)

func WithRedisHost(host string) RedisLockerOption {
	return func(r *RedisLocker) {
		r.host = host
	}
}

func WithRedisPort(port string) RedisLockerOption {
	return func(r *RedisLocker) {
		r.port = port
	}
}

func NewRedisLocker(opts ...RedisLockerOption) Locker {
	locker := &RedisLocker{
		host: "localhost",
		port: "6379",
	}
	for _, opt := range opts {
		opt(locker)
	}

	locker.client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", locker.host, locker.port),
	})

	return locker
}

// NewLock creates a new lock.
func (r *RedisLocker) NewLock(name string) (*Lock, error) {

	lock := &Lock{
		id:   uuid.New(),
		Name: name,
		mu:   &sync.Mutex{},
	}

	// Check if the redis client is connected
	if err := r.client.Ping(context.TODO()).Err(); err != nil {
		return nil, errors.Join(ErrRedisClientNotConnected, err)
	}

	return lock, nil
}
