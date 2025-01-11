package dlock

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testRedis struct {
	container testcontainers.Container
	port      string
	host      string
}

func setupRedis(t *testing.T) *testRedis {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	mappedPort, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	return &testRedis{
		container: container,
		port:      mappedPort.Port(),
		host:      host,
	}
}

func (tr *testRedis) GetPort() int {
	port, _ := strconv.Atoi(tr.port)
	return port
}

func (tr *testRedis) Cleanup(ctx context.Context) error {
	return tr.container.Terminate(ctx)
}

func TestFailToConnectToRedis(t *testing.T) {

	// Create a new redis locker, which will fail to connect to the redis server
	locker := NewRedisLocker(WithRedisHost("localhost"), WithRedisPort("6666"))

	_, err := locker.NewLock("testlock", "testgroup")

	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	if !errors.Is(err, ErrRedisClientNotConnected) {
		t.Errorf("Expected error %v, got %v", ErrRedisClientNotConnected, err)
	}

}

func TestSuccessfullyCreateLock(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	require.NotNil(t, lock)
}

func TestSuccessfullyLock(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	require.NotNil(t, lock)

	// Lock the lock
	err = locker.Lock(lock)
	require.NoError(t, err)

	// Check if we are the owner
	amIOwner, ownerID, err := locker.Owner(lock)
	require.NoError(t, err)
	require.True(t, amIOwner)
	require.Equal(t, lock.userID, ownerID)

	t.Logf("My ID: %s", lock.userID)
	t.Logf("Owner ID: %s", ownerID)

	// Unlock the lock
	err = locker.Unlock(lock)
	require.NoError(t, err)

}

func TestUnlockLockNotTaken(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	err = locker.Unlock(lock)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnlockUnlocked)
}

func TestUnlockLockNotOwned(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock1, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	lock2, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	locker.Lock(lock1)

	err = locker.Unlock(lock2)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotOwner)

}

func TestMultipleLocks(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock1, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	lock2, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	locker.Lock(lock1)

	go func() {
		time.Sleep(2 * time.Second)
		locker.Unlock(lock1)
		t.Logf("Unlocked lock1")
	}()

	ch := make(chan bool)
	go func() {
		err := locker.Lock(lock2)
		require.NoError(t, err)
		ch <- true
	}()

	timer := time.NewTimer(100 * time.Millisecond)

loop:
	for {
		select {
		case <-ch:
			t.Logf("Lock2 acquired")
			break loop
		case <-timer.C:
			t.Logf("Lock2 not acquired")
			break
		}
	}

	err = locker.Unlock(lock2)
	require.NoError(t, err)

}

func TestIsLocked(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	// Check unlocked lock
	locked, err := locker.IsLocked(lock)
	require.NoError(t, err)
	require.False(t, locked)

	// Check locked lock
	locker.Lock(lock)

	locked, err = locker.IsLocked(lock)
	require.NoError(t, err)
	require.True(t, locked)

	// Check unlocked lock
	locker.Unlock(lock)

	locked, err = locker.IsLocked(lock)
	require.NoError(t, err)
	require.False(t, locked)

}

func TestIsLockedByMe(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	locker.Lock(lock)

	// Check if we are the owner
	locked, err := locker.IsLockedByMe(lock)
	require.NoError(t, err)
	require.True(t, locked)

	// Unlock the lock
	locker.Unlock(lock)

	// Should return false, when the lock is unlocked
	locked, err = locker.IsLockedByMe(lock)
	require.NoError(t, err)
	require.False(t, locked)

	// Check if we are not the owner
	lock2, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	locker.Lock(lock2)

	locked, err = locker.IsLockedByMe(lock)
	require.NoError(t, err)
	require.False(t, locked)

	locked, err = locker.IsLockedByMe(lock2)
	require.NoError(t, err)
	require.True(t, locked)
	locker.Unlock(lock2)

}

func TestTryLockUnlock(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	// Try to lock the lock
	locked, err := locker.TryLock(lock)
	require.NoError(t, err)
	require.True(t, locked)

	// Try to unlock the lock
	err = locker.Unlock(lock)
	require.NoError(t, err)

	// Introduce a new user and try to lock the lock
	lock2, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	locker.Lock(lock2)

	locked, err = locker.TryLock(lock)
	require.NoError(t, err)
	require.False(t, locked)

	locker.Unlock(lock2)

}

func TestLockWithTimeout(t *testing.T) {

	redis := setupRedis(t)

	locker := NewRedisLocker(WithRedisHost(redis.host), WithRedisPort(redis.port))

	lock, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	err = locker.LockWithTimeout(lock, 1*time.Second)
	require.NoError(t, err)

	lock2, err := locker.NewLock("testlock", "testgroup")
	require.NoError(t, err)

	err = locker.LockWithTimeout(lock2, 1*time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrLockTimeout)

	locker.Unlock(lock)

}
