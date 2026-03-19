// intenal/auth/jwk_store.go
package auth

import (
	"sync"

	"github.com/clerk/clerk-sdk-go/v2"
)

type JWKStore interface {
	GetJWK() *clerk.JSONWebKey
	SetJWK(*clerk.JSONWebKey)
	Clear()
}

type InMemoryJWKStore struct {
	mu  sync.RWMutex
	jwk *clerk.JSONWebKey
}

func NewJWKStore() JWKStore {
	return &InMemoryJWKStore{}
}

func (s *InMemoryJWKStore) GetJWK() *clerk.JSONWebKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jwk
}

func (s *InMemoryJWKStore) SetJWK(j *clerk.JSONWebKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jwk = j
}

func (s *InMemoryJWKStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jwk = nil
}