package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"workbuddy-helper/internal/model"
	"workbuddy-helper/internal/secret"
)

// 账号备份文件：密码派生的 AES-256-GCM 加密 JSON，令牌不落明文。
// 导入时按 (版本, UID) 去重，只新增、不覆盖已有账号。
const (
	backupFormat     = "workbuddy-helper-accounts"
	backupVersion    = 1
	backupKDF        = "pbkdf2-sha256"
	backupCipherName = "aes-256-gcm"
	backupIterations = 120000
	backupSaltSize   = 16
	backupMinPass    = 6
	backupMaxIter    = 1000000
	maxAccounts      = 50
)

var errBackupCorrupt = userErr("密码错误或备份文件已损坏")

type userError struct{ message string }

func (e *userError) Error() string { return e.message }

func userErr(format string, args ...any) error {
	return &userError{message: fmt.Sprintf(format, args...)}
}

// IsUserError 标记可以直接回显给用户的操作错误（HTTP 400）。
func IsUserError(err error) bool {
	var target *userError
	return errors.As(err, &target)
}

type backupEnvelope struct {
	Format     string `json:"format"`
	Version    int    `json:"version"`
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	Cipher     string `json:"cipher"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Data       string `json:"data"`
}

type backupPayload struct {
	ExportedAt string          `json:"exportedAt"`
	Accounts   []backupAccount `json:"accounts"`
}

// backupAccount 只保留可迁移的账号信息：身份、令牌与展示名。
// 积分、今日状态等运行态数据不导出，导入后重新查询即可。
type backupAccount struct {
	UID          string `json:"uid"`
	EnterpriseID string `json:"enterpriseId,omitempty"`
	Nickname     string `json:"nickname,omitempty"`
	Alias        string `json:"alias,omitempty"`
	Domain       string `json:"domain,omitempty"`
	Version      string `json:"version"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"`
	Enabled      bool   `json:"enabled"`
}

// ExportAccounts 生成密码加密的账号备份，返回文件内容、建议文件名与账号数。
func (s *Service) ExportAccounts(password string) (content, filename string, count int, err error) {
	if utf8.RuneCountInString(password) < backupMinPass {
		return "", "", 0, userErr("导出密码至少 %d 位", backupMinPass)
	}
	s.mu.RLock()
	accounts := make([]backupAccount, 0, len(s.state.Accounts))
	for _, a := range s.state.Accounts {
		accounts = append(accounts, backupAccount{
			UID:          a.UID,
			EnterpriseID: a.EnterpriseID,
			Nickname:     a.Nickname,
			Alias:        a.Alias,
			Domain:       a.Domain,
			Version:      a.VersionMode(),
			AccessToken:  a.AccessToken,
			RefreshToken: a.RefreshToken,
			ExpiresAt:    a.ExpiresAt,
			Enabled:      a.Enabled,
		})
	}
	s.mu.RUnlock()
	if len(accounts) == 0 {
		return "", "", 0, userErr("当前没有可导出的账号")
	}

	now := time.Now().In(locChina)
	plain, merr := json.Marshal(backupPayload{
		ExportedAt: now.Format(time.RFC3339),
		Accounts:   accounts,
	})
	if merr != nil {
		return "", "", 0, merr
	}
	envelope, serr := sealBackup(plain, password)
	if serr != nil {
		return "", "", 0, serr
	}
	raw, merr := json.MarshalIndent(envelope, "", "  ")
	if merr != nil {
		return "", "", 0, merr
	}
	return string(raw), fmt.Sprintf("workbuddy-accounts-%s.json", now.Format("20060102-1504")), len(accounts), nil
}

