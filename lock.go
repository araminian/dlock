package dlock

import (
	"sync"

	"github.com/google/uuid"
)

// Lock is a distributed lock, which is used to lock a resource.
type Lock struct {
	// ID is the unique identifier of the lock.
	id uuid.UUID
	// Name is the name of the lock.
	Name string
	// mu is the mutex of the lock, which is used for internal access control.
	mu *sync.Mutex
}

type Locker interface {
	// NewLock creates a new lock.
	NewLock(name string) (*Lock, error)
}
