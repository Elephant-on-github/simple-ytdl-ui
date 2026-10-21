// Package auth holds the optional password that guards the app.
//
// When a password is configured every API call needs a session, except the
// login itself and the health check. The session is a random token in a cookie;
// it lives in memory only, so a restart logs everybody out again.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"sync"
	"time"
)

// SessionTTL is how long a session stays valid.
const SessionTTL = 30 * 24 * time.Hour

// Manager keeps the password hash and the open sessions.
type Manager struct {
	mu       sync.Mutex
	hash     [sha256.Size]byte
	required bool
	sessions map[string]time.Time
	now      func() time.Time
}

// New returns a manager for the given password. An empty password means the
// app is open and no session is needed.
func New(password string) *Manager {
	m := &Manager{
		sessions: map[string]time.Time{},
		now:      time.Now,
	}
	if password != "" {
		m.required = true
		m.hash = sha256.Sum256([]byte(password))
	}
	return m
}

// Required reports whether a password is set.
func (m *Manager) Required() bool {
	return m.required
}

// Login checks the password and opens a session. It returns the session token.
func (m *Manager) Login(password string) (string, bool) {
	if !m.required {
		return "", true
	}
	given := sha256.Sum256([]byte(password))
	if subtle.ConstantTimeCompare(given[:], m.hash[:]) != 1 {
		return "", false
	}

	token := randomToken()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()
	m.sessions[token] = m.now().Add(SessionTTL)
	return token, true
}

// Valid reports whether the token belongs to an open session.
func (m *Manager) Valid(token string) bool {
	if !m.required {
		return true
	}
	if token == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()
	expiry, ok := m.sessions[token]
	return ok && m.now().Before(expiry)
}

// Logout closes the session of the given token.
func (m *Manager) Logout(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

// pruneLocked drops the sessions that have expired. The lock must be held.
func (m *Manager) pruneLocked() {
	now := m.now()
	for token, expiry := range m.sessions {
		if now.After(expiry) {
			delete(m.sessions, token)
		}
	}
}

func randomToken() string {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(buf[:])
}
