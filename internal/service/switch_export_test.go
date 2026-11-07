package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"workbuddy-helper/internal/model"
)

func TestExportSwitchAccountsMapsFields(t *testing.T) {
	now := time.Now()
	cnExpiry := now.Add(time.Hour).Unix()
	svc, _ := newBackupService(t, []model.Account{
		{
			ID: "cn1", UID: "u-cn", Nickname: "国内账户", Alias: "主力号",
			EnterpriseID: "ent-cn", Domain: "www.codebuddy.cn", Version: model.ModeCN,
			AccessToken: "access-cn", RefreshToken: "refresh-cn",
			ExpiresAt: cnExpiry, Enabled: true,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			// 国际版且历史数据没有 domain：应补 www.workbuddy.ai，避免被判成国内版。
			ID: "intl1", UID: "u-intl", Nickname: "Intl",
			Version: model.ModeIntl, AccessToken: "access-intl",
			Enabled: false, CreatedAt: now, UpdatedAt: now,
		},
		{
			// 缺 access_token 的记录 switch 无法使用，导出时跳过。
			ID: "bad", UID: "u-bad", Version: model.ModeCN, CreatedAt: now, UpdatedAt: now,
		},
	})

	content, filename, count, err := svc.ExportSwitchAccounts()
	if err != nil {
		t.Fatalf("ExportSwitchAccounts: %v", err)
	}
	if count != 2 {
		t.Fatalf("count=%d, want 2", count)
	}
	if !strings.HasPrefix(filename, "wb-switch-accounts-") || !strings.HasSuffix(filename, ".json") {
		t.Fatalf("unexpected filename %q", filename)
	}

	var records []map[string]any
	if uerr := json.Unmarshal([]byte(content), &records); uerr != nil {
		t.Fatalf("导出内容应为 JSON 数组：%v（%s）", uerr, content)
	}
	if len(records) != 2 {
		t.Fatalf("records=%d, want 2", len(records))
	}

	cn := records[0]
	if cn["access_token"] != "access-cn" {
		t.Fatalf("access_token=%v, want access-cn（switch 只认 snake_case）", cn["access_token"])
	}
	if cn["refresh_token"] != "refresh-cn" {
		t.Fatalf("refresh_token=%v", cn["refresh_token"])
	}
	if cn["variant"] != "cn" {
		t.Fatalf("variant=%v, want cn", cn["variant"])
	}
	if cn["token_type"] != "Bearer" {
		t.Fatalf("token_type=%v", cn["token_type"])
	}
	if cn["domain"] != "www.codebuddy.cn" {
		t.Fatalf("domain=%v, want 原样保留", cn["domain"])
	}
	if cn["enterpriseId"] != "ent-cn" {
		t.Fatalf("enterpriseId=%v", cn["enterpriseId"])
	}
	if cn["note"] != "主力号" || cn["displayField"] != "note" {
		t.Fatalf("alias 应写入 note 并置 displayField=note：note=%v displayField=%v", cn["note"], cn["displayField"])
	}
	// switch 以毫秒判定凭据是否过期；helper 存的是秒。
	if got, ok := cn["expiresAt"].(float64); !ok || int64(got) != cnExpiry*1000 {
		t.Fatalf("expiresAt=%v, want %d（毫秒）", cn["expiresAt"], cnExpiry*1000)
	}

	intl := records[1]
	if intl["variant"] != "ai" {
		t.Fatalf("variant=%v, want ai", intl["variant"])
	}
	if intl["domain"] != "www.workbuddy.ai" {
		t.Fatalf("domain=%v, want 按国际版补默认域", intl["domain"])
	}
	if _, has := intl["expiresAt"]; has {
		t.Fatalf("过期时间未知时不得写入 expiresAt：%v", intl["expiresAt"])
	}
	if _, has := intl["note"]; has {
		t.Fatalf("alias 为空时不应写 note：%v", intl["note"])
	}
}

func TestExportSwitchAccountsDeduplicatesByUID(t *testing.T) {
	now := time.Now()
	svc, _ := newBackupService(t, []model.Account{
		{ID: "a", UID: "same", Version: model.ModeCN, AccessToken: "t1", CreatedAt: now, UpdatedAt: now},
		{ID: "b", UID: "same", Version: model.ModeIntl, AccessToken: "t2", CreatedAt: now, UpdatedAt: now},
	})

	content, _, count, err := svc.ExportSwitchAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d, want 1（switch 按 uid 合并）", count)
	}
	if !strings.Contains(content, "t1") {
		t.Fatalf("应保留首条记录：%s", content)
	}
}

func TestExportSwitchAccountsWithoutCredentials(t *testing.T) {
	svc, _ := newBackupService(t, nil)
	_, _, _, err := svc.ExportSwitchAccounts()
	if err == nil {
		t.Fatal("无账号时应返回错误")
	}
	if !IsUserError(err) {
		t.Fatalf("错误应可直接回显给用户：%v", err)
	}
}
