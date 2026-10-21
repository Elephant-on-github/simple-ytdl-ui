package auth

import (
	"testing"
	"time"
)

func TestWithoutAPasswordEverythingIsOpen(t *testing.T) {
	m := New("")

	if m.Required() {
		t.Error("Required() = true, want false without a password")
	}
	if !m.Valid("") {
		t.Error("Valid() = false, want true without a password")
	}
	if _, ok := m.Login("anything"); !ok {
		t.Error("Login() failed, want it to succeed without a password")
	}
}

func TestLoginWithTheRightPassword(t *testing.T) {
	m := New("hunter2")

	if !m.Required() {
		t.Error("Required() = false, want true")
	}
	token, ok := m.Login("hunter2")
	if !ok {
		t.Fatal("Login() failed with the right password")
	}
	if token == "" {
		t.Fatal("Login() returned no token")
	}
	if !m.Valid(token) {
		t.Error("Valid() = false for a fresh session")
	}
}

func TestLoginWithTheWrongPassword(t *testing.T) {
	m := New("hunter2")

	token, ok := m.Login("hunter3")
	if ok {
		t.Error("Login() succeeded with the wrong password")
	}
	if token != "" {
		t.Errorf("Login() returned a token %q, want none", token)
	}
	if m.Valid("") {
		t.Error("Valid() = true without a token")
	}
}

func TestLogoutClosesTheSession(t *testing.T) {
	m := New("hunter2")

	token, _ := m.Login("hunter2")
	m.Logout(token)

	if m.Valid(token) {
		t.Error("Valid() = true after the logout")
	}
}

func TestAnExpiredSessionIsNotValid(t *testing.T) {
	m := New("hunter2")
	// A clock the test can move.
	now := time.Now()
	m.now = func() time.Time { return now }

	token, _ := m.Login("hunter2")
	if !m.Valid(token) {
		t.Fatal("the fresh session is not valid")
	}

	now = now.Add(SessionTTL + time.Hour)
	if m.Valid(token) {
		t.Error("Valid() = true for an expired session")
	}
}
