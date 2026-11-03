package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"workbuddy-helper/internal/client"
	"workbuddy-helper/internal/model"
	"workbuddy-helper/internal/store"
)

func newBackupService(t *testing.T, accounts []model.Account) (*Service, *store.Store) {
	t.Helper()
	state := model.NewState()
	state.Accounts = accounts
	st := store.New(t.TempDir())
	if err := st.Save(state); err != nil {
		t.Fatal(err)
	}
	svc, err := New(st, client.New())
	if err != nil {
		t.Fatal(err)
	}
	return svc, st
}

func TestExportImportAccountsRoundTrip(t *testing.T) {
	now := time.Now()
	src, _ := newBackupService(t, []model.Account{
		{
			ID: "cn1", UID: "u-cn", Nickname: "国内账户", Alias: "主力号",
			Domain: "codebuddy.cn", Version: model.ModeCN,
			AccessToken: "access-cn", RefreshToken: "refresh-cn",
			ExpiresAt: now.Add(time.Hour).Unix(), Enabled: true,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "intl1", UID: "u-intl", Nickname: "Intl", Domain: "www.workbuddy.ai",
			Version: model.ModeIntl, AccessToken: "access-intl", RefreshToken: "refresh-intl",
			ExpiresAt: now.Add(time.Hour).Unix(), Enabled: false,
			CreatedAt: now, UpdatedAt: now,
		},
	})

	content, filename, count, err := src.ExportAccounts("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || !strings.HasSuffix(filename, ".json") {
		t.Fatalf("count=%d filename=%q", count, filename)
	}
	if strings.Contains(content, "access-cn") || strings.Contains(content, "refresh-cn") {
		t.Fatal("backup leaked plaintext tokens")
	}

	dst, dstStore := newBackupService(t, nil)
	imported, skipped, err := dst.ImportAccounts("secret123", content)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 2 || skipped != 0 {
		t.Fatalf("imported=%d skipped=%d", imported, skipped)
	}

	state, err := dstStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Accounts) != 2 {
		t.Fatalf("unexpected account count: %d", len(state.Accounts))
	}
	cn, err := dst.getAccount(state.Accounts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	var intl model.Account
	for _, a := range state.Accounts {
		if a.VersionMode() == model.ModeIntl {
			intl = a
		}
	}
	if cn.UID != "u-cn" || cn.Alias != "主力号" || cn.AccessToken != "access-cn" ||
		cn.RefreshToken != "refresh-cn" || cn.VersionMode() != model.ModeCN || !cn.Enabled {
		t.Fatalf("domestic account not restored: %+v", cn)
	}
	if intl.UID != "u-intl" || intl.AccessToken != "access-intl" || intl.VersionMode() != model.ModeIntl || intl.Enabled {
		t.Fatalf("international account not restored: %+v", intl)
	}
	if intl.Status != "disabled" {
		t.Fatalf("disabled account status not restored: %+v", intl)
	}
}

func TestImportAccountsRejectsWrongPassword(t *testing.T) {
	src, _ := newBackupService(t, []model.Account{{
		ID: "a1", UID: "u1", Version: model.ModeCN, AccessToken: "access",
		Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}})
	content, _, _, err := src.ExportAccounts("secret123")
	if err != nil {
		t.Fatal(err)
	}
	dst, _ := newBackupService(t, nil)
	_, _, err = dst.ImportAccounts("wrong-password", content)
	if err == nil || !IsUserError(err) {
		t.Fatalf("wrong password was not rejected with a user error: %v", err)
	}
}

func TestImportAccountsSkipsExisting(t *testing.T) {
	account := model.Account{
		ID: "a1", UID: "u1", Version: model.ModeCN, AccessToken: "access",
		Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	src, _ := newBackupService(t, []model.Account{account})
	content, _, _, err := src.ExportAccounts("secret123")
	if err != nil {
		t.Fatal(err)
	}

	dst, dstStore := newBackupService(t, []model.Account{{
		ID: "local", UID: "u1", Version: model.ModeCN, AccessToken: "local-token",
		Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}})
	imported, skipped, err := dst.ImportAccounts("secret123", content)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 0 || skipped != 1 {
		t.Fatalf("imported=%d skipped=%d", imported, skipped)
	}
	state, err := dstStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Accounts) != 1 || state.Accounts[0].AccessToken != "local-token" {
		t.Fatalf("existing account was overwritten: %+v", state.Accounts)
	}

	// 同一份备份导入两次，第二次全部跳过。
	imported, skipped, err = dst.ImportAccounts("secret123", content)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 0 || skipped != 1 {
		t.Fatalf("second import: imported=%d skipped=%d", imported, skipped)
	}
}

func TestExportAccountsRequiresPasswordAndData(t *testing.T) {
	src, _ := newBackupService(t, nil)
	if _, _, _, err := src.ExportAccounts("12345"); err == nil || !IsUserError(err) {
		t.Fatalf("short password was not rejected: %v", err)
	}
	if _, _, _, err := src.ExportAccounts("secret123"); err == nil || !IsUserError(err) {
		t.Fatalf("empty account list was not rejected: %v", err)
	}
}

func TestImportAccountsRejectsCorruptPayload(t *testing.T) {
	dst, _ := newBackupService(t, nil)
	if _, _, err := dst.ImportAccounts("secret123", `{"format":"nope"}`); err == nil || !IsUserError(err) {
		t.Fatalf("corrupt payload was not rejected: %v", err)
	}
	if _, _, err := dst.ImportAccounts("secret123", "not-json"); err == nil || !IsUserError(err) {
		t.Fatalf("invalid json was not rejected: %v", err)
	}
	var target *userError
	if !errors.As(userErr("x"), &target) {
		t.Fatal("userErr must produce a userError")
	}
}
