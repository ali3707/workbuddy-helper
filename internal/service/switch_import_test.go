package service

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"workbuddy-helper/internal/model"
)

func TestImportSwitchAccountsMapsFields(t *testing.T) {
	svc, _ := newBackupService(t, nil)
	expiryMS := time.Now().Add(2 * time.Hour).UnixMilli()
	content := `[
		{
			"uid": "u-cn",
			"nickname": "国内账户",
			"note": "主力号",
			"displayField": "note",
			"variant": "cn",
			"enterpriseId": "ent-1",
			"access_token": "access-cn",
			"refresh_token": "refresh-cn",
			"token_type": "Bearer",
			"domain": "www.workbuddy.cn",
			"expiresAt": ` + strconv.FormatInt(expiryMS, 10) + `,
			"createdAt": 1700000000000
		},
		{
			"uid": "u-intl",
			"nickname": "Intl",
			"variant": "ai",
			"access_token": "access-intl"
		}
	]`

	imported, skipped, err := svc.ImportAccounts("", content)
	if err != nil {
		t.Fatalf("导入 switch 数组失败：%v", err)
	}
	if imported != 2 || skipped != 0 {
		t.Fatalf("imported=%d skipped=%d, want 2/0", imported, skipped)
	}

	svc.mu.RLock()
	accounts := append([]model.Account(nil), svc.state.Accounts...)
	svc.mu.RUnlock()
	if len(accounts) != 2 {
		t.Fatalf("accounts=%d, want 2", len(accounts))
	}

	cn := accounts[0]
	if cn.UID != "u-cn" || cn.AccessToken != "access-cn" || cn.RefreshToken != "refresh-cn" {
		t.Fatalf("凭据字段映射错误：%+v", cn)
	}
	if cn.VersionMode() != model.ModeCN || cn.Domain != "www.workbuddy.cn" {
		t.Fatalf("版本/域名映射错误：version=%s domain=%s", cn.Version, cn.Domain)
	}
	if cn.Alias != "主力号" || cn.Nickname != "国内账户" {
		t.Fatalf("备注/昵称映射错误：alias=%q nickname=%q", cn.Alias, cn.Nickname)
	}
	if cn.EnterpriseID != "ent-1" {
		t.Fatalf("enterpriseId=%q", cn.EnterpriseID)
	}
	// switch 的 expiresAt 是毫秒，helper 内部用 Unix 秒。
	if cn.ExpiresAt != expiryMS/1000 {
		t.Fatalf("expiresAt=%d, want %d（秒）", cn.ExpiresAt, expiryMS/1000)
	}
	if !cn.Enabled || cn.Status != "ready" {
		t.Fatalf("导入账号应为启用可用状态：enabled=%v status=%s", cn.Enabled, cn.Status)
	}
	if cn.CreatedAt.UnixMilli() != 1700000000000 {
		t.Fatalf("createdAt 应保留原值：%s", cn.CreatedAt)
	}

	intl := accounts[1]
	if intl.VersionMode() != model.ModeIntl {
		t.Fatalf("variant=ai 应映射为国际版，得到 %s", intl.VersionMode())
	}
	if intl.Domain != "www.workbuddy.ai" {
		t.Fatalf("缺 domain 时应补国际版默认域，得到 %q", intl.Domain)
	}
	if intl.ExpiresAt != 0 {
		t.Fatalf("缺 expiresAt 时应为 0，得到 %d", intl.ExpiresAt)
	}
}

func TestImportSwitchAccountsSkipsEncryptedAndInvalid(t *testing.T) {
	svc, _ := newBackupService(t, nil)
	content := `[
		{"uid": "u-env", "access_token": {"$wbEncrypted": 1, "envelope": "x"}},
		{"uid": "", "access_token": "token"},
		{"uid": "u-no-token"}
	]`

	_, _, err := svc.ImportAccounts("", content)
	if err == nil {
		t.Fatal("没有可导入的明文凭据时应报错")
	}
	if !IsUserError(err) || !strings.Contains(err.Error(), "明文") {
		t.Fatalf("错误应提示明文凭据问题：%v", err)
	}
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	if len(svc.state.Accounts) != 0 {
		t.Fatalf("加密凭据不得入库：%+v", svc.state.Accounts)
	}
}

