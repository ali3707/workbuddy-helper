package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"workbuddy-helper/internal/model"
)

// WorkBuddy Switch（github.com/changexbc/workbuddy-switch）的账号库格式：
// 顶层是明文 JSON 数组，凭据字段用 snake_case（access_token / refresh_token），
// 元数据用 camelCase（variant / expiresAt / displayField），expiresAt 为毫秒。
//
// helper 默认导出的备份是「AES-256-GCM 加密信封 + camelCase accessToken + 秒级
// expiresAt」，switch 侧 export_import::parse_accounts_json 要求顶层 JSON 数组，
// 因此无法直接导入。这里单独产出一份 switch 可直接消费的明文文件：
// 不加密（switch 的导入没有密码输入）、字段对齐 switch 的
// merge_import_record / build_auth_obj 口径。
const (
	switchDomainCN  = "www.workbuddy.cn"
	switchDomainAI  = "www.workbuddy.ai"
	switchTokenType = "Bearer"
	switchVariantCN = "cn"
	switchVariantAI = "ai"
)

// switchAccountRecord 是 switch 账号库（~/.wb-switch/accounts.json）单条记录中
// helper 需要迁移的字段。积分、签到、旅行等运行态数据不迁移（switch 会重新拉取）。
type switchAccountRecord struct {
	UID      string `json:"uid,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	// Note / DisplayField 对应 switch 的「备注 + 展示字段」：helper 的 alias
	// （界面上的账户名称）在本地优先展示，迁移后保持同样的显示效果。
	Note         string `json:"note,omitempty"`
	DisplayField string `json:"displayField,omitempty"`
	// Variant：cn（国内版）/ ai（国际版）。缺失时 switch 会按 domain 后缀兜底，
	// 显式写入避免空 domain 的历史账号被误判成国内版。
	Variant      string `json:"variant"`
	EnterpriseID string `json:"enterpriseId,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	Domain       string `json:"domain,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"`
	CreatedAt    int64  `json:"createdAt,omitempty"`
}

// ExportSwitchAccounts 生成 workbuddy-switch 可直接导入的明文账号 JSON。
//
// 只跳过无法被 switch 使用的记录（缺 UID 或缺 access_token）：switch 的
// merge_import_record 同样会略过这类记录。备注 alias 写入 note 并置
// displayField=note，使 switch 切号后的展示名与 helper 一致。
func (s *Service) ExportSwitchAccounts() (content, filename string, count int, err error) {
	now := time.Now().In(locChina)
	nowMS := now.UnixMilli()

	s.mu.RLock()
	accounts := make([]switchAccountRecord, 0, len(s.state.Accounts))
	seen := make(map[string]bool, len(s.state.Accounts))
	for _, a := range s.state.Accounts {
		uid := strings.TrimSpace(a.UID)
		token := strings.TrimSpace(a.AccessToken)
		if uid == "" || token == "" {
			continue
		}
		// switch 导入按 uid 合并（不看档位）：同一 UID 出现多条会互相覆盖，
		// 这里先按 UID 去重，保留列表里第一条。
		if seen[uid] {
			continue
		}
		seen[uid] = true

		version := a.VersionMode()
		domain := strings.TrimSpace(a.Domain)
		if domain == "" {
			if version == model.ModeIntl {
				domain = switchDomainAI
			} else {
				domain = switchDomainCN
			}
		}
		record := switchAccountRecord{
			UID:          uid,
			Nickname:     a.Nickname,
			Variant:      switchVariant(version),
			EnterpriseID: a.EnterpriseID,
			AccessToken:  token,
			RefreshToken: strings.TrimSpace(a.RefreshToken),
			TokenType:    switchTokenType,
			Domain:       domain,
			ExpiresAt:    switchExpiresAt(a),
			CreatedAt:    nowMS,
		}
		if alias := strings.TrimSpace(a.Alias); alias != "" {
			record.Note = alias
			record.DisplayField = "note"
		}
		accounts = append(accounts, record)
	}
	s.mu.RUnlock()

	if len(accounts) == 0 {
		return "", "", 0, userErr("没有可导出的账号：请先登录至少一个账号")
	}
	raw, merr := json.MarshalIndent(accounts, "", "  ")
	if merr != nil {
		return "", "", 0, merr
	}
	return string(raw), fmt.Sprintf("wb-switch-accounts-%s.json", now.Format("2006-01-02")), len(accounts), nil
}

// switchVariant 把 helper 的版本模式映射为 switch 档位取值。
func switchVariant(version string) string {
	if version == model.ModeIntl {
		return switchVariantAI
	}
	return switchVariantCN
}

// switchExpiresAt 返回 switch 口径的 access token 过期时间（毫秒）。
//
// helper 的 ExpiresAt 是 Unix 秒（可能缺失或为 0），switch 侧
// account::has_unexpired_plain_token 用毫秒与 now_ms() 比较，所以这里统一
// 换算；过期时间未知时返回 0（字段省略），switch 会按「无 expiresAt 视为有效」处理。
func switchExpiresAt(a model.Account) int64 {
	seconds := a.ExpiresAt
	if exp, ok := model.JWTExpiryUnix(a.AccessToken); ok {
		seconds = exp
	}
	if seconds <= 0 {
		return 0
	}
	return seconds * 1000
}
