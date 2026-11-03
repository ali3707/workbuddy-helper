package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"workbuddy-helper/internal/secret"
)

const (
	// FileName 面板凭据文件名（位于 data 目录，仅保存盐与哈希）
	FileName = "auth.json"

	SessionTTL     = 7 * 24 * time.Hour
	minPasswordLen = 8
	maxPasswordLen = 256
	maxUsernameLen = 64
	iterations     = 120000
	saltSize       = 16
	keyLen         = 32

	failWindow   = 5 * time.Minute
	maxFails     = 5
	failCooldown = 5 * time.Minute
)

var (
	ErrAlreadySetup = errors.New("面板账号已初始化")
	ErrInvalidLogin = errors.New("用户名或密码不正确")
	ErrTooManyFails = errors.New("登录失败次数过多，请稍后再试")
	ErrNotSetup     = errors.New("尚未设置面板账号")
)

type record struct {
	Username   string    `json:"username"`
	Salt       string    `json:"salt"`
	Hash       string    `json:"hash"`
	Iterations int       `json:"iterations"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type failState struct {
	count int
	until time.Time
}

// Manager 管理面板账号与登录会话。会话只存内存，进程重启后需要重新登录。
type Manager struct {
	mu       sync.Mutex
	path     string
	rec      record
	sessions map[string]time.Time
	fails    map[string]*failState
}

func New(dir string) (*Manager, error) {
	m := &Manager{
		path:     filepath.Join(dir, FileName),
		sessions: make(map[string]time.Time),
		fails:    make(map[string]*failState),
	}
	raw, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取面板凭据失败: %w", err)
	}
	if err := json.Unmarshal(raw, &m.rec); err != nil {
		return nil, fmt.Errorf("面板凭据文件损坏: %w", err)
	}
	if strings.TrimSpace(m.rec.Username) == "" || m.rec.Salt == "" || m.rec.Hash == "" || m.rec.Iterations < 10000 {
		return nil, errors.New("面板凭据文件缺少必要字段")
	}
	return m, nil
}

func (m *Manager) Configured() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.configuredLocked()
}

func (m *Manager) configuredLocked() bool {
	return strings.TrimSpace(m.rec.Username) != "" && m.rec.Hash != ""
}

func (m *Manager) Username() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rec.Username
}

// Setup 首次设置面板账号；已初始化时返回 ErrAlreadySetup。
func (m *Manager) Setup(username, password string) error {
	next, err := buildRecord(username, password)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.configuredLocked() {
		return ErrAlreadySetup
	}
	if err := m.saveLocked(next); err != nil {
		return err
	}
	m.rec = next
	return nil
}

// SetCredentials 用环境变量引导面板账号（每次启动同步），不受已初始化限制。
// 密码变更后旧会话全部失效。
func (m *Manager) SetCredentials(username, password string) error {
	next, err := buildRecord(username, password)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.saveLocked(next); err != nil {
		return err
	}
	m.rec = next
	m.sessions = make(map[string]time.Time)
	return nil
}

func buildRecord(username, password string) (record, error) {
	username = strings.TrimSpace(username)
	if err := validateCredentials(username, password); err != nil {
		return record{}, err
	}
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return record{}, err
	}
	hash := secret.PBKDF2SHA256([]byte(password), salt, iterations, keyLen)
	return record{
		Username:   username,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Hash:       base64.StdEncoding.EncodeToString(hash),
		Iterations: iterations,
		UpdatedAt:  time.Now(),
	}, nil
}

// Login 校验口令并创建会话，返回会话令牌。
func (m *Manager) Login(username, password, clientIP string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.configuredLocked() {
		return "", ErrNotSetup
	}
	if until, blocked := m.blockedLocked(clientIP); blocked {
		_ = until
		return "", ErrTooManyFails
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(username)), []byte(m.rec.Username)) != 1 || !m.verifyLocked(password) {
		m.recordFailLocked(clientIP)
		return "", ErrInvalidLogin
	}
	delete(m.fails, clientIP)
	token, err := newToken()
	if err != nil {
		return "", err
	}
	m.cleanupLocked(time.Now())
	m.sessions[token] = time.Now().Add(SessionTTL)
	return token, nil
}

// Authenticate 校验会话令牌，返回登录用户名。
func (m *Manager) Authenticate(token string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if token == "" {
		return "", false
	}
	expires, ok := m.sessions[token]
	if !ok {
		return "", false
	}
	if time.Now().After(expires) {
		delete(m.sessions, token)
		return "", false
	}
	// 滑动续期：活跃会话每次访问延长有效期
	m.sessions[token] = time.Now().Add(SessionTTL)
	return m.rec.Username, true
}

func (m *Manager) Logout(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

func (m *Manager) verifyLocked(password string) bool {
	salt, err := base64.StdEncoding.DecodeString(m.rec.Salt)
	if err != nil {
		return false
	}
	want, err := base64.StdEncoding.DecodeString(m.rec.Hash)
	if err != nil {
		return false
	}
	iter := m.rec.Iterations
	if iter < 10000 {
		iter = iterations
	}
	got := secret.PBKDF2SHA256([]byte(password), salt, iter, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func (m *Manager) blockedLocked(ip string) (time.Time, bool) {
	state := m.fails[ip]
	if state == nil {
		return time.Time{}, false
	}
	if time.Now().After(state.until) {
		delete(m.fails, ip)
		return time.Time{}, false
	}
	return state.until, state.count >= maxFails
}

func (m *Manager) recordFailLocked(ip string) {
	state := m.fails[ip]
	if state == nil || time.Now().After(state.until) {
		state = &failState{until: time.Now().Add(failWindow)}
		m.fails[ip] = state
	}
	state.count++
	if state.count >= maxFails {
		state.until = time.Now().Add(failCooldown)
	}
}

func (m *Manager) cleanupLocked(now time.Time) {
	for token, expires := range m.sessions {
		if now.After(expires) {
			delete(m.sessions, token)
		}
	}
	for ip, state := range m.fails {
		if now.After(state.until) {
			delete(m.fails, ip)
		}
	}
}

func (m *Manager) saveLocked(next record) error {
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}

func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func validateCredentials(username, password string) error {
	if username == "" {
		return errors.New("用户名不能为空")
	}
	if utf8.RuneCountInString(username) > maxUsernameLen {
		return fmt.Errorf("用户名最多 %d 个字符", maxUsernameLen)
	}
	if utf8.RuneCountInString(password) < minPasswordLen {
		return fmt.Errorf("密码至少 %d 位", minPasswordLen)
	}
	if utf8.RuneCountInString(password) > maxPasswordLen {
		return fmt.Errorf("密码最多 %d 位", maxPasswordLen)
	}
	return nil
}
