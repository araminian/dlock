package dlock

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

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
func (r *RedisLocker) NewLock(name string, group string) (*Lock, error) {

	lock := &Lock{
		userID: uuid.New(),
		name:   name,
		group:  group,
	}

	// Check if the redis client is connected
	if err := r.client.Ping(context.TODO()).Err(); err != nil {
		return nil, errors.Join(ErrRedisClientNotConnected, err)
	}

	return lock, nil
}

// isLockTaken checks if the lock is taken.
func (r *RedisLocker) isLockTaken(lock *Lock) (bool, error) {
	lockKey := fmt.Sprintf("%s:%s", lock.group, lock.name)

	res, err := r.client.LLen(context.TODO(), lockKey).Result()
	if err != nil {
		return false, errors.Join(ErrLockerError, err)
	}

	return res > 0, nil
}

// amIOwner checks if the current user is the owner of the lock.
func (r *RedisLocker) amIOwner(lock *Lock) (bool, error) {

	lockKey := fmt.Sprintf("%s:%s", lock.group, lock.name)
	res, err := r.client.LRange(context.TODO(), lockKey, 0, -1).Result()
	if err != nil {
		return false, errors.Join(ErrLockerError, err)
	}

	return slices.Contains(res, lock.userID.String()), nil
}

// Lock locks the lock, using redis list to store the lock.
func (r *RedisLocker) Lock(lock *Lock) error {

	// Check if the lock is already taken
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		isTaken, err := r.isLockTaken(lock)
		if err != nil {
			return errors.Join(ErrLockerError, err)
		}

		if !isTaken {
			break
		}

		<-ticker.C
	}

	lockKey := fmt.Sprintf("%s:%s", lock.group, lock.name)

	_, err := r.client.LPush(context.TODO(), lockKey, lock.userID.String()).Result()

	if err != nil {
		return errors.Join(ErrLockerError, err)
	}

	return nil

}

// Unlock unlocks the lock.
func (r *RedisLocker) Unlock(lock *Lock) error {

	// check if the lock is unlocked
	isTaken, err := r.isLockTaken(lock)
	if err != nil {
		return errors.Join(ErrLockerError, err)
	}

	if !isTaken {
		return ErrUnlockUnlocked
	}

	// Check if the lock is owned by the current user
	amIOwner, err := r.amIOwner(lock)
	if err != nil {
		return errors.Join(ErrLockerError, err)
	}

	if !amIOwner {
		return ErrNotOwner
	}

	lockKey := fmt.Sprintf("%s:%s", lock.group, lock.name)
	_, err = r.client.LPop(context.TODO(), lockKey).Result()
	if err != nil {
		return errors.Join(ErrLockerError, err)
	}

	return nil

}

// Owner returns the owner of the lock.
func (r *RedisLocker) Owner(lock *Lock) (bool, uuid.UUID, error) {

	// Check if the lock is taken
	isTaken, err := r.isLockTaken(lock)
	if err != nil {
		return false, uuid.Nil, errors.Join(ErrLockerError, err)
	}

	if !isTaken {
		return false, uuid.Nil, nil
	}

	lockKey := fmt.Sprintf("%s:%s", lock.group, lock.name)
	res, err := r.client.LRange(context.TODO(), lockKey, 0, -1).Result()
	if err != nil {
		return false, uuid.Nil, errors.Join(ErrLockerError, err)
	}

	return true, uuid.MustParse(res[0]), nil
}

// IsLocked returns true if the lock is locked.
func (r *RedisLocker) IsLocked(lock *Lock) (bool, error) {
	return r.isLockTaken(lock)
}

// IsLockedByMe returns true if the lock is locked by the current user.
func (r *RedisLocker) IsLockedByMe(lock *Lock) (bool, error) {
	return r.amIOwner(lock)
}

// TryLock tries to lock the lock.
func (r *RedisLocker) TryLock(lock *Lock) (bool, error) {

	// Check if the lock is already taken
	isTaken, err := r.isLockTaken(lock)
	if err != nil {
		return false, errors.Join(ErrLockerError, err)
	}

	if isTaken {
		return false, nil
	}

	lockKey := fmt.Sprintf("%s:%s", lock.group, lock.name)
	_, err = r.client.LPush(context.TODO(), lockKey, lock.userID.String()).Result()
	if err != nil {
		return false, errors.Join(ErrLockerError, err)
	}

	return true, nil
}
