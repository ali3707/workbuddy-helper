package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSetupLoginAndPersistence(t *testing.T) {
	dir := t.TempDir()
	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Configured() {
		t.Fatal("fresh manager should not be configured")
	}
	if err := m.Setup("admin", "short"); err == nil {
		t.Fatal("short password should be rejected")
	}
	if err := m.Setup("admin", "strong-password"); err != nil {
		t.Fatal(err)
	}
	if err := m.Setup("other", "another-password"); !errors.Is(err, ErrAlreadySetup) {
		t.Fatalf("second setup must fail: %v", err)
	}

	if _, err := m.Login("admin", "wrong-password", "127.0.0.1"); !errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("wrong password accepted: %v", err)
	}
	token, err := m.Login("admin", "strong-password", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if user, ok := m.Authenticate(token); !ok || user != "admin" {
		t.Fatalf("session rejected: user=%q ok=%v", user, ok)
	}

	// 凭据落盘后重启仍可登录
	reloaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Configured() {
		t.Fatal("credentials were not persisted")
	}
	if _, err := reloaded.Login("admin", "strong-password", "10.0.0.1"); err != nil {
		t.Fatalf("login after reload failed: %v", err)
	}

	// 凭据文件不包含明文密码
	raw, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || strings.Contains(string(raw), "strong-password") {
		t.Fatal("credential file leaked plaintext password")
	}
}

func TestLoginRateLimit(t *testing.T) {
	m, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Setup("admin", "strong-password"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxFails; i++ {
		if _, err := m.Login("admin", "nope", "192.0.2.9"); !errors.Is(err, ErrInvalidLogin) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if _, err := m.Login("admin", "strong-password", "192.0.2.9"); !errors.Is(err, ErrTooManyFails) {
		t.Fatalf("rate limit not enforced: %v", err)
	}
	// 其他来源不受影响
	if _, err := m.Login("admin", "strong-password", "192.0.2.10"); err != nil {
		t.Fatalf("unrelated ip blocked: %v", err)
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	m, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Setup("admin", "strong-password"); err != nil {
		t.Fatal(err)
	}
	token, err := m.Login("admin", "strong-password", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	m.Logout(token)
	if _, ok := m.Authenticate(token); ok {
		t.Fatal("session survived logout")
	}
	if _, ok := m.Authenticate("bogus"); ok {
		t.Fatal("bogus token accepted")
	}
}

func TestCorruptCredentialFileFailsClosed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir); err == nil {
		t.Fatal("corrupt credentials must return an error")
	}
}

func TestSessionSlidingExpiry(t *testing.T) {
	m, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Setup("admin", "strong-password"); err != nil {
		t.Fatal(err)
	}
	token, err := m.Login("admin", "strong-password", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.sessions[token] = time.Now().Add(-time.Minute)
	m.mu.Unlock()
	if _, ok := m.Authenticate(token); ok {
		t.Fatal("expired session accepted")
	}
}
