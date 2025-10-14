// Copyright (c) 2025, Kamaran Layne <kamaran@layne.dev>
// See LICENSE for licensing information

// Package state provides a thread-safe global key-value store for sharing data across different parts of an application.
// It uses a sync.RWMutex to ensure safe concurrent access to the internal map.
// The package exposes generic functions for getting and setting values, as well as utilities for deleting and clearing entries.
//
// Functions:
//   - Get[T any](key string) (value T, ok bool): Retrieves a value of type T by key, returning the value and a boolean indicating success.
//   - Set[T any](key string, value T): Stores a value of any type under the specified key.
//   - Delete(key string): Removes the entry associated with the given key.
//   - Clear(): Removes all entries from the state.
//
// Usage example:
//
//	state.Set("username", "alice")
//	username, ok := state.Get[string]("username")
//	state.Delete("username")
//	state.Clear()
package state

import (
	"sync"
)

var (
	mu   sync.RWMutex
	data = map[string]any{}
)

// Get retrieves a value of type T from the state using the provided key.
// It returns the value and a boolean indicating whether the key was found and the value could be asserted to type T.
// If the key does not exist or the value cannot be asserted to type T, the zero value of T and false are returned.
func Get[T any](key string) (value T, ok bool) {
	mu.RLock()
	defer mu.RUnlock()

	v, ok := data[key]
	if !ok {

		var zero T
		return zero, false
	}

	value, ok = v.(T)
	return
}

// Set stores a value of any type in the state map under the specified key.
// It is safe for concurrent use.
//
// Parameters:
//
//	key   - the string key under which the value will be stored
//	value - the value to store, of any type
func Set[T any](key string, value T) {
	mu.Lock()
	data[key] = value
	mu.Unlock()
}

// Delete removes the entry associated with the given key from the shared data map.
// It acquires a lock to ensure thread-safe access during the deletion.
func Delete(key string) {
	mu.Lock()
	delete(data, key)
	mu.Unlock()
}

// Clear resets the internal state by acquiring a lock and reinitializing the data map.
// This effectively removes all stored entries in a thread-safe manner.
func Clear() {
	mu.Lock()
	data = make(map[string]any)
	mu.Unlock()
}
