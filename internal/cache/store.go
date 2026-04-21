package cache

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("key not found")

type Store struct {
	mu    sync.RWMutex
	items map[string]string
}

func NewStore() *Store {
	return &Store{
		items: make(map[string]string),
	}
}

func (s *Store) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.items[key]
	if !ok {
		return "", ErrNotFound
	}

	return value, nil
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = value
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, key)
}
