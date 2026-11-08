package service

import (
	"encoding/json"
	"strings"
	"time"

	"workbuddy-helper/internal/model"
)

// workbuddy-switch（github.com/changexbc/workbuddy-switch）的账号库格式：
// 顶层明文 JSON 数组，凭据字段为 snake_case（access_token / refresh_token），
// expiresAt 为毫秒；同一记录还可能带 WorkBuddy 5.6 的 `{$wbEncrypted}` 加密信封
// （本机 DPAPI / keyblob 加密，helper 无法解密），这类记录按「无凭据」跳过。
//
// 这是 switch「导出为 JSON」的产物，用户把它交给 helper 的导入即可迁回。
type switchImportAccount struct {
	UID      string          `json:"uid"`
	Nickname json.RawMessage `json:"nickname"`
	Note     string          `json:"note"`
	// Variant: switch 的档位 cn / ai；缺失时按 domain 后缀兜底。
	Variant      string          `json:"variant"`
	EnterpriseID string          `json:"enterpriseId"`
	AccessToken  json.RawMessage `json:"access_token"`
	RefreshToken json.RawMessage `json:"refresh_token"`
	Domain       string          `json:"domain"`
	// expiresAt 为毫秒时间戳（switch 用它判断凭据是否过期）。
	ExpiresAt int64 `json:"expiresAt"`
	CreatedAt int64 `json:"createdAt"`
}

// importSwitchAccounts 导入 switch 的明文账号数组。
//
// 与加密备份导入共用同一套去重与落盘逻辑：按 (版本, UID) 跳过已存在账号，
// 不覆盖本地令牌与别名；缺少 UID 或明文令牌的记录计入跳过。
func (s *Service) importSwitchAccounts(content string) (imported, skipped int, err error) {
	var items []switchImportAccount
	if uerr := json.Unmarshal([]byte(content), &items); uerr != nil {
		return 0, 0, userErr("文件不是合法的 JSON 数组")
	}
	if len(items) == 0 {
		return 0, 0, userErr("文件里没有账号")
	}

	now := time.Now()
	// 与登录提交共用一把锁，避免同时登录/导入时重复插入同一 UID。
	s.loginCommitMu.Lock()
	defer s.loginCommitMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]bool, len(items))
	pending := make([]model.Account, 0, len(items))
	noCredential := 0
	for _, item := range items {
		uid := strings.TrimSpace(item.UID)
		token := plainSecret(item.AccessToken)
		if uid == "" || token == "" {
			skipped++
			noCredential++
			continue
		}
		version := switchAccountVersion(item.Variant, item.Domain)
		key := version + "\x00" + uid
		if seen[key] {
			skipped++
			continue
		}
		seen[key] = true
		if s.hasAccountLocked(uid, version) {
			skipped++
			continue
		}
		id, idErr := randomID(12)
		if idErr != nil {
			return 0, 0, idErr
		}
		domain := strings.TrimSpace(item.Domain)
		if domain == "" {
			domain = defaultDomainFor(version)
		}
		createdAt := now
		if item.CreatedAt > 0 {
			createdAt = time.UnixMilli(item.CreatedAt)
		}
		pending = append(pending, model.Account{
			ID:           id,
			UID:          uid,
			EnterpriseID: strings.TrimSpace(item.EnterpriseID),
			Nickname:     plainSecret(item.Nickname),
			// switch 的 note 就是账号备注，对应 helper 的 alias（展示名优先取 alias）。
			Alias:        strings.TrimSpace(item.Note),
			Domain:       domain,
			Version:      version,
			AccessToken:  token,
			RefreshToken: plainSecret(item.RefreshToken),
			ExpiresAt:    switchExpiresToUnix(item.ExpiresAt),
			Enabled:      true,
			Status:       "ready",
			CreatedAt:    createdAt,
			UpdatedAt:    now,
		})
	}
	if len(pending) == 0 && noCredential == len(items) {
		// 全都没有明文凭据：多半是 switch 采集到的加密信封，只能重新登录。
		return 0, 0, userErr("文件里没有可用的明文令牌：加密凭据无法导入，请用官方登录页重新登录该账号")
	}
	return s.saveImportedLocked(pending, skipped)
}

// plainSecret 从 JSON 值中取明文凭据：只有字符串可用；`{$wbEncrypted}` 信封
// 由 WorkBuddy 本机密钥加密，helper 无法解密，按「无凭据」处理。
func plainSecret(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

// switchAccountVersion 把 switch 的档位映射为 helper 的版本模式；
// 档位缺失或未知时按 domain 后缀判断（与采集口径一致）。
func switchAccountVersion(variant, domain string) string {
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "ai", "intl", "global":
		return model.ModeIntl
	case "cn", "china":
		return model.ModeCN
	}
	return versionForDomain(domain)
}

// defaultDomainFor 返回版本的默认登录域（switch 记录可能没有 domain）。
func defaultDomainFor(version string) string {
	if version == model.ModeIntl {
		return "www.workbuddy.ai"
	}
	return "www.workbuddy.cn"
}

// switchExpiresToUnix 把 switch 的毫秒过期时间换算为 helper 使用的 Unix 秒。
// 缺失或非正数返回 0（未知）；兼容被误存成秒的历史数据（阈值 1e11）。
func switchExpiresToUnix(ms int64) int64 {
	if ms <= 0 {
		return 0
	}
	if ms < 1e11 {
		return ms
	}
	return ms / 1000
}