// ImportAccounts 解密备份并合并账号，返回新增与跳过的数量。
// 已存在的 (版本, UID) 一律跳过，不覆盖本地令牌与别名。
func (s *Service) ImportAccounts(password, content string) (imported, skipped int, err error) {
	if strings.TrimSpace(content) == "" {
		return 0, 0, userErr("备份文件内容为空")
	}
	if utf8.RuneCountInString(password) < backupMinPass {
		return 0, 0, userErr("请输入导出时设置的密码")
	}
	plain, oerr := openBackup([]byte(content), password)
	if oerr != nil {
		return 0, 0, oerr
	}
	var payload backupPayload
	if uerr := json.Unmarshal(plain, &payload); uerr != nil {
		return 0, 0, userErr("备份内容格式不正确")
	}
	if len(payload.Accounts) == 0 {
		return 0, 0, userErr("备份中没有账号")
	}

	now := time.Now()
	// 与登录提交共用一把锁，避免同时登录/导入时重复插入同一 UID。
	s.loginCommitMu.Lock()
	defer s.loginCommitMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]bool, len(payload.Accounts))
	pending := make([]model.Account, 0, len(payload.Accounts))
	for _, item := range payload.Accounts {
		version := item.Version
		if version != model.ModeCN && version != model.ModeIntl {
			version = versionForDomain(item.Domain)
		}
		uid := strings.TrimSpace(item.UID)
		if uid == "" || strings.TrimSpace(item.AccessToken) == "" {
			skipped++
			continue
		}
		key := version + "\x00" + uid
		if seen[key] {
			skipped++
			continue
		}
		seen[key] = true
		duplicate := false
		for i := range s.state.Accounts {
			if s.state.Accounts[i].UID == uid && s.state.Accounts[i].VersionMode() == version {
				duplicate = true
				break
			}
		}
		if duplicate {
			skipped++
			continue
		}
		id, idErr := randomID(12)
		if idErr != nil {
			return 0, 0, idErr
		}
		status := "ready"
		if !item.Enabled {
			status = "disabled"
		}
		pending = append(pending, model.Account{
			ID:           id,
			UID:          uid,
			EnterpriseID: item.EnterpriseID,
			Nickname:     item.Nickname,
			Alias:        item.Alias,
			Domain:       item.Domain,
			Version:      version,
			AccessToken:  item.AccessToken,
			RefreshToken: item.RefreshToken,
			ExpiresAt:    item.ExpiresAt,
			Enabled:      item.Enabled,
			Status:       status,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	if len(pending) == 0 {
		return 0, skipped, nil
	}
	if len(s.state.Accounts)+len(pending) > maxAccounts {
		return 0, 0, userErr("导入后将超过 %d 个账号上限（当前 %d 个，待导入 %d 个）", maxAccounts, len(s.state.Accounts), len(pending))
	}

	before := cloneState(s.state)
	beforeLogs := append([]model.LogEntry(nil), s.logs...)
	s.state.Accounts = append(s.state.Accounts, pending...)
	s.appendLogLocked("info", "", fmt.Sprintf("导入账号：新增 %d 个，跳过 %d 个", len(pending), skipped))
	if saveErr := s.saveMutationLocked(before, beforeLogs); saveErr != nil {
		return 0, 0, saveErr
	}
	return len(pending), skipped, nil
}

func sealBackup(plain []byte, password string) (backupEnvelope, error) {
	salt := make([]byte, backupSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return backupEnvelope{}, err
	}
	key := secret.PBKDF2SHA256([]byte(password), salt, backupIterations, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return backupEnvelope{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return backupEnvelope{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return backupEnvelope{}, err
	}
	sealed := gcm.Seal(nil, nonce, plain, nil)
	return backupEnvelope{
		Format:     backupFormat,
		Version:    backupVersion,
		KDF:        backupKDF,
		Iterations: backupIterations,
		Cipher:     backupCipherName,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Data:       base64.StdEncoding.EncodeToString(sealed),
	}, nil
}

func openBackup(raw []byte, password string) ([]byte, error) {
	var envelope backupEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, userErr("不是有效的账号备份文件")
	}
	if envelope.Format != backupFormat {
		return nil, userErr("不是有效的账号备份文件")
	}
	if envelope.Version != backupVersion {
		return nil, userErr("备份文件版本 %d 不受支持，请升级程序", envelope.Version)
	}
	if envelope.KDF != backupKDF || envelope.Cipher != backupCipherName {
		return nil, userErr("备份文件使用了不支持的加密参数")
	}
	if envelope.Iterations < 10000 || envelope.Iterations > backupMaxIter {
		return nil, userErr("备份文件的加密参数异常")
	}
	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil || len(salt) < backupSaltSize {
		return nil, userErr("备份文件已损坏")
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, userErr("备份文件已损坏")
	}
	sealed, err := base64.StdEncoding.DecodeString(envelope.Data)
	if err != nil {
		return nil, userErr("备份文件已损坏")
	}
	key := secret.PBKDF2SHA256([]byte(password), salt, envelope.Iterations, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, userErr("备份文件已损坏")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, userErr("备份文件已损坏")
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, userErr("备份文件已损坏")
	}
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, errBackupCorrupt
	}
	return plain, nil
}
