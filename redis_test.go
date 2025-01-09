package dlock

import (
	"context"
	"errors"
	"strconv"
	"testing"

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

	_, err := locker.NewLock("testlock")

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

	lock, err := locker.NewLock("testlock")
	require.NoError(t, err)

	require.NotNil(t, lock)
}
