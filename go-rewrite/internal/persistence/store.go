package persistence

import (
	"encoding/json"
	"os"
	"sync"
)

type Store[T any] struct {
	mu   sync.RWMutex
	path string
	data T
}

func NewStore[T any](path string, initial T) (*Store[T], error) {
	s := &Store[T]{
		path: path,
		data: initial,
	}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &s.data); err != nil {
			return s, nil
		}
	}
	return s, nil
}

func (s *Store[T]) Read() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *Store[T]) Update(fn func(*T) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.data); err != nil {
		return err
	}
	return s.flush()
}

func (s *Store[T]) flush() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store[T]) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flush()
}
