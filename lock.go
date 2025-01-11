package dlock

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrLockerError    = errors.New("locker error")
	ErrNotOwner       = errors.New("not the owner of the lock")
	ErrUnlockUnlocked = errors.New("unlocking an unlocked lock")
	ErrLockTimeout    = errors.New("lock timeout")
)

// Lock is a distributed lock, which is used to lock a resource.
type Lock struct {
	// userID is the unique identifier of the user who owns the lock.
	userID uuid.UUID
	// name is the name of the lock.
	name string
	// group is the group of the lock, which helps to have multiple locks with the same name,
	// but different groups.
	group string
}

type Locker interface {
	// NewLock creates a new lock.
	NewLock(name string, group string) (*Lock, error)

	// Lock locks the lock.
	Lock(lock *Lock) error

	// Unlock unlocks the lock.
	Unlock(lock *Lock) error

	// TryLock tries to lock the lock.
	TryLock(lock *Lock) (bool, error)

	// LockWithTimeout locks the lock with a timeout
	LockWithTimeout(lock *Lock, timeout time.Duration) error

	// Owner returns the owner of the lock.
	Owner(lock *Lock) (bool, uuid.UUID, error)

	// IsLockedByMe returns true if the lock is locked by the current user.
	IsLockedByMe(lock *Lock) (bool, error)

	// IsLocked returns true if the lock is locked.
	IsLocked(lock *Lock) (bool, error)
}
