package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type ctxKey int

const (
	ctxKeyTraceID ctxKey = iota
)

// DB describes a simple typed key-value data store.
type DB[T any] interface {
	Load(key string) (T, bool)
	Store(key string, value T)
	Range(f func(string string, value T) bool)
	Delete(key any)
}

// memDB is a simple in-memory key-value store that is safe for concurrent use.
// It is essentially a typed wrapper for `sync.Map` that can persist to disk.
type memDB[T any] struct {
	sync.Map
	filename string
}

// NewDB creates a new instance of the DB for the given type. If a filename
// is given, it is used to persist data to disk.
func NewDB[T any](filename string) DB[T] {
	db := &memDB[T]{filename: filename}

	if filename != "" {
		// Load from disk
		f, err := os.Open(filename)
		if err == nil {
			defer f.Close()
			var items map[string]T
			if err := gob.NewDecoder(f).Decode(&items); err != nil {
				panic(err)
			}

			for k, v := range items {
				db.Store(k, v)
			}
		}
	}

	return db
}

// Load retrieves a value from the DB by key.
func (db *memDB[T]) Load(key string) (T, bool) {
	v, ok := db.Map.Load(key)
	if !ok {
		var t T
		return t, false
	}
	return v.(T), true
}

// Store sets a value in the DB by key.
func (db *memDB[T]) Store(key string, value T) {
	db.Map.Store(key, value)
	if db.filename != "" {
		// Persist to disk
		items := map[string]T{}
		db.Range(func(k string, v T) bool {
			items[k] = v
			return true
		})
		f, _ := os.Create(db.filename)
		_ = gob.NewEncoder(f).Encode(items)
		f.Close()
	}
}

// Range calls the given function for each key-value pair in the DB.
func (db *memDB[T]) Range(f func(key string, value T) bool) {
	db.Map.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(T))
	})
}

// Hash computes a SHA1 hash of the given value and returns it as a
// base64-encoded string. Fields are ordered and stable, based on the JSON
// marshaling field rules, so two values of the same type with the same field
// values will produce the same hash.
func Hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha1.Sum(b)
	return base64.URLEncoding.EncodeToString(h[:])
}

// GetTraceID generates a new random trace ID in the W3C TraceContext format,
// usable for distributed tracing via the `traceparent` header.
func GetTraceID() string {
	t := make([]byte, 24)
	_, _ = rand.Read(t)
	return fmt.Sprintf("00-%x-%x-00", t[:16], t[16:24])
}