func TestImportSwitchAccountsSkipsExistingAccount(t *testing.T) {
	now := time.Now()
	svc, _ := newBackupService(t, []model.Account{{
		ID: "local", UID: "u1", Version: model.ModeCN, AccessToken: "local-token",
		Domain: "www.workbuddy.cn", Enabled: true, CreatedAt: now, UpdatedAt: now,
	}})

	imported, skipped, err := svc.ImportAccounts("", `[{"uid":"u1","variant":"cn","access_token":"new-token"}]`)
	if err != nil {
		t.Fatalf("已存在账号不应报错：%v", err)
	}
	if imported != 0 || skipped != 1 {
		t.Fatalf("imported=%d skipped=%d, want 0/1", imported, skipped)
	}
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	if svc.state.Accounts[0].AccessToken != "local-token" {
		t.Fatal("已存在账号的本地令牌不得被覆盖")
	}
}

// 顶层是数组时不应再要求密码（switch 的导入没有口令）。
func TestImportAccountsAcceptsArrayWithoutPassword(t *testing.T) {
	svc, _ := newBackupService(t, nil)
	imported, _, err := svc.ImportAccounts("", `[{"uid":"u2","access_token":"tok"}]`)
	if err != nil {
		t.Fatalf("明文数组不应要求密码：%v", err)
	}
	if imported != 1 {
		t.Fatalf("imported=%d, want 1", imported)
	}
}

// 导出给 switch 的文件应能被本程序再导入，字段不丢（双向往返）。
func TestSwitchExportImportRoundTrip(t *testing.T) {
	now := time.Now()
	expiry := now.Add(3 * time.Hour).Unix()
	src, _ := newBackupService(t, []model.Account{
		{
			ID: "cn", UID: "u-cn", Nickname: "国内账户", Alias: "主力号",
			EnterpriseID: "ent-cn", Domain: "www.workbuddy.cn", Version: model.ModeCN,
			AccessToken: "access-cn", RefreshToken: "refresh-cn", ExpiresAt: expiry,
			Enabled: true, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "intl", UID: "u-intl", Nickname: "Intl", Version: model.ModeIntl,
			Domain: "www.workbuddy.ai", AccessToken: "access-intl",
			Enabled: true, CreatedAt: now, UpdatedAt: now,
		},
	})

	content, _, count, err := src.ExportSwitchAccounts()
	if err != nil {
		t.Fatal(err)
	}
	dst, _ := newBackupService(t, nil)
	imported, skipped, err := dst.ImportAccounts("", content)
	if err != nil {
		t.Fatalf("往返导入失败：%v", err)
	}
	if imported != count || skipped != 0 {
		t.Fatalf("imported=%d skipped=%d count=%d", imported, skipped, count)
	}

	dst.mu.RLock()
	accounts := append([]model.Account(nil), dst.state.Accounts...)
	dst.mu.RUnlock()
	if len(accounts) != 2 {
		t.Fatalf("accounts=%d, want 2", len(accounts))
	}
	cn := accounts[0]
	if cn.UID != "u-cn" || cn.AccessToken != "access-cn" || cn.RefreshToken != "refresh-cn" {
		t.Fatalf("凭据往返丢失：%+v", cn)
	}
	if cn.VersionMode() != model.ModeCN || cn.Alias != "主力号" || cn.EnterpriseID != "ent-cn" {
		t.Fatalf("元数据往返丢失：%+v", cn)
	}
	if cn.ExpiresAt != expiry {
		t.Fatalf("expiresAt=%d, want %d（毫秒↔秒换算应可逆）", cn.ExpiresAt, expiry)
	}
	if accounts[1].VersionMode() != model.ModeIntl || accounts[1].AccessToken != "access-intl" {
		t.Fatalf("国际版账号往返丢失：%+v", accounts[1])
	}
}

// 对象（加密信封）仍走密码流程，缺密码时报「请输入导出时设置的密码」。
func TestImportAccountsStillRequiresPasswordForEnvelope(t *testing.T) {
	svc, _ := newBackupService(t, nil)
	_, _, err := svc.ImportAccounts("", `{"format":"workbuddy-helper-accounts","version":1}`)
	if err == nil || !strings.Contains(err.Error(), "密码") {
		t.Fatalf("加密备份缺密码时应提示输入密码：%v", err)
	}
}
