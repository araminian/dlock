# dlock - Distributed Locking Library for Go

`dlock` is a distributed locking library implemented in Go using Redis as the backend. It provides a simple and reliable way to implement distributed locks in your applications.

## Features

- Simple and intuitive API
- Redis-backed distributed locking
- Multiple locking strategies (basic, timeout, retry with backoff)
- Group-based lock namespacing
- Unique user identification for lock ownership
- Thread-safe operations

## Installation

```bash
go get github.com/araminian/dlock
```

## Quick Start

```go
package main

import (
    "github.com/araminian/dlock"
    "fmt"
    "time"
)

func main() {
    // Create a new Redis locker
    locker := dlock.NewRedisLocker(
        dlock.WithRedisHost("localhost"),
        dlock.WithRedisPort("6379"),
    )

    // Create a new lock
    lock, err := locker.NewLock("my-resource", "my-group")
    if err != nil {
        panic(err)
    }

    // Basic lock operation
    if err := locker.Lock(lock); err != nil {
        panic(err)
    }
    defer locker.Unlock(lock)

    // Your critical section code here
    fmt.Println("Executing critical section...")
}
```

## Concurrent Lock Example

Here's an example demonstrating how two goroutines compete for the same lock:

```go
package main

import (
    "github.com/araminian/dlock"
    "fmt"
    "time"
)

func main() {
    locker := dlock.NewRedisLocker()

    // Create two locks with the same name and group
    lock1, _ := locker.NewLock("shared-resource", "group1")
    lock2, _ := locker.NewLock("shared-resource", "group1")

    // Goroutine 1
    go func() {
        fmt.Println("Goroutine 1: Attempting to acquire lock")
        err := locker.Lock(lock1)
        if err != nil {
            fmt.Println("Goroutine 1: Failed to acquire lock:", err)
            return
        }
        fmt.Println("Goroutine 1: Lock acquired")
        
        // Simulate some work
        time.Sleep(2 * time.Second)
        
        locker.Unlock(lock1)
        fmt.Println("Goroutine 1: Lock released")
    }()

    // Goroutine 2
    go func() {
        // Wait a bit before trying to acquire the lock
        time.Sleep(500 * time.Millisecond)
        
        fmt.Println("Goroutine 2: Attempting to acquire lock")
        err := locker.Lock(lock2)
        if err != nil {
            fmt.Println("Goroutine 2: Failed to acquire lock:", err)
            return
        }
        fmt.Println("Goroutine 2: Lock acquired")
        
        // Simulate some work
        time.Sleep(1 * time.Second)
        
        locker.Unlock(lock2)
        fmt.Println("Goroutine 2: Lock released")
    }()

    // Wait for both goroutines to complete
    time.Sleep(4 * time.Second)
}

// Output will be something like:
// Goroutine 1: Attempting to acquire lock
// Goroutine 1: Lock acquired
// Goroutine 2: Attempting to acquire lock
// Goroutine 1: Lock released
// Goroutine 2: Lock acquired
// Goroutine 2: Lock released
```

## API Reference

### Creating a Locker

```go
// Create with default settings (localhost:6379)
locker := dlock.NewRedisLocker()

// Create with custom Redis connection
locker := dlock.NewRedisLocker(
    dlock.WithRedisHost("redis.example.com"),
    dlock.WithRedisPort("6380"),
)
```

### Basic Lock Operations

```go
// Create a new lock
lock, err := locker.NewLock("resource-name", "group-name")
if err != nil {
    // Handle error
}

// Acquire lock (blocking)
err = locker.Lock(lock)

// Release lock
err = locker.Unlock(lock)

// Try to acquire lock (non-blocking)
acquired, err := locker.TryLock(lock)
if acquired {
    // Lock was successfully acquired
    defer locker.Unlock(lock)
}
```

### Advanced Locking Strategies

```go
// Lock with timeout
err = locker.LockWithTimeout(lock, 5*time.Second)

// Lock with retry and exponential backoff
err = locker.LockWithRetryBackoff(lock, 3, 100*time.Millisecond)
// This will retry 3 times with exponential backoff starting at 100ms
```

### Lock Status and Ownership

```go
// Check if a lock is taken
isLocked, err := locker.IsLocked(lock)

// Check if current user owns the lock
isOwner, err := locker.IsLockedByMe(lock)

// Get the current owner of the lock
exists, ownerID, err := locker.Owner(lock)
if exists {
    fmt.Printf("Lock is owned by: %s\n", ownerID)
}
```

## Error Handling

The library provides several error types for different scenarios:

```go
var (
    ErrLockerError           // Generic locker error
    ErrNotOwner             // Attempted to unlock a lock owned by another user
    ErrUnlockUnlocked       // Attempted to unlock an already unlocked lock
    ErrLockTimeout          // Lock acquisition timed out
    ErrRedisClientNotConnected // Cannot connect to Redis server
)
```

## Best Practices

1. Always use `defer locker.Unlock(lock)` after successfully acquiring a lock to ensure it's released
2. Use appropriate timeouts to prevent deadlocks
3. Implement proper error handling
4. Use meaningful group names to organize your locks
5. Consider using `TryLock` for non-blocking operations
6. Use `LockWithRetryBackoff` for resilient lock acquisition in distributed systems

## Thread Safety

The library is thread-safe and can be safely used in concurrent applications. Each lock operation is atomic and handled by Redis. 

Important: When using locks across different goroutines that need to access the same resource (same name and group), each goroutine should create its own lock object using `NewLock()`. Do not share the same lock object between goroutines. This is demonstrated in the concurrent example above where `lock1` and `lock2` are separate objects but reference the same resource.

## License

MIT License

Copyright (c) 2024 Armin Aminian 