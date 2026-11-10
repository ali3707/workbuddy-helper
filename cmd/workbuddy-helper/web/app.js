(() => {
  "use strict";

  const appToken = String(
    window.__APP_TOKEN__ || document.querySelector('meta[name="app-token"]')?.getAttribute("content") || "",
  );

  const elements = {
    connectionStatus: document.querySelector("#connection-status"),
    connectionLabel: document.querySelector("#connection-label"),
    lastUpdated: document.querySelector("#last-updated"),
    summaryTotal: document.querySelector("#summary-total"),
    summaryChecked: document.querySelector("#summary-checked"),
    summaryCheckedLabel: document.querySelector("#summary-checked-label"),
    summaryPoints: document.querySelector("#summary-points"),
    summaryEnabled: document.querySelector("#summary-enabled"),
    accountCount: document.querySelector("#account-count"),
    accountRows: document.querySelector("#account-rows"),
    tableState: document.querySelector("#table-state"),
    search: document.querySelector("#account-search"),
    statusFilter: document.querySelector("#status-filter"),
    refreshState: document.querySelector("#refresh-state"),
    runAll: document.querySelector("#run-all"),
    runAllLabel: document.querySelector("#run-all-label"),
    runTravel: document.querySelector("#run-travel"),
    travelEnabled: document.querySelector("#travel-enabled"),
    travelAutoAdopt: document.querySelector("#travel-auto-adopt"),
    travelEnabledRow: document.querySelector("#travel-enabled-row"),
    travelAdoptRow: document.querySelector("#travel-adopt-row"),
    refreshPointsAll: document.querySelector("#refresh-points-all"),
    addAccount: document.querySelector("#add-account"),
    scheduleForm: document.querySelector("#schedule-form"),
    scheduleEnabled: document.querySelector("#schedule-enabled"),
    scheduleTime: document.querySelector("#schedule-time"),
    scheduleSummary: document.querySelector("#schedule-summary"),
    saveSchedule: document.querySelector("#save-schedule"),
    versionSwitch: document.querySelector("#version-switch"),
    proxyScheme: document.querySelector("#proxy-scheme"),
    proxyHost: document.querySelector("#proxy-host"),
    pageTitle: document.querySelector("#page-title"),
    pageBadge: document.querySelector("#page-badge"),
    pageSubtitle: document.querySelector("#page-subtitle"),
    activeModel: document.querySelector("#active-model"),
    activeModelRow: document.querySelector("#active-model-row"),
    activePrompt: document.querySelector("#active-prompt"),
    proxyRow: document.querySelector("#proxy-row"),
    activePromptRow: document.querySelector("#active-prompt-row"),
    recheckDelay: document.querySelector("#recheck-delay"),
    recheckDelayRow: document.querySelector("#recheck-delay-row"),
    colDaily: document.querySelector("#col-daily"),
    drawer: document.querySelector("#log-drawer"),
    drawerBackdrop: document.querySelector("#drawer-backdrop"),
    openLogs: document.querySelector("#open-logs"),
    closeLogs: document.querySelector("#close-logs"),
    refreshLogs: document.querySelector("#refresh-logs"),
    logList: document.querySelector("#log-list"),
    logSubtitle: document.querySelector("#log-subtitle"),
    authPage: document.querySelector("#auth-page"),
    authForm: document.querySelector("#auth-form"),
    authTitle: document.querySelector("#auth-title"),
    authSubtitle: document.querySelector("#auth-subtitle"),
    authUsername: document.querySelector("#auth-username"),
    authPassword: document.querySelector("#auth-password"),
    authConfirm: document.querySelector("#auth-confirm"),
    authConfirmLabel: document.querySelector("#auth-confirm-label"),
    authHint: document.querySelector("#auth-hint"),
    authSubmit: document.querySelector("#auth-submit"),
    userChip: document.querySelector("#user-chip"),
    userName: document.querySelector("#user-name"),
    logoutButton: document.querySelector("#logout-button"),
    loginPage: document.querySelector("#login-page"),
    cancelLogin: document.querySelector("#cancel-login"),
    dashboard: document.querySelector("#accounts"),
    loginStarting: document.querySelector("#login-starting"),
    loginReady: document.querySelector("#login-ready"),
    loginError: document.querySelector("#login-error"),
    loginErrorMessage: document.querySelector("#login-error-message"),
    loginSubtitle: document.querySelector("#login-subtitle"),
    loginVersionHint: document.querySelector("#login-version-hint"),
    loginVersionButtons: document.querySelectorAll("[data-login-version]"),
    loginUrlLabel: document.querySelector("#login-url-label"),
    loginFrame: document.querySelector("#login-frame"),
    openLoginLink: document.querySelector("#open-login-link"),
    copyLoginLink: document.querySelector("#copy-login-link"),
    loginPollStatus: document.querySelector("#login-poll-status"),
    retryLogin: document.querySelector("#retry-login"),
    backupDialog: document.querySelector("#backup-dialog"),
    backupTitle: document.querySelector("#backup-title"),
    backupSubtitle: document.querySelector("#backup-subtitle"),
    backupExportStage: document.querySelector("#backup-export-stage"),
    backupImportStage: document.querySelector("#backup-import-stage"),
    backupPassword: document.querySelector("#backup-password"),
    backupImportPassword: document.querySelector("#backup-import-password"),
    backupFile: document.querySelector("#backup-file"),
    backupPasswordRow: document.querySelector("#backup-password-row"),
    importFormatHint: document.querySelector("#import-format-hint"),
    confirmExport: document.querySelector("#confirm-export"),
    confirmImport: document.querySelector("#confirm-import"),
    exportAccounts: document.querySelector("#export-accounts"),
    importAccounts: document.querySelector("#import-accounts"),
    exportSwitch: document.querySelector("#export-switch"),
    switchDialog: document.querySelector("#switch-dialog"),
    confirmExportSwitch: document.querySelector("#confirm-export-switch"),
    editDialog: document.querySelector("#edit-dialog"),
    editForm: document.querySelector("#edit-form"),
    editAccountId: document.querySelector("#edit-account-id"),
    nicknameInput: document.querySelector("#nickname-input"),
    closeEdit: document.querySelector("#close-edit"),
    cancelEdit: document.querySelector("#cancel-edit"),
    saveNickname: document.querySelector("#save-nickname"),
    packagesDialog: document.querySelector("#packages-dialog"),
    packagesSubtitle: document.querySelector("#packages-subtitle"),
    packageList: document.querySelector("#package-list"),
    closePackages: document.querySelector("#close-packages"),
    toastRegion: document.querySelector("#toast-region"),
  };

  const app = {
    accounts: [],
    allAccounts: [],
    viewVersion: "cn",
    // 首次加载设置后按「默认版本」对齐默认查看列表，之后尊重用户手动切换
    viewInitialized: false,
    // 登录弹窗中选择的版本，决定登录链接与账号归属列表
    loginVersion: "cn",
    // 面板级登录状态：required=false 表示已用环境变量 Basic Auth 保护
    auth: { required: false, setupRequired: false, authenticated: true, username: "" },
    settings: {
      scheduleEnabled: false,
      scheduleTime: "09:15",
      mode: "cn",
      activeModel: "hy3",
      recheckDelayMinutes: 840,
      proxyUrl: "",
      activePrompt: "",
      travelEnabled: false,
      travelAutoAdopt: false,
    },
    isLoading: true,
    loadError: "",
    busyAccounts: new Set(),
    loginId: "",
    loginUrl: "",
    loginPollTimer: null,
    loginPollGeneration: 0,
    editingAccountId: "",
    packagesAccountId: "",
    proxyDirty: false,
    proxyRevision: 0,
    settingsDirty: false,
    settingsRevision: 0,
    stateLoadGeneration: 0,
    stateLoads: 0,
    tokenExpiryTimer: null,
  };

  const numberFormatter = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
  const integerFormatter = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 0 });
  const dateTimeFormatter = new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: "Asia/Shanghai",
  });
  const timeFormatter = new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZone: "Asia/Shanghai",
  });
  const tokenExpiryFormatter = new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: "Asia/Shanghai",
  });
  // 积分包到期时间：服务端给的是北京时间，展示精确到分（对齐 workbuddy-switch 的积分包窗口）
  const packageExpiryFormatter = new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: "Asia/Shanghai",
  });
  const logTimeFormatter = new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZone: "Asia/Shanghai",
  });
  const logTitleFormatter = new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZone: "Asia/Shanghai",
  });

  const logYearFormatter = new Intl.DateTimeFormat("en-CA", {
    year: "numeric",
    timeZone: "Asia/Shanghai",
  });

  function formatterParts(formatter, date) {
    const parts = {};
    for (const part of formatter.formatToParts(date)) {
      if (part.type !== "literal") parts[part.type] = part.value;
    }
    return parts;
  }

  // 当年日志只显示月-日，跨年日志自动补上年份，避免同日不同年混淆。
  const currentYearTag = formatterParts(logYearFormatter, new Date()).year;

  function refreshIcons(root = document) {
    if (window.lucide && typeof window.lucide.createIcons === "function") {
      window.lucide.createIcons({
        attrs: { "aria-hidden": "true" },
        root,
      });
    }
  }

  function escapeHTML(value) {
    return String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#039;");
  }

  function getErrorMessage(error, fallback = "操作失败，请稍后重试") {
    if (error instanceof Error && error.message) return error.message;
    if (typeof error === "string" && error.trim()) return error;
    return fallback;
  }

  async function api(path, options = {}) {
    const method = String(options.method || "GET").toUpperCase();
    const headers = new Headers(options.headers || {});
    headers.set("Accept", "application/json");

    if (method !== "GET" && method !== "HEAD") {
      headers.set("X-App-Token", appToken);
    }
    if (options.body !== undefined && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }

    const response = await fetch(path, { ...options, method, headers });
    const contentType = response.headers.get("content-type") || "";
    let payload = null;

    if (response.status !== 204) {
      if (contentType.includes("application/json")) {
        payload = await response.json().catch(() => null);
      } else {
        const text = await response.text();
        payload = text ? { raw: text } : null;
      }
    }

    if (response.status === 401 && !path.startsWith("/api/auth/")) {
      app.auth.authenticated = false;
      showAuthPage(false);
    }
    if (!response.ok) {
      const message =
        payload?.message ||
        payload?.error?.message ||
        (typeof payload?.error === "string" ? payload.error : "") ||
        payload?.raw ||
        `请求失败 (${response.status})`;
      const error = new Error(message);
      error.status = response.status;
      error.payload = payload;
      throw error;
    }

    return payload ?? {};
  }

  function firstDefined(...values) {
    return values.find((value) => value !== undefined && value !== null && value !== "");
  }

  function parseBoolean(value, fallback = false) {
    if (typeof value === "boolean") return value;
    if (typeof value === "number") return value !== 0;
    if (typeof value === "string") {
      if (["true", "1", "yes", "enabled", "active"].includes(value.toLowerCase())) return true;
      if (["false", "0", "no", "disabled", "inactive"].includes(value.toLowerCase())) return false;
    }
    return fallback;
  }

  function parseNumber(value) {
    if (typeof value === "number" && Number.isFinite(value)) return value;
    if (typeof value === "string" && value.trim()) {
      const parsed = Number(value.replaceAll(",", ""));
      if (Number.isFinite(parsed)) return parsed;
    }
    return null;
  }

  // API 持久化 Unix 秒；同时兼容旧客户端可能返回的毫秒或 ISO 时间。
  function normalizeTimestamp(value) {
    if (value instanceof Date) {
      const timestamp = value.getTime();
      return Number.isFinite(timestamp) ? timestamp : null;
    }
    if (typeof value === "number" && Number.isFinite(value)) {
      return Math.abs(value) < 1e12 ? value * 1000 : value;
    }
    if (typeof value !== "string" || !value.trim()) return null;
    const text = value.trim();
    if (/^-?\d+(?:\.\d+)?$/.test(text)) {
      const numeric = Number(text);
      if (Number.isFinite(numeric)) return Math.abs(numeric) < 1e12 ? numeric * 1000 : numeric;
    }
    const parsed = Date.parse(text);
    return Number.isFinite(parsed) ? parsed : null;
  }

  function normalizeTime(value) {
    if (typeof value !== "string") return "09:15";
    const match = value.match(/^(\d{1,2}):(\d{2})/);
    if (!match) return "09:15";
    const hour = Math.min(23, Math.max(0, Number(match[1])));
    const minute = Math.min(59, Math.max(0, Number(match[2])));
    return `${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}`;
  }

  function normalizeAccount(raw, index) {
    const id = String(firstDefined(raw?.id, raw?.uid, raw?.accountId, raw?.userId, index));
    const displayName = String(firstDefined(raw?.displayName, raw?.alias, raw?.nickname, raw?.name, ""));
    const nickname = String(firstDefined(raw?.alias, raw?.displayName, raw?.nickname, raw?.name, ""));
    const identity = String(
      firstDefined(raw?.email, raw?.phone, raw?.username, raw?.uid, raw?.userId, raw?.id, `账户 ${index + 1}`),
    );
    const enabled = raw?.enabled !== undefined ? parseBoolean(raw.enabled, true) : !parseBoolean(raw?.disabled, false);
    const points = parseNumber(
      firstDefined(
        raw?.points,
        raw?.credits,
      raw?.credit,
      raw?.balance,
        raw?.score,
        raw?.quota?.points,
        raw?.point?.total,
      ),
    );
    const checkinRaw = firstDefined(
      raw?.checkinStatus,
      raw?.checkInStatus,
      raw?.signinStatus,
      raw?.signInStatus,
      raw?.lastCheckin,
      raw?.today?.status,
      raw?.checkin?.status,
    );
    const checkedFlag = firstDefined(
      raw?.checkedInToday,
      raw?.checkedIn,
      raw?.signedToday,
      raw?.isCheckedIn,
      raw?.today?.checked,
      raw?.checkin?.checked,
    );
    const healthRaw = String(firstDefined(raw?.status, raw?.health, raw?.tokenStatus, "")).toLowerCase();
    const reason = String(firstDefined(raw?.reason, raw?.lastError, raw?.error, raw?.message, raw?.statusMessage, ""));
    const tokenExpiresAt = normalizeTimestamp(
      firstDefined(
        raw?.tokenExpiresAt,
        // 兼容早期候选接口字段；正式接口使用 tokenExpiresAt。
        raw?.accessTokenExpiresAt,
        raw?.expiresAt,
        raw?.jwtExpiresAt,
      ),
    );

    const rawDomain = String(raw?.domain || "").trim().toLowerCase();
    const inferredVersion = raw?.region === "global" || rawDomain === "workbuddy.ai" || rawDomain.endsWith(".workbuddy.ai")
      ? "intl"
      : "cn";

    return {
      ...raw,
      id,
      version: String(firstDefined(raw?.version, inferredVersion, "cn")),
      displayName,
      nickname,
      identity,
      enabled,
      points,
      checkinRaw,
      checkedFlag: checkedFlag === undefined ? inferCheckedToday(raw, checkinRaw) : checkedFlag,
      healthRaw,
      reason,
      tokenExpiresAt,
      updatedAt: firstDefined(
        raw?.pointsUpdatedAt,
        raw?.creditUpdatedAt,
        raw?.lastPointsAt,
        raw?.updatedAt,
        raw?.lastUpdated,
      ),
    };
  }

  function localDateKey(date = new Date()) {
    const parts = new Intl.DateTimeFormat("en-CA", {
      timeZone: "Asia/Shanghai",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).formatToParts(date);
    const values = Object.fromEntries(parts.map((part) => [part.type, part.value]));
    return `${values.year}-${values.month}-${values.day}`;
  }

  function inferCheckedToday(raw, checkinRaw) {
    const state = String(checkinRaw ?? "").toLowerCase();
    if (!["confirmed", "already_done", "success", "done", "checked", "checked_in", "signed", "signed_in", "ok", "completed"].includes(state)) {
      return undefined;
    }
    const dateValue = firstDefined(raw?.lastCheckinDate, raw?.checkinDate, raw?.today?.date);
    if (!dateValue) return ["confirmed", "already_done"].includes(state);
    const dateText = String(dateValue).slice(0, 10);
    return dateText === localDateKey();
  }

  function normalizeState(payload) {
    const source = payload?.data && typeof payload.data === "object" ? payload.data : payload || {};
    const rawAccounts = Array.isArray(source) ? source : source.accounts || source.items || [];
    const rawSettings = source.settings || source.schedule || payload?.settings || {};

    const cnList = Array.isArray(source.cnAccounts) ? source.cnAccounts.map(normalizeAccount) : null;
    const intlList = Array.isArray(source.intlAccounts) ? source.intlAccounts.map(normalizeAccount) : null;
    const fallback = Array.isArray(rawAccounts) ? rawAccounts.map(normalizeAccount) : [];
    return {
      // 两套账号列表：国内版 / 国际版独立管理，按当前查看版本取对应列表
      cnAccounts: cnList || fallback.filter((a) => (a.version || "cn") !== "intl"),
      intlAccounts: intlList || fallback.filter((a) => a.version === "intl"),
      settings: {
        scheduleEnabled: parseBoolean(
          firstDefined(rawSettings.scheduleEnabled, rawSettings.enabled, source.scheduleEnabled),
          false,
        ),
        scheduleTime: normalizeTime(firstDefined(rawSettings.scheduleTime, rawSettings.time, source.scheduleTime, "09:15")),
        mode: String(firstDefined(rawSettings.mode, source.mode, "cn")) === "intl" ? "intl" : "cn",
        activeModel: ["hy3", "hy4-preview"].includes(String(rawSettings.activeModel))
          ? String(rawSettings.activeModel)
          : "hy3",
        recheckDelayMinutes: clampRecheckDelay(parseNumber(rawSettings.recheckDelayMinutes) ?? 840),
        proxyUrl: String(firstDefined(rawSettings.proxyUrl, source.proxyUrl, "") || ""),
        activePrompt: String(firstDefined(rawSettings.activePrompt, source.activePrompt, "") || ""),
        travelEnabled: parseBoolean(firstDefined(rawSettings.travelEnabled, source.travelEnabled), false),
        travelAutoAdopt: parseBoolean(firstDefined(rawSettings.travelAutoAdopt, source.travelAutoAdopt), false),
      },
    };
  }

  // 代理地址拆分：socks5://user:pass@host:port -> { scheme, host, auth }
  function parseProxyURL(value) {
    const raw = String(value || "").trim();
    if (!raw) return { scheme: "", host: "" };
    const m = raw.match(/^(https?|socks5h?|):\/\/(.+)$/i);
    if (!m) return { scheme: "", host: raw };
    return { scheme: m[1].toLowerCase(), host: m[2] };
  }

  function joinProxyURL(scheme, host) {
    scheme = String(scheme || "").trim().toLowerCase();
    host = String(host || "").trim();
    if (!scheme) return "";
    return host ? `${scheme}://${host}` : "";
  }

  function syncProxyInput({ clearDirect = false } = {}) {
    const direct = !elements.proxyScheme.value;
    if (direct && clearDirect) elements.proxyHost.value = "";
    elements.proxyHost.disabled = direct;
    elements.proxyHost.placeholder = direct
      ? "直连模式不需要代理地址"
      : "host:port（支持认证 user:pass@host:port）";
  }

  function clampRecheckDelay(value) {
    if (!Number.isFinite(value)) return 840;
    return Math.min(1440, Math.max(60, Math.round(value)));
  }

  function getCheckinState(account) {
    if (typeof account.checkedFlag === "boolean") {
      return account.checkedFlag
        ? { key: "done", label: "已签到", className: "checkin-done" }
        : { key: "pending", label: "未签到", className: "checkin-pending" };
    }

    const value = String(account.checkinRaw ?? "").toLowerCase();
    if (["confirmed", "already_done", "success", "done", "checked", "checked_in", "signed", "signed_in", "ok", "completed"].includes(value)) {
      return { key: "done", label: "已签到", className: "checkin-done" };
    }
    if (["failed", "error", "expired"].includes(value)) {
      return { key: "error", label: "签到失败", className: "checkin-error" };
    }
    if (["pending", "not_checked", "unsigned", "not_signed", "ready"].includes(value)) {
      return { key: "pending", label: "未签到", className: "checkin-pending" };
    }
    return { key: "unknown", label: "暂无记录", className: "checkin-unknown" };
  }

  // A completed chat is not proof that delayed activity credits arrived.
  function getActiveState(account) {
    const state = String(account.lastActive ?? "").toLowerCase();
    const isToday = account.lastActiveDate === localDateKey();
    if (isToday && state === "credited") {
      return { key: "done", label: "积分已到账", className: "checkin-done" };
    }
    if (isToday && state === "confirmed") {
      const label = account.pendingRecheckAt
        ? "已发送待复核"
        : (Number(account.recheckAttempts) >= 6 ? "复核未到账" : "已发送未确认");
      return { key: "done", label, className: "checkin-pending" };
    }
    if (["failed", "error"].includes(state)) {
      return { key: "error", label: "活跃失败", className: "checkin-error" };
    }
    return { key: "pending", label: "未活跃", className: "checkin-pending" };
  }

  // 今日任务状态：国内版列表取签到，国际版列表取活跃（跟随当前查看的版本）
  function getDailyState(account) {
    return app.viewVersion === "intl" ? getActiveState(account) : getCheckinState(account);
  }

  // 积分复核倒计时提示（国际版活跃后积分延迟到账）
  function formatRecheckMeta(account) {
    if (!account.pendingRecheckAt) return "";
    const due = new Date(account.pendingRecheckAt);
    if (Number.isNaN(due.getTime())) return "";
    const attempts = Number(account.recheckAttempts) || 0;
    const suffix = attempts > 0 ? ` · 第 ${attempts + 1} 次复核` : "";
    return `等待积分到账 ${timeFormatter.format(due)}${suffix}`;
  }

  function getHealthState(account) {
    if (!account.enabled) return { key: "disabled", label: "已停用", className: "status-muted" };
    if (account.cooling || ["cooling", "rate_limited", "limited"].includes(account.healthRaw)) {
      return { key: "attention", label: "冷却中", className: "status-warn" };
    }
    if (["needs_login", "unauthorized", "expired", "auth_dead"].includes(account.healthRaw)) {
      return { key: "attention", label: "登录失效", className: "status-error" };
    }
    if (["credits_exhausted"].includes(account.healthRaw)) {
      return { key: "attention", label: "积分耗尽", className: "status-warn" };
    }
    if (["running", "pending"].includes(account.healthRaw)) {
      return { key: "attention", label: "执行中", className: "status-warn" };
    }
    if (["ready", "confirmed", "already_done", "ok", "success", "done"].includes(account.healthRaw)) {
      return { key: "enabled", label: "正常", className: "status-ok" };
    }
    if (
      account.reason ||
      parseBoolean(account.expired, false) ||
      ["error", "failed", "expired", "invalid", "unauthorized"].includes(account.healthRaw)
    ) {
      return { key: "attention", label: account.healthRaw === "expired" ? "登录失效" : "需处理", className: "status-error" };
    }
    return { key: "enabled", label: "正常", className: "status-ok" };
  }

  function accountDisplayName(account) {
    return account.displayName || account.nickname || account.identity || `账户 ${account.id}`;
  }

  function accountInitial(account) {
    const value = accountDisplayName(account).trim();
    if (!value) return "W";
    return Array.from(value)[0].toUpperCase();
  }

  function formatPoints(value) {
    return value === null ? "--" : numberFormatter.format(value);
  }

  function formatUpdatedAt(value) {
    if (!value) return "尚未查询";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return String(value);
    return `${dateTimeFormatter.format(date)} 更新`;
  }

  function formatTokenDuration(milliseconds) {
    let seconds = Math.max(1, Math.ceil(milliseconds / 1000));
    const days = Math.floor(seconds / 86400);
    seconds %= 86400;
    const hours = Math.floor(seconds / 3600);
    seconds %= 3600;
    const minutes = Math.floor(seconds / 60);
    seconds %= 60;
    if (days > 0) return `${days}天${hours ? ` ${hours}小时` : ""}`;
    if (hours > 0) return `${hours}小时${minutes ? ` ${minutes}分` : ""}`;
    if (minutes > 0) return `${minutes}分`;
    return `${seconds}秒`;
  }

  function formatTokenExpiry(account, now = Date.now()) {
    const timestamp = Number(account.tokenExpiresAt);
    if (!Number.isFinite(timestamp) || timestamp <= 0) {
      return {
        label: "JWT 有效期未知",
        title: "服务端未提供 JWT 令牌到期时间",
        className: "token-expiry-unknown",
      };
    }

    const expiry = new Date(timestamp);
    if (Number.isNaN(expiry.getTime())) {
      return {
        label: "JWT 有效期未知",
        title: "JWT 令牌到期时间格式无效",
        className: "token-expiry-unknown",
      };
    }

    const remaining = timestamp - now;
    const exact = tokenExpiryFormatter.format(expiry);
    if (remaining <= 0) {
      return {
        label: "JWT 已过期",
        title: `JWT 已于 ${exact} 过期`,
        className: "token-expiry-expired",
      };
    }

    const duration = formatTokenDuration(remaining);
    return {
      label: `JWT 剩余 ${duration}`,
      title: `JWT 有效至 ${exact}（剩余 ${duration}）`,
      className: remaining <= 24 * 60 * 60 * 1000 ? "token-expiry-warning" : "token-expiry-ok",
    };
  }

  // 旅行状态文案：traveling/arrived/done/error/未旅行
  function travelInfo(account) {
    const state = String(account.travelState || "");
    const place = String(account.travelLocation || "").trim();
    const reward = Math.trunc(Number(account.travelReward) || 0);
    const arriveMs = Number(account.travelArriveAt) > 0 ? Number(account.travelArriveAt) * 1000 : 0;
    const message = String(account.travelMessage || "").trim();
    const join = (parts) => parts.filter(Boolean).join(" · ");
    if (state === "traveling") {
      const base = join([place, reward > 0 ? `预计 +${reward}` : "旅行中"]);
      return { base, arriveMs, className: "travel-active", title: message || base, text: base };
    }
    if (state === "arrived") {
      const base = join([place, reward > 0 ? `预计 +${reward}` : "已到达", "待领取"]);
      return { base, arriveMs: 0, className: "travel-arrived", title: message || base, text: base };
    }
    if (state === "done") {
      const base = join([place, reward > 0 ? `+${reward}` : "已完成"]);
      return { base: base || "旅行已完成", arriveMs: 0, className: "travel-done", title: message || base, text: base || "旅行已完成" };
    }
    if (state === "error" || state === "no_buddy") {
      const text = state === "no_buddy" ? "无 Buddy" : "旅行失败";
      return { base: text, arriveMs: 0, className: "travel-error", title: message || text, text };
    }
    return { base: "未旅行", arriveMs: 0, className: "travel-idle", title: "未旅行", text: "未旅行" };
  }

  function travelRemaining(arriveMs, now = Date.now()) {
    if (!arriveMs || arriveMs <= 0) return "";
    const diff = arriveMs - now;
    if (diff <= 0) return "即将到达";
    const totalMinutes = Math.max(1, Math.ceil(diff / 60000));
    const hours = Math.floor(totalMinutes / 60);
    const minutes = totalMinutes % 60;
    if (hours > 0 && minutes > 0) return `剩余 ${hours} 小时 ${minutes} 分钟`;
    if (hours > 0) return `剩余 ${hours} 小时`;
    return `剩余 ${minutes} 分钟`;
  }

  function refreshTravelMeta() {
    document.querySelectorAll("[data-travel-arrive]").forEach((node) => {
      const arriveMs = Number(node.dataset.travelArrive) || 0;
      const base = node.dataset.travelBase || "";
      const remaining = travelRemaining(arriveMs);
      node.textContent = remaining ? `${base} · ${remaining}` : base;
    });
  }

  function renderSummary() {
    const intl = app.viewVersion === "intl";
    const enabled = app.accounts.filter((account) => account.enabled).length;
    const checked = app.accounts.filter((account) => getDailyState(account).key === "done").length;
    const pointsValues = app.accounts.map((account) => account.points).filter((value) => value !== null);
    const pointsTotal = pointsValues.reduce((sum, value) => sum + value, 0);

    elements.summaryTotal.textContent = integerFormatter.format(app.accounts.length);
    elements.summaryChecked.textContent = integerFormatter.format(checked);
    elements.summaryCheckedLabel.textContent = intl ? "今日已活跃" : "今日已签到";
    elements.summaryEnabled.textContent = integerFormatter.format(enabled);
    elements.summaryPoints.textContent = pointsValues.length ? numberFormatter.format(pointsTotal) : "--";
  }

  function accountMatchesFilter(account) {
    const query = elements.search.value.trim().toLocaleLowerCase("zh-CN");
    const filter = elements.statusFilter.value;
    const health = getHealthState(account);
    const haystack = `${account.nickname} ${account.identity} ${account.id}`.toLocaleLowerCase("zh-CN");

    if (query && !haystack.includes(query)) return false;
    if (filter === "all") return true;
    if (filter === "enabled") return account.enabled;
    if (filter === "disabled") return !account.enabled;
    if (filter === "attention") return health.key === "attention";
    return true;
  }

  function renderTableState(kind, title, detail, actionLabel = "") {
    const icon = kind === "error" ? "wifi-off" : kind === "empty" ? "users" : kind === "search" ? "search-x" : "loader-circle";
    const action = actionLabel
      ? `<button class="button button-secondary" type="button" data-state-action="${kind}">
          <i data-lucide="${kind === "error" ? "rotate-cw" : "user-plus"}" aria-hidden="true"></i>
          <span>${escapeHTML(actionLabel)}</span>
        </button>`
      : "";
    elements.tableState.innerHTML = `
      ${kind === "loading" ? '<div class="loading-spinner" aria-hidden="true"></div>' : `<div class="table-state-icon"><i data-lucide="${icon}" aria-hidden="true"></i></div>`}
      <strong>${escapeHTML(title)}</strong>
      <span>${escapeHTML(detail)}</span>
      ${action}
    `;
    elements.tableState.hidden = false;
    refreshIcons(elements.tableState);
  }

  function renderAccounts() {
    if (app.isLoading) {
      elements.accountRows.innerHTML = "";
      renderTableState("loading", "正在读取账户", "请稍候");
      elements.accountCount.textContent = "正在同步";
      return;
    }

    if (app.loadError && !app.accounts.length) {
      elements.accountRows.innerHTML = "";
      renderTableState("error", "无法读取账户", app.loadError, "重新加载");
      elements.accountCount.textContent = "连接失败";
      return;
    }

    const filtered = app.accounts.filter(accountMatchesFilter);
    elements.accountCount.textContent = filtered.length === app.accounts.length
      ? `${integerFormatter.format(app.accounts.length)} 个账户`
      : `显示 ${integerFormatter.format(filtered.length)} / ${integerFormatter.format(app.accounts.length)}`;

    if (!app.accounts.length) {
      elements.accountRows.innerHTML = "";
      const intl = app.viewVersion === "intl";
      renderTableState(
        "empty",
        intl ? "还没有登录国际版账号" : "还没有登录国内版账号",
        intl ? "登录后即可每天发送活跃会话" : "登录后即可每天签到领取积分",
        "登录账号",
      );
      return;
    }

    if (!filtered.length) {
      elements.accountRows.innerHTML = "";
      renderTableState("search", "没有匹配的账户", "调整搜索内容或筛选条件");
      return;
    }

    elements.tableState.hidden = true;
    const intl = app.viewVersion === "intl";
    elements.accountRows.innerHTML = filtered
      .map((account) => {
        const daily = getDailyState(account);
        const health = getHealthState(account);
        const busy = app.busyAccounts.has(account.id);
        const name = accountDisplayName(account);
        const detail = account.nickname && account.identity !== account.nickname ? account.identity : `ID: ${account.id}`;
        const tokenExpiry = formatTokenExpiry(account);
        const reasonTooltip = account.reason ? ` data-tooltip="${escapeHTML(account.reason)}"` : "";
        const recheckMeta = formatRecheckMeta(account);
        const pointsMeta = recheckMeta || formatUpdatedAt(account.updatedAt);
        const packageCount = accountPackages(account).length;
        const packagesTooltip = packageCount
          ? `查看全部 ${integerFormatter.format(packageCount)} 个积分包`
          : "查看积分包明细";
        const dailyActionIcon = intl ? "send-horizontal" : "calendar-check-2";
        const dailyActionLabel = intl ? "活跃" : "签到";
        const travel = travelInfo(account);
        const travelRemainingText = travel.arriveMs ? travelRemaining(travel.arriveMs) : "";
        const travelText = travelRemainingText ? `${travel.base} · ${travelRemainingText}` : travel.base;
        const travelHtml = intl
          ? ""
          : `<span class="travel-meta ${travel.className}" title="${escapeHTML(travel.title)}"${travel.arriveMs ? ` data-travel-arrive="${travel.arriveMs}" data-travel-base="${escapeHTML(travel.base)}"` : ""}>${escapeHTML(travelText)}</span>`;

        return `
          <tr class="${account.enabled ? "" : "is-disabled"} ${busy ? "row-busy" : ""}" data-account-id="${escapeHTML(account.id)}">
            <td data-label="账户">
              <div class="account-cell">
                <span class="account-avatar" aria-hidden="true">${escapeHTML(accountInitial(account))}</span>
                <span class="account-identity">
                  <span class="account-name" title="${escapeHTML(name)}">${escapeHTML(name)}</span>
                  <span class="account-detail" title="${escapeHTML(detail)}">${escapeHTML(detail)}</span>
                  <span class="token-expiry ${tokenExpiry.className}" title="${escapeHTML(tokenExpiry.title)}">${escapeHTML(tokenExpiry.label)}</span>
                </span>
              </div>
            </td>
            <td data-label="状态">
              <span class="status-badge ${health.className}"${reasonTooltip}>${escapeHTML(health.label)}</span>
            </td>
            <td data-label="${intl ? "今日活跃" : "今日签到"}">
              <span class="checkin-label ${daily.className}">${escapeHTML(daily.label)}</span>
              ${travelHtml}
            </td>
            <td data-label="积分">
              <button class="points-cell" type="button" data-action="packages" data-tooltip="${escapeHTML(packagesTooltip)}" aria-label="查看${escapeHTML(name)}的积分包明细">
                <strong class="points-value">${escapeHTML(formatPoints(account.points))}</strong>
                <span class="points-meta">${escapeHTML(pointsMeta)}</span>
              </button>
            </td>
            <td data-label="启用">
              <label class="switch" data-tooltip="${account.enabled ? "停用账户" : "启用账户"}">
                <input type="checkbox" data-action="toggle" aria-label="${account.enabled ? "停用" : "启用"}${escapeHTML(name)}" ${account.enabled ? "checked" : ""} ${busy ? "disabled" : ""} />
                <span aria-hidden="true"></span>
              </label>
            </td>
            <td data-label="操作">
              <div class="row-actions">
                ${intl ? "" : `<button class="icon-button${account.travelState === "no_buddy" ? " button-attention" : ""}" type="button" data-action="adopt-buddy" aria-label="为${escapeHTML(name)}领取 Buddy" data-tooltip="领取 Buddy" ${busy || !account.enabled ? "disabled" : ""}>
                  <i data-lucide="gift" aria-hidden="true"></i>
                </button>`}
                ${intl ? "" : `<button class="icon-button" type="button" data-action="travel" aria-label="为${escapeHTML(name)}派猫猫旅行" data-tooltip="旅行" ${busy || !account.enabled ? "disabled" : ""}>
                  <i data-lucide="plane-takeoff" aria-hidden="true"></i>
                </button>`}
                <button class="icon-button" type="button" data-action="${intl ? "active" : "checkin"}" aria-label="为${escapeHTML(name)}${intl ? "发送活跃会话" : "签到"}" data-tooltip="${dailyActionLabel}" ${busy || !account.enabled ? "disabled" : ""}>
                  <i data-lucide="${dailyActionIcon}" aria-hidden="true"></i>
                </button>
                <button class="icon-button" type="button" data-action="points" aria-label="查询${escapeHTML(name)}积分" data-tooltip="查询积分" ${busy || !account.enabled ? "disabled" : ""}>
                  <i data-lucide="refresh-cw" aria-hidden="true"></i>
                </button>
                <button class="icon-button" type="button" data-action="edit" aria-label="修改${escapeHTML(name)}名称" data-tooltip="修改名称" ${busy ? "disabled" : ""}>
                  <i data-lucide="pencil" aria-hidden="true"></i>
                </button>
                <button class="icon-button button-danger" type="button" data-action="delete" aria-label="删除${escapeHTML(name)}" data-tooltip="删除账户" ${busy ? "disabled" : ""}>
                  <i data-lucide="trash-2" aria-hidden="true"></i>
                </button>
              </div>
            </td>
          </tr>
        `;
      })
      .join("");
    refreshIcons(elements.accountRows);
  }

  function renderSchedule() {
    const viewIntl = app.viewVersion === "intl";
    if (elements.versionSwitch) elements.versionSwitch.value = app.viewVersion;
    if (elements.pageTitle) elements.pageTitle.textContent = viewIntl ? "国际版" : "国内版";
    if (elements.pageBadge) elements.pageBadge.textContent = viewIntl ? "每日活跃" : "每日签到";
    if (elements.pageSubtitle) {
      elements.pageSubtitle.textContent = viewIntl
        ? "每天发送一次活跃会话，获取积分"
        : "每天在 codebuddy.cn 签到，领取积分";
    }
    elements.scheduleEnabled.checked = app.settings.scheduleEnabled;
    elements.scheduleTime.value = app.settings.scheduleTime;
    elements.scheduleTime.disabled = !app.settings.scheduleEnabled;
    elements.scheduleSummary.textContent = app.settings.scheduleEnabled
      ? `每日 ${app.settings.scheduleTime} 同时执行国内签到与国际活跃`
      : "当前已关闭";
    elements.activeModel.value = app.settings.activeModel;
    elements.activePrompt.value = app.settings.activePrompt || "";
    elements.recheckDelay.value = String(app.settings.recheckDelayMinutes);
    const proxy = parseProxyURL(app.settings.proxyUrl);
    elements.proxyScheme.value = proxy.scheme;
    elements.proxyHost.value = proxy.host;
    syncProxyInput();
    // 活跃模型/提示词/复核延迟跟随当前查看的版本列表显示（切到国际版即可见、可改）
    elements.activeModelRow.hidden = !viewIntl;
    elements.activePromptRow.hidden = !viewIntl;
    elements.recheckDelayRow.hidden = !viewIntl;
    if (elements.proxyRow) elements.proxyRow.hidden = !viewIntl;
    // 按钮文案跟随当前查看的版本列表（签到=国内列表 / 活跃=国际列表）
    elements.runAllLabel.textContent = viewIntl ? "全部活跃" : "全部签到";
    if (elements.colDaily) elements.colDaily.textContent = viewIntl ? "今日活跃" : "今日签到";
    // 自动旅行仅国内版
    const showTravel = !viewIntl;
    if (elements.runTravel) elements.runTravel.hidden = !showTravel;
    if (elements.travelEnabledRow) elements.travelEnabledRow.hidden = !showTravel;
    if (elements.travelAdoptRow) elements.travelAdoptRow.hidden = !showTravel;
    if (elements.travelEnabled) elements.travelEnabled.checked = Boolean(app.settings.travelEnabled);
    if (elements.travelAutoAdopt) elements.travelAutoAdopt.checked = Boolean(app.settings.travelAutoAdopt);
  }

  // 设置面板草稿：切版本/刷新时不丢用户未保存的编辑
  function readSettingsDraft() {
    if (!elements.scheduleForm) return null;
    return {
      scheduleEnabled: elements.scheduleEnabled.checked,
      scheduleTime: elements.scheduleTime.value,
      activeModel: elements.activeModel.value,
      activePrompt: elements.activePrompt.value,
      recheckDelay: elements.recheckDelay.value,
      proxyScheme: elements.proxyScheme.value,
      proxyHost: elements.proxyHost.value,
      travelEnabled: elements.travelEnabled.checked,
      travelAutoAdopt: elements.travelAutoAdopt.checked,
    };
  }

  function writeSettingsDraft(draft) {
    if (!draft) return;
    elements.scheduleEnabled.checked = draft.scheduleEnabled;
    elements.scheduleTime.value = draft.scheduleTime;
    elements.scheduleTime.disabled = !draft.scheduleEnabled;
    elements.activeModel.value = draft.activeModel;
    elements.activePrompt.value = draft.activePrompt;
    elements.recheckDelay.value = draft.recheckDelay;
    elements.proxyScheme.value = draft.proxyScheme;
    elements.proxyHost.value = draft.proxyHost;
    elements.travelEnabled.checked = draft.travelEnabled;
    elements.travelAutoAdopt.checked = draft.travelAutoAdopt;
    syncProxyInput();
  }

  function renderAll() {
    renderSummary();
    renderAccounts();
    renderSchedule();
    refreshPackagesDialog();
  }

  function refreshTokenExpiry() {
    if (app.isLoading) return;
    elements.accountRows.querySelectorAll("[data-account-id]").forEach((row) => {
      const account = getAccountById(row.dataset.accountId);
      if (!account || !Number.isFinite(account.tokenExpiresAt)) return;
      const node = row.querySelector(".token-expiry");
      if (!node) return;
      const tokenExpiry = formatTokenExpiry(account);
      node.className = `token-expiry ${tokenExpiry.className}`;
      node.title = tokenExpiry.title;
      node.textContent = tokenExpiry.label;
    });
  }

  function setConnection(status, label) {
    elements.connectionStatus.classList.toggle("is-online", status === "online");
    elements.connectionStatus.classList.toggle("is-error", status === "error");
    elements.connectionLabel.textContent = label;
  }

  async function loadState({ quiet = false } = {}) {
    const generation = ++app.stateLoadGeneration;
    const settingsRevision = app.settingsRevision;
    app.stateLoads += 1;
    if (!quiet) {
      app.isLoading = true;
      app.loadError = "";
      renderAccounts();
    }
    setConnection("loading", "正在连接");
    elements.refreshState.classList.add("is-spinning");
    elements.refreshState.disabled = true;

    try {
      const payload = await api("/api/state");
      if (generation !== app.stateLoadGeneration) return;
      const normalized = normalizeState(payload);
      app.cnAccounts = normalized.cnAccounts;
      app.intlAccounts = normalized.intlAccounts;
      if (settingsRevision === app.settingsRevision) {
        app.settings = normalized.settings;
        if (!app.settingsDirty) app.proxyDirty = false;
      }
      // 首次拿到服务端设置后按「默认版本」决定默认展示的列表，之后尊重手动切换
      if (!app.viewInitialized) {
        app.viewInitialized = true;
        app.viewVersion = app.settings.mode === "intl" ? "intl" : "cn";
      }
      app.accounts = app.viewVersion === "intl" ? app.intlAccounts : app.cnAccounts;
      app.loadError = "";
      setConnection("online", "服务正常");
      elements.lastUpdated.textContent = `上次同步 ${timeFormatter.format(new Date())}`;
    } catch (error) {
      if (generation !== app.stateLoadGeneration) return;
      app.loadError = getErrorMessage(error, "请确认本地服务正在运行");
      setConnection("error", "连接失败");
      elements.lastUpdated.textContent = "同步失败";
      if (quiet) toast(app.loadError, "error");
    } finally {
      app.stateLoads = Math.max(0, app.stateLoads - 1);
      if (app.stateLoads === 0) {
        elements.refreshState.classList.remove("is-spinning");
        elements.refreshState.disabled = false;
      }
      if (generation === app.stateLoadGeneration) {
        const draft = app.settingsDirty ? readSettingsDraft() : null;
        app.isLoading = false;
        renderAll();
        if (draft) writeSettingsDraft(draft);
      }
    }
  }

  function setButtonBusy(button, busy, busyText = "处理中") {
    if (!button) return;
    if (busy) {
      button.dataset.originalHtml = button.innerHTML;
      button.disabled = true;
      button.innerHTML = `<span class="loading-spinner" aria-hidden="true"></span><span>${escapeHTML(busyText)}</span>`;
    } else {
      button.disabled = false;
      if (button.dataset.originalHtml) {
        button.innerHTML = button.dataset.originalHtml;
        delete button.dataset.originalHtml;
      }
      refreshIcons(button);
    }
  }

  async function runAccountAction(account, action) {
    if (app.busyAccounts.has(account.id)) return;
    app.busyAccounts.add(account.id);
    renderAccounts();

    const actionNames = {
      checkin: "签到",
      active: "活跃会话",
      travel: "旅行",
      "adopt-buddy": "领取 Buddy",
      points: "积分查询",
      toggle: account.enabled ? "停用" : "启用",
      delete: "删除",
    };

    try {
      let result;
      if (action === "checkin") {
        result = await api(`/api/accounts/${encodeURIComponent(account.id)}/checkin`, { method: "POST" });
      } else if (action === "active") {
        result = await api(`/api/accounts/${encodeURIComponent(account.id)}/active`, { method: "POST" });
      } else if (action === "travel") {
        result = await api(`/api/accounts/${encodeURIComponent(account.id)}/travel`, { method: "POST" });
      } else if (action === "adopt-buddy") {
        result = await api(`/api/accounts/${encodeURIComponent(account.id)}/adopt-buddy`, { method: "POST" });
      } else if (action === "points") {
        result = await api(`/api/accounts/${encodeURIComponent(account.id)}/points`, { method: "POST" });
      } else if (action === "toggle") {
        result = await api(`/api/accounts/${encodeURIComponent(account.id)}`, {
          method: "PUT",
          body: JSON.stringify({ enabled: !account.enabled }),
        });
      } else if (action === "delete") {
        result = await api(`/api/accounts/${encodeURIComponent(account.id)}`, { method: "DELETE" });
      }
      const travelMessage = result?.account?.travelMessage;
      toast(
        result?.message ||
          (action === "active"
            ? "会话已发送，等待积分复核"
            : action === "travel" && travelMessage
              ? travelMessage
              : `${actionNames[action]}成功`),
      );
      await loadState({ quiet: true });
    } catch (error) {
      const actionLabel = action === "adopt-buddy" ? "领取 Buddy " : actionNames[action];
      toast(`${actionLabel}失败：${getErrorMessage(error)}`, "error");
      renderAccounts();
    } finally {
      app.busyAccounts.delete(account.id);
      renderAccounts();
    }
  }

  async function runTravelAll() {
    if (!app.accounts.some((account) => account.enabled)) {
      toast("没有可执行的已启用账户", "info");
      return;
    }
    if (elements.runTravel.disabled) return;
    setButtonBusy(elements.runTravel, true, "旅行中");
    try {
      const result = await api("/api/travel/run-all", { method: "POST", body: JSON.stringify({}) });
      const accounts = Array.isArray(result?.accounts) ? result.accounts : [];
      const traveling = accounts.filter((account) => account.travelState === "traveling" || account.travelState === "arrived").length;
      const done = accounts.filter((account) => account.travelState === "done").length;
      const failed = accounts.length - traveling - done;
      toast(
        `旅行已处理 ${integerFormatter.format(accounts.length)} 个账户：进行中 ${integerFormatter.format(traveling)}、已完成 ${integerFormatter.format(done)}${failed ? `、失败 ${integerFormatter.format(failed)}` : ""}`,
        failed ? "error" : "success",
      );
      await loadState({ quiet: true });
    } catch (error) {
      toast(`批量旅行失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.runTravel, false);
    }
  }

  async function runAllCheckins() {
    const intl = app.viewVersion === "intl";
    if (elements.runAll.disabled || !app.accounts.some((account) => account.enabled)) {
      if (!app.accounts.some((account) => account.enabled)) toast("没有可执行的已启用账户", "info");
      return;
    }

    setButtonBusy(elements.runAll, true, intl ? "活跃中" : "签到中");
    try {
      // 按当前查看的版本列表执行：cn=全部签到，intl=全部活跃
      const result = await api("/api/run-all", {
        method: "POST",
        body: JSON.stringify({ version: app.viewVersion }),
      });
      const accounts = Array.isArray(result?.accounts) ? result.accounts : [];
      const failed = accounts.filter((account) => account.lastError).length;
      const succeeded = accounts.length - failed;
      const action = intl ? "活跃会话" : "签到";
      toast(`${action}已处理 ${integerFormatter.format(accounts.length)} 个账户，成功 ${integerFormatter.format(succeeded)} 个${intl ? "（积分待复核）" : ""}${failed ? `，失败 ${integerFormatter.format(failed)} 个` : ""}`, failed ? "error" : "success");
      await loadState({ quiet: true });
    } catch (error) {
      toast(`批量${intl ? "活跃" : "签到"}失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.runAll, false);
    }
  }

  async function refreshAllPoints() {
    if (elements.refreshPointsAll.disabled || !app.accounts.some((account) => account.enabled)) {
      if (!app.accounts.some((account) => account.enabled)) toast("没有已启用的账户", "info");
      return;
    }
    setButtonBusy(elements.refreshPointsAll, true, "刷新中");
    try {
      await api("/api/refresh-points", {
        method: "POST",
        body: JSON.stringify({ version: app.viewVersion }),
      });
      toast("积分已刷新");
      await loadState({ quiet: true });
    } catch (error) {
      toast(`积分刷新失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.refreshPointsAll, false);
    }
  }

  // —— 积分包小窗 ——
  // 服务端 PointPackage.ExpiresAt 是「2026-11-25 19:12:00」这种北京时间字符串，
  // 无时区后缀时按 +08:00 解析，避免浏览器本地时区影响到期判断。
  function parsePackageExpiry(value) {
    const text = String(value || "").trim();
    if (!text) return null;
    let normalized = text.includes("T") ? text : text.replace(" ", "T");
    if (/^\d{4}-\d{2}-\d{2}$/.test(normalized)) normalized += "T00:00:00";
    const hasZone = /(?:Z|[+-]\d{2}:?\d{2})$/i.test(normalized);
    const parsed = Date.parse(hasZone ? normalized : `${normalized}+08:00`);
    return Number.isFinite(parsed) ? parsed : null;
  }

  // 只列出还有剩余的积分包（与 switch 的「全部积分包」口径一致），按到期时间升序，长期有效排最后。
  function accountPackages(account) {
    const list = Array.isArray(account?.packages) ? account.packages : [];
    return list
      .map((raw, index) => ({
        name: String(raw?.name || "").trim() || "未命名积分包",
        remain: Math.max(0, Number(raw?.remain) || 0),
        used: Math.max(0, Number(raw?.used) || 0),
        total: Math.max(0, Number(raw?.total) || 0),
        expiresAt: parsePackageExpiry(raw?.expiresAt),
        index,
      }))
      .filter((pkg) => pkg.remain > 0)
      .sort((left, right) => {
        const leftExpiry = left.expiresAt ?? Number.POSITIVE_INFINITY;
        const rightExpiry = right.expiresAt ?? Number.POSITIVE_INFINITY;
        return leftExpiry === rightExpiry ? left.index - right.index : leftExpiry - rightExpiry;
      });
  }

  function openPackagesDialog(account) {
    if (!elements.packagesDialog) return;
    app.packagesAccountId = account.id;
    const packages = accountPackages(account);
    const name = accountDisplayName(account);
    if (elements.packagesSubtitle) {
      elements.packagesSubtitle.textContent = `${name} · 共 ${integerFormatter.format(packages.length)} 个积分包`;
    }

    if (!packages.length) {
      const hint = Number.isFinite(account.points)
        ? "当前没有可展示的积分包。"
        : "尚未查询到积分明细，先点账户行的「查询积分」再试。";
      elements.packageList.innerHTML = `<p class="package-empty">${escapeHTML(hint)}</p>`;
    } else {
      const now = Date.now();
      const expiringSoonMs = 7 * 24 * 60 * 60 * 1000;
      elements.packageList.innerHTML = packages
        .map((pkg) => {
          const ratio = pkg.total > 0 ? Math.min(100, Math.max(0, (pkg.remain / pkg.total) * 100)) : 0;
          const expired = pkg.expiresAt !== null && pkg.expiresAt <= now;
          const expiringSoon = !expired && pkg.expiresAt !== null && pkg.expiresAt - now <= expiringSoonMs;
          const expiryText = expired
            ? "已到期"
            : pkg.expiresAt === null
              ? "长期有效"
              : `到期 ${packageExpiryFormatter.format(new Date(pkg.expiresAt))}`;
          const tone = expired ? " is-expired" : expiringSoon ? " is-soon" : "";
          return `
            <div class="package-item" role="listitem">
              <div class="package-row">
                <div class="package-info">
                  <span class="package-name" title="${escapeHTML(pkg.name)}">${escapeHTML(pkg.name)}</span>
                  <span class="package-expiry${tone}">${escapeHTML(expiryText)}</span>
                </div>
                <div class="package-amount">
                  <span class="package-remain">${escapeHTML(numberFormatter.format(pkg.remain))} / ${escapeHTML(numberFormatter.format(pkg.total))}</span>
                  <span class="package-used">已用 ${escapeHTML(numberFormatter.format(pkg.used))}</span>
                </div>
              </div>
              <div class="package-bar" aria-hidden="true">
                <span class="package-bar-fill${tone}" data-ratio="${ratio.toFixed(2)}"></span>
              </div>
            </div>
          `;
        })
        .join("");
      // 页面 CSP 是 style-src 'self'，HTML 里的内联 style 属性会被浏览器丢弃（进度条会变成满格），
      // 所以宽度改由 CSSOM 赋值——CSP 不拦脚本设置 el.style。
      elements.packageList.querySelectorAll(".package-bar-fill[data-ratio]").forEach((bar) => {
        bar.style.width = `${bar.dataset.ratio}%`;
      });
    }

    if (!elements.packagesDialog.open) elements.packagesDialog.showModal();
  }

  function closePackagesDialog() {
    app.packagesAccountId = "";
    if (elements.packagesDialog?.open) elements.packagesDialog.close();
  }

  // 刷新账户状态时，若小窗还开着就跟着换一份新数据（例如在窗口外点了「刷新积分」）。
  function refreshPackagesDialog() {
    if (!elements.packagesDialog?.open || !app.packagesAccountId) return;
    const account = getAccountById(app.packagesAccountId);
    if (account) {
      openPackagesDialog(account);
    } else {
      closePackagesDialog();
    }
  }

  function getAccountById(id) {
    return app.accounts.find((account) => account.id === id);
  }

  function openEditDialog(account) {
    app.editingAccountId = account.id;
    elements.editAccountId.textContent = account.identity || `ID: ${account.id}`;
    elements.nicknameInput.value = String(account.alias || "");
    elements.editDialog.showModal();
    window.setTimeout(() => {
      elements.nicknameInput.focus();
      elements.nicknameInput.select();
    }, 0);
  }

  function closeEditDialog() {
    app.editingAccountId = "";
    if (elements.editDialog.open) elements.editDialog.close();
  }

  async function saveNickname(event) {
    event.preventDefault();
    const account = getAccountById(app.editingAccountId);
    if (!account) return closeEditDialog();

    const nickname = elements.nicknameInput.value.trim();
    setButtonBusy(elements.saveNickname, true, "保存中");
    try {
      const result = await api(`/api/accounts/${encodeURIComponent(account.id)}`, {
        method: "PUT",
        body: JSON.stringify({ nickname }),
      });
      toast(result?.message || "账户名称已更新");
      closeEditDialog();
      await loadState({ quiet: true });
    } catch (error) {
      toast(`保存失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.saveNickname, false);
    }
  }

  async function saveSchedule(event) {
    event.preventDefault();
    const scheduleEnabled = elements.scheduleEnabled.checked;
    const scheduleTime = normalizeTime(elements.scheduleTime.value);
    const activeModel = ["hy3", "hy4-preview"].includes(elements.activeModel.value) ? elements.activeModel.value : "hy3";
    const recheckDelayMinutes = clampRecheckDelay(parseNumber(elements.recheckDelay.value) ?? 840);
    if (elements.proxyScheme.value && !elements.proxyHost.value.trim()) {
      toast("请填写代理网关地址和端口", "error");
      elements.proxyHost.focus();
      return;
    }
    const proxyUrl = joinProxyURL(elements.proxyScheme.value, elements.proxyHost.value);
    const activePrompt = elements.activePrompt.value.trim().slice(0, 200);
    const travelEnabled = elements.travelEnabled.checked;
    const travelAutoAdopt = elements.travelAutoAdopt.checked;
    const settingsRevision = app.settingsRevision;
    const proxyRevision = app.proxyRevision;
    setButtonBusy(elements.saveSchedule, true, "保存中");

    try {
      const settingsPayload = { scheduleEnabled, scheduleTime, activeModel, recheckDelayMinutes, activePrompt, travelEnabled, travelAutoAdopt };
      // 公开设置会隐藏代理认证信息。只有用户实际编辑过代理字段时才提交，
      // 避免保存其他设置时用脱敏 URL 覆盖 data/proxy.conf 中的真实凭据。
      if (app.proxyDirty) settingsPayload.proxyUrl = proxyUrl;
      const result = await api("/api/settings", {
        method: "PUT",
        body: JSON.stringify(settingsPayload),
      });
      // 以服务端权威响应为准（含代理等可能被后端规范化/保留的字段），避免刷新后回退
      const saved = result?.settings || {};
      const hasNewerDraft = app.settingsRevision !== settingsRevision;
      const hasNewerProxyDraft = app.proxyRevision !== proxyRevision;
      const newerDraft = hasNewerDraft ? readSettingsDraft() : null;
      app.settings = {
        scheduleEnabled: parseBoolean(firstDefined(saved.scheduleEnabled, scheduleEnabled), false),
        scheduleTime: normalizeTime(firstDefined(saved.scheduleTime, scheduleTime)),
        mode: String(firstDefined(saved.mode, app.settings.mode)) === "intl" ? "intl" : "cn",
        activeModel: ["hy3", "hy4-preview"].includes(String(saved.activeModel)) ? String(saved.activeModel) : activeModel,
        recheckDelayMinutes: clampRecheckDelay(parseNumber(saved.recheckDelayMinutes) ?? recheckDelayMinutes),
        proxyUrl: String(firstDefined(saved.proxyUrl, proxyUrl, "") || ""),
        activePrompt: String(firstDefined(saved.activePrompt, activePrompt, "") || ""),
        travelEnabled: parseBoolean(firstDefined(saved.travelEnabled, travelEnabled), false),
        travelAutoAdopt: parseBoolean(firstDefined(saved.travelAutoAdopt, travelAutoAdopt), false),
      };
      app.proxyDirty = hasNewerProxyDraft;
      app.settingsDirty = hasNewerDraft;
      app.settingsRevision += 1;
      renderAll();
      writeSettingsDraft(newerDraft);
      const proxyLabel = app.settings.proxyUrl ? "代理已启用" : "直连";
      toast(`设置已保存并持久化（国际版${proxyLabel}）`);
    } catch (error) {
      toast(`设置保存失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.saveSchedule, false);
    }
  }

  // 版本切换只影响浏览偏好：静默保存，失败不影响使用
  async function persistViewVersion(version) {
    try {
      const result = await api("/api/settings", {
        method: "PUT",
        body: JSON.stringify({ mode: version === "intl" ? "intl" : "cn" }),
      });
      if (result?.settings) {
        app.settings.mode = String(result.settings.mode) === "intl" ? "intl" : "cn";
      }
    } catch {
      // 浏览偏好的保存失败不阻塞当前视图
    }
  }

  // 导入支持两种文件：本程序的加密备份（需密码）与 workbuddy-switch 导出的
  // 明文账号数组（顶层是 JSON 数组，不需要密码）。
  const IMPORT_HINT_DEFAULT =
    "支持本程序的加密备份，以及 workbuddy-switch 导出的账号 JSON（明文数组，无需密码）。";

  function isSwitchImport(content) {
    return String(content || "")
      .replace(/^\uFEFF/, "")
      .trimStart()
      .startsWith("[");
  }

  function openBackupDialog(mode) {
    if (!elements.backupDialog) return;
    const importing = mode === "import";
    elements.backupTitle.textContent = importing ? "导入账号" : "导出账号";
    elements.backupSubtitle.textContent = importing
      ? "选择账号文件：加密备份需输入密码，switch 明文文件直接导入"
      : "导出为密码加密的 JSON 文件（含登录令牌）";
    elements.backupExportStage.hidden = importing;
    elements.backupImportStage.hidden = !importing;
    elements.backupPassword.value = "";
    elements.backupImportPassword.value = "";
    elements.backupFile.value = "";
    if (elements.backupPasswordRow) elements.backupPasswordRow.hidden = false;
    if (elements.importFormatHint) elements.importFormatHint.textContent = IMPORT_HINT_DEFAULT;
    elements.backupDialog.showModal();
  }

  // 选中文件后按首个非空白字符判断来源，即时隐藏/显示密码输入。
  async function updateImportFormatHint() {
    if (!elements.backupFile) return;
    const file = elements.backupFile.files?.[0];
    if (!file) {
      if (elements.backupPasswordRow) elements.backupPasswordRow.hidden = false;
      if (elements.importFormatHint) elements.importFormatHint.textContent = IMPORT_HINT_DEFAULT;
      return;
    }
    let switchFormat = false;
    try {
      switchFormat = isSwitchImport(await file.slice(0, 256).text());
    } catch (error) {
      switchFormat = false;
    }
    if (elements.backupPasswordRow) elements.backupPasswordRow.hidden = switchFormat;
    if (elements.importFormatHint) {
      elements.importFormatHint.textContent = switchFormat
        ? "已识别为 Switch 明文格式，无需密码；加密凭据（$wbEncrypted）无法导入，会提示重新登录。"
        : "已识别为加密备份，请输入导出时设置的密码。";
    }
  }

  async function exportAccounts() {
    const password = elements.backupPassword.value;
    if (password.length < 6) {
      toast("导出密码至少 6 位", "error");
      elements.backupPassword.focus();
      return;
    }
    setButtonBusy(elements.confirmExport, true, "导出中");
    try {
      const result = await api("/api/accounts/export", {
        method: "POST",
        body: JSON.stringify({ password }),
      });
      const content = String(result?.content || "");
      if (!content) throw new Error("服务未返回备份内容");
      downloadTextFile(content, String(result?.filename || ""), "workbuddy-accounts.json");
      toast(`已导出 ${integerFormatter.format(Number(result?.count) || 0)} 个账号`);
      elements.backupDialog.close();
    } catch (error) {
      toast(`导出失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.confirmExport, false);
    }
  }

  // 导出接口统一返回 { content, filename }，这里负责落成本地文件。
  function downloadTextFile(content, filename, fallbackName) {
    const blob = new Blob([content], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename || fallbackName;
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  // 导出 workbuddy-switch 可直接导入的明文账号 JSON（不加密，见弹窗提示）。
  async function exportSwitchAccounts() {
    setButtonBusy(elements.confirmExportSwitch, true, "导出中");
    try {
      const result = await api("/api/accounts/export-switch", { method: "POST" });
      const content = String(result?.content || "");
      if (!content) throw new Error("服务未返回导出内容");
      downloadTextFile(content, String(result?.filename || ""), "wb-switch-accounts.json");
      toast(`已导出 ${integerFormatter.format(Number(result?.count) || 0)} 个账号（Switch 格式）`);
      elements.switchDialog.close();
    } catch (error) {
      toast(`导出失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.confirmExportSwitch, false);
    }
  }

  async function importAccounts() {
    const file = elements.backupFile.files?.[0];
    const password = elements.backupImportPassword.value;
    if (!file) {
      toast("请选择账号文件", "error");
      return;
    }
    setButtonBusy(elements.confirmImport, true, "导入中");
    try {
      const content = await file.text();
      const switchFormat = isSwitchImport(content);
      if (!switchFormat && password.length < 6) {
        toast("请输入导出时设置的密码", "error");
        elements.backupImportPassword.focus();
        return;
      }
      const result = await api("/api/accounts/import", {
        method: "POST",
        body: JSON.stringify({ password: switchFormat ? "" : password, content }),
      });
      toast(result?.message || "导入完成");
      elements.backupDialog.close();
      await loadState({ quiet: true });
    } catch (error) {
      toast(`导入失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.confirmImport, false);
    }
  }

  function setLoginStage(stage, message = "") {
    elements.loginStarting.hidden = stage !== "starting";
    elements.loginReady.hidden = stage !== "ready";
    elements.loginError.hidden = stage !== "error";
    if (stage === "error") elements.loginErrorMessage.textContent = message || "请稍后重试";
    refreshIcons(elements.loginPage);
  }

  function stopLoginPolling() {
    app.loginPollGeneration += 1;
    if (app.loginPollTimer) {
      window.clearTimeout(app.loginPollTimer);
      app.loginPollTimer = null;
    }
  }

  // 登录版本明确二选一：国内版=签到，国际版=活跃，不再跟随当前列表隐式决定
  function renderLoginVersion() {
    const intl = app.loginVersion === "intl";
    elements.loginVersionButtons.forEach((button) => {
      const active = button.dataset.loginVersion === app.loginVersion;
      button.classList.toggle("is-active", active);
      button.setAttribute("aria-pressed", active ? "true" : "false");
    });
    if (elements.loginVersionHint) {
      elements.loginVersionHint.textContent = intl
        ? "国际版登录页：workbuddy.ai，登录后每天发送活跃会话"
        : "国内版登录页：codebuddy.cn，登录后每天签到领积分";
    }
    const openLabel = elements.openLoginLink.querySelector("span");
    if (openLabel) openLabel.textContent = intl ? "在新窗口打开国际版登录页" : "在新窗口打开国内版登录页";
  }

  // 面板级登录：未通过时只显示登录/初始化页，所有 API 由服务端拦截
  function setAuthChrome() {
    const authed = !app.auth.required || app.auth.authenticated;
    document.body.classList.toggle("auth-locked", !authed);
    if (elements.userChip) elements.userChip.hidden = !(authed && app.auth.required);
    if (elements.userName) elements.userName.textContent = app.auth.username || "admin";
  }

  function showAuthPage(setup) {
    app.auth.authenticated = false;
    app.auth.setupRequired = Boolean(setup);
    elements.dashboard.hidden = true;
    elements.loginPage.hidden = true;
    elements.authPage.hidden = false;
    elements.authTitle.textContent = setup ? "初始化面板" : "登录面板";
    elements.authSubtitle.textContent = setup
      ? "首次使用，请设置管理员用户名与密码"
      : "输入面板账号密码后继续";
    elements.authHint.textContent = setup
      ? "密码至少 8 位，只在本机 data/auth.json 保存哈希，不会上传。"
      : "忘记密码可删除 data/auth.json 后重新初始化。";
    elements.authConfirm.hidden = !setup;
    elements.authConfirmLabel.hidden = !setup;
    // setButtonBusy 会重建按钮内部节点，这里按 id 重新查询，避免引用失效
    const submitLabel = elements.authSubmit.querySelector("#auth-submit-label");
    if (submitLabel) submitLabel.textContent = setup ? "创建并登录" : "登录";
    elements.authPassword.value = "";
    elements.authConfirm.value = "";
    setAuthChrome();
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function showDashboard() {
    elements.authPage.hidden = true;
    elements.loginPage.hidden = true;
    elements.dashboard.hidden = false;
    setAuthChrome();
  }

  async function submitAuth(event) {
    event.preventDefault();
    const setup = app.auth.setupRequired;
    const username = elements.authUsername.value.trim();
    const password = elements.authPassword.value;
    if (!username) {
      toast("请输入用户名", "error");
      return;
    }
    if (setup ? password.length < 8 : !password) {
      toast(setup ? "密码至少 8 位" : "请输入密码", "error");
      return;
    }
    if (setup && password !== elements.authConfirm.value) {
      toast("两次输入的密码不一致", "error");
      return;
    }
    setButtonBusy(elements.authSubmit, true, setup ? "创建中" : "登录中");
    try {
      const result = await api(setup ? "/api/auth/setup" : "/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ username, password }),
      });
      app.auth = {
        required: true,
        setupRequired: false,
        authenticated: true,
        username: String(result?.username || username),
      };
      elements.authPassword.value = "";
      elements.authConfirm.value = "";
      showDashboard();
      await loadState();
    } catch (error) {
      toast(`登录失败：${getErrorMessage(error)}`, "error");
    } finally {
      setButtonBusy(elements.authSubmit, false);
    }
  }

  async function logoutPanel() {
    try {
      await api("/api/auth/logout", { method: "POST" });
    } catch {
      // 退出失败也回到登录页，本地会话已不可用
    }
    app.auth = { required: true, setupRequired: false, authenticated: false, username: "" };
    showAuthPage(false);
    elements.authUsername.value = "";
  }

  async function bootApp() {
    try {
      const state = await api("/api/auth/state");
      app.auth = {
        required: Boolean(state?.required),
        setupRequired: Boolean(state?.setupRequired),
        authenticated: Boolean(state?.authenticated),
        username: String(state?.username || ""),
      };
    } catch (error) {
      app.auth = { required: true, setupRequired: false, authenticated: false, username: "" };
      toast(getErrorMessage(error, "无法读取面板认证状态"), "error");
    }
    if (!app.auth.required || app.auth.authenticated) {
      showDashboard();
      await loadState();
      return;
    }
    showAuthPage(app.auth.setupRequired);
  }

  // 登录是页面内的独立视图：隐藏账户看板，展示版本选择与授权状态
  function openLoginPage(version = app.viewVersion) {
    app.loginVersion = version === "intl" ? "intl" : "cn";
    renderLoginVersion();
    elements.dashboard.hidden = true;
    elements.loginPage.hidden = false;
    window.scrollTo({ top: 0, behavior: "smooth" });
    startLogin();
  }

  function closeLoginPage() {
    stopLoginPolling();
    elements.loginPage.hidden = true;
    elements.dashboard.hidden = false;
  }

  async function startLogin() {
    stopLoginPolling();
    const generation = app.loginPollGeneration;
    const loginVersion = app.loginVersion;
    app.loginId = "";
    app.loginUrl = "";
    setLoginStage("starting");
    if (elements.loginPage.hidden) return;

    try {
      const result = await api("/api/login/start", {
        method: "POST",
        body: JSON.stringify({ version: loginVersion }),
      });
      if (generation !== app.loginPollGeneration || elements.loginPage.hidden) return;
      app.loginId = String(firstDefined(result?.loginId, result?.id, result?.sessionId, ""));
      app.loginUrl = String(firstDefined(result?.authUrl, result?.url, result?.loginUrl, ""));
      if (!app.loginId || !app.loginUrl) throw new Error("服务未返回有效的登录信息");

      elements.openLoginLink.href = app.loginUrl;
      elements.loginUrlLabel.textContent = `登录页：${compactUrl(app.loginUrl)}`;
      if (elements.loginFrame.getAttribute("src") !== app.loginUrl) {
        elements.loginFrame.src = app.loginUrl;
      }
      const loginIntl = loginVersion === "intl";
      const openLabel = elements.openLoginLink.querySelector("span");
      if (openLabel) openLabel.textContent = loginIntl ? "在新窗口打开国际版登录页" : "在新窗口打开国内版登录页";
      elements.loginPollStatus.className = "poll-status";
      elements.loginPollStatus.innerHTML = '<span class="pulse-dot" aria-hidden="true"></span><span>等待登录完成</span>';
      setLoginStage("ready");
      pollLogin(generation);
    } catch (error) {
      if (generation !== app.loginPollGeneration || elements.loginPage.hidden) return;
      setLoginStage("error", getErrorMessage(error));
    }
  }

  function compactUrl(url) {
    try {
      const parsed = new URL(url);
      return `${parsed.host}${parsed.pathname === "/" ? "" : parsed.pathname}`;
    } catch {
      return url.length > 54 ? `${url.slice(0, 51)}...` : url;
    }
  }

  function loginResultState(result) {
    const status = String(firstDefined(result?.status, result?.state, "pending")).toLowerCase();
    if (result?.success === true || result?.authenticated === true || ["success", "complete", "completed", "authenticated", "ok"].includes(status)) {
      return "success";
    }
    if (["error", "failed", "expired", "cancelled", "canceled", "timeout"].includes(status)) return "error";
    return "pending";
  }

  async function pollLogin(generation) {
    if (!app.loginId || generation !== app.loginPollGeneration || elements.loginPage.hidden) return;

    try {
      const result = await api("/api/login/poll", {
        method: "POST",
        body: JSON.stringify({ loginId: app.loginId }),
      });
      if (generation !== app.loginPollGeneration) return;

      const state = loginResultState(result);
      if (state === "success") {
        elements.loginPollStatus.className = "poll-status is-success";
        elements.loginPollStatus.innerHTML = '<span class="pulse-dot" aria-hidden="true"></span><span>登录成功，正在同步账户</span>';
        toast(result?.message || "账户添加成功");
        await loadState({ quiet: true });
        window.setTimeout(() => {
          if (generation === app.loginPollGeneration && !elements.loginPage.hidden) closeLoginPage();
        }, 700);
        return;
      }
      if (state === "error") {
        elements.loginPollStatus.className = "poll-status is-error";
        elements.loginPollStatus.innerHTML = `<span class="pulse-dot" aria-hidden="true"></span><span>${escapeHTML(result?.message || "登录已失效，请重试")}</span>`;
        return;
      }

      elements.loginPollStatus.className = "poll-status";
      elements.loginPollStatus.innerHTML = `<span class="pulse-dot" aria-hidden="true"></span><span>${escapeHTML(result?.message || "等待登录完成")}</span>`;
      app.loginPollTimer = window.setTimeout(() => pollLogin(generation), 1800);
    } catch (error) {
      if (generation !== app.loginPollGeneration) return;
      if ([400, 401, 403, 404, 409, 410].includes(error?.status)) {
        setLoginStage("error", getErrorMessage(error, "登录会话已过期，请重试"));
        return;
      }
      elements.loginPollStatus.className = "poll-status is-error";
      elements.loginPollStatus.innerHTML = `<span class="pulse-dot" aria-hidden="true"></span><span>${escapeHTML(getErrorMessage(error, "状态查询失败，正在重试"))}</span>`;
      app.loginPollTimer = window.setTimeout(() => pollLogin(generation), 3000);
    }
  }

  async function copyLoginLink() {
    if (!app.loginUrl) return;
    try {
      await navigator.clipboard.writeText(app.loginUrl);
      toast("登录链接已复制");
    } catch {
      const input = document.createElement("textarea");
      input.value = app.loginUrl;
      input.style.position = "fixed";
      input.style.opacity = "0";
      document.body.append(input);
      input.select();
      document.execCommand("copy");
      input.remove();
      toast("登录链接已复制");
    }
  }

  function normalizeLogs(payload) {
    if (Array.isArray(payload)) return payload;
    if (Array.isArray(payload?.logs)) return payload.logs;
    if (Array.isArray(payload?.items)) return payload.items;
    if (Array.isArray(payload?.data)) return payload.data;
    if (typeof payload?.raw === "string") return payload.raw.split(/\r?\n/).filter(Boolean);
    return [];
  }

  function normalizeLog(log, index) {
    if (typeof log === "string") {
      const match = log.match(/^\[?([^\]]+?)\]?\s+(DEBUG|INFO|WARN|WARNING|ERROR|SUCCESS)\s+(.*)$/i);
      if (match) return { time: match[1], level: match[2], message: match[3] };
      return { time: "", level: "INFO", message: log, key: index };
    }
    return {
      time: String(firstDefined(log?.time, log?.timestamp, log?.createdAt, "")),
      level: String(firstDefined(log?.level, log?.type, log?.severity, "INFO")),
      message: String(firstDefined(log?.message, log?.msg, log?.text, JSON.stringify(log))),
      key: firstDefined(log?.id, index),
    };
  }

  function formatLogTime(value) {
    if (!value) return "--";
    const date = new Date(value);
    if (!Number.isNaN(date.getTime())) {
      const parts = formatterParts(logTimeFormatter, date);
      const yearPrefix = parts.year && parts.year !== currentYearTag ? `${parts.year}-` : "";
      return `${yearPrefix}${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second}`;
    }
    const text = String(value).trim();
    const prefixed = text.match(/(\d{4})[-/](\d{1,2})[-/](\d{1,2})[T\s]+(\d{1,2}:\d{2}(?::\d{2})?)/);
    if (prefixed) {
      const pad = (n) => String(n).padStart(2, "0");
      const yearPrefix = prefixed[1] !== currentYearTag ? `${prefixed[1]}-` : "";
      return `${yearPrefix}${pad(prefixed[2])}-${pad(prefixed[3])} ${prefixed[4]}`;
    }
    const timeOnly = text.match(/\b(\d{1,2}:\d{2}(?::\d{2})?)\b/);
    return timeOnly ? timeOnly[1] : text.slice(0, 16);
  }

  function formatLogTitle(value) {
    if (!value) return "";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return String(value);
    const parts = formatterParts(logTitleFormatter, date);
    return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second}`;
  }

  function renderLogs(logs) {
    if (!logs.length) {
      elements.logList.innerHTML = `
        <div class="drawer-empty">
          <i data-lucide="file-clock" aria-hidden="true"></i>
          <strong>暂无运行记录</strong>
          <span>任务执行后会显示在这里</span>
        </div>`;
      elements.logSubtitle.textContent = "暂无记录";
      refreshIcons(elements.logList);
      return;
    }

    const normalized = logs.map(normalizeLog);
    elements.logSubtitle.textContent = `最近 ${integerFormatter.format(normalized.length)} 条`;
    elements.logList.innerHTML = normalized
      .map((log) => {
        const level = log.level.toLowerCase();
        const levelClass = level.includes("error") || level.includes("fail")
          ? "is-error"
          : level.includes("warn")
            ? "is-warn"
            : level.includes("success") || level.includes("ok")
              ? "is-success"
              : "";
        const title = formatLogTitle(log.time);
        return `
          <div class="log-entry">
            <time class="log-time"${title ? ` title="${escapeHTML(title)}"` : ""}>${escapeHTML(formatLogTime(log.time))}</time>
            <span class="log-level ${levelClass}">${escapeHTML(log.level)}</span>
            <span class="log-message">${escapeHTML(log.message)}</span>
          </div>`;
      })
      .join("");
  }

  async function loadLogs() {
    elements.refreshLogs.classList.add("is-spinning");
    elements.refreshLogs.disabled = true;
    elements.logList.innerHTML = '<div class="drawer-loading"><div class="loading-spinner" aria-hidden="true"></div><span>正在读取日志</span></div>';
    try {
      const result = await api("/api/logs");
      renderLogs(normalizeLogs(result));
    } catch (error) {
      elements.logList.innerHTML = `
        <div class="drawer-empty">
          <i data-lucide="circle-alert" aria-hidden="true"></i>
          <strong>日志读取失败</strong>
          <span>${escapeHTML(getErrorMessage(error))}</span>
        </div>`;
      elements.logSubtitle.textContent = "读取失败";
      refreshIcons(elements.logList);
    } finally {
      elements.refreshLogs.classList.remove("is-spinning");
      elements.refreshLogs.disabled = false;
    }
  }

  function openLogDrawer() {
    elements.drawerBackdrop.hidden = false;
    elements.drawer.setAttribute("aria-hidden", "false");
    requestAnimationFrame(() => {
      elements.drawerBackdrop.classList.add("is-visible");
      elements.drawer.classList.add("is-open");
    });
    document.body.style.overflow = "hidden";
    loadLogs();
  }

  function closeLogDrawer() {
    elements.drawerBackdrop.classList.remove("is-visible");
    elements.drawer.classList.remove("is-open");
    elements.drawer.setAttribute("aria-hidden", "true");
    document.body.style.overflow = "";
    window.setTimeout(() => {
      if (!elements.drawer.classList.contains("is-open")) elements.drawerBackdrop.hidden = true;
    }, 200);
  }

  function toast(message, type = "success") {
    const node = document.createElement("div");
    const icon = type === "error" ? "circle-alert" : type === "info" ? "info" : "circle-check";
    node.className = `toast is-${type}`;
    node.innerHTML = `
      <i data-lucide="${icon}" aria-hidden="true"></i>
      <span class="toast-message">${escapeHTML(message)}</span>
      <button class="toast-close" type="button" aria-label="关闭提示"><i data-lucide="x" aria-hidden="true"></i></button>
    `;
    elements.toastRegion.append(node);
    refreshIcons(node);

    let timer = window.setTimeout(() => dismissToast(node), type === "error" ? 6000 : 4000);
    node.querySelector(".toast-close").addEventListener("click", () => {
      window.clearTimeout(timer);
      dismissToast(node);
    });
  }

  function dismissToast(node) {
    if (!node?.isConnected) return;
    node.classList.add("is-leaving");
    window.setTimeout(() => node.remove(), 170);
  }

  function bindEvents() {
    elements.refreshState.addEventListener("click", () => loadState({ quiet: true }));
    elements.runAll.addEventListener("click", runAllCheckins);
    if (elements.runTravel) elements.runTravel.addEventListener("click", runTravelAll);
    elements.refreshPointsAll.addEventListener("click", refreshAllPoints);
    // 两个版本统一走浏览器授权登录（国际版为 workbuddy.ai 授权页）
    elements.addAccount.addEventListener("click", () => openLoginPage());
    elements.search.addEventListener("input", renderAccounts);
    elements.statusFilter.addEventListener("change", renderAccounts);
    elements.scheduleForm.addEventListener("submit", saveSchedule);
    // 页头版本切换：直接切换账号列表（国内版/国际版独立容器）
    elements.versionSwitch.addEventListener("change", () => {
      // 保留用户在设置面板中未保存的编辑，避免切版本/刷新时被回显值冲掉
      const draft = readSettingsDraft();
      app.viewVersion = elements.versionSwitch.value === "intl" ? "intl" : "cn";
      app.accounts = app.viewVersion === "intl" ? app.intlAccounts : app.cnAccounts;
      elements.lastUpdated.textContent = `上次同步 ${timeFormatter.format(new Date())}`;
      renderAll();
      writeSettingsDraft(draft);
      persistViewVersion(app.viewVersion);
    });
    elements.scheduleEnabled.addEventListener("change", () => {
      app.settingsDirty = true;
      app.settingsRevision += 1;
      elements.scheduleTime.disabled = !elements.scheduleEnabled.checked;
    });
    elements.scheduleForm.addEventListener("input", (event) => {
      if (event.target !== elements.proxyHost) {
        app.settingsDirty = true;
        app.settingsRevision += 1;
      }
    });
    elements.scheduleForm.addEventListener("change", (event) => {
      if (event.target !== elements.proxyScheme && event.target !== elements.scheduleEnabled) {
        app.settingsDirty = true;
        app.settingsRevision += 1;
      }
    });
    elements.proxyScheme.addEventListener("change", () => {
      app.proxyDirty = true;
      app.proxyRevision += 1;
      app.settingsDirty = true;
      app.settingsRevision += 1;
      syncProxyInput({ clearDirect: true });
    });
    elements.proxyHost.addEventListener("input", () => {
      app.proxyDirty = true;
      app.proxyRevision += 1;
      app.settingsDirty = true;
      app.settingsRevision += 1;
    });

    elements.accountRows.addEventListener("click", (event) => {
      const button = event.target.closest("[data-action]");
      if (!button || button.matches('input[type="checkbox"]')) return;
      const row = button.closest("[data-account-id]");
      const account = row ? getAccountById(row.dataset.accountId) : null;
      if (!account) return;

      const action = button.dataset.action;
      if (action === "edit") {
        openEditDialog(account);
      } else if (action === "packages") {
        openPackagesDialog(account);
      } else if (action === "delete") {
        const confirmed = window.confirm(`确定删除“${accountDisplayName(account)}”吗？此操作无法撤销。`);
        if (confirmed) runAccountAction(account, "delete");
      } else {
        runAccountAction(account, action);
      }
    });

    elements.accountRows.addEventListener("change", (event) => {
      const toggle = event.target.closest('input[data-action="toggle"]');
      if (!toggle) return;
      const row = toggle.closest("[data-account-id]");
      const account = row ? getAccountById(row.dataset.accountId) : null;
      if (!account) return;
      toggle.checked = account.enabled;
      runAccountAction(account, "toggle");
    });

    elements.tableState.addEventListener("click", (event) => {
      const button = event.target.closest("[data-state-action]");
      if (!button) return;
      if (button.dataset.stateAction === "error") loadState();
      if (button.dataset.stateAction === "empty") openLoginPage();
    });

    elements.openLogs.addEventListener("click", openLogDrawer);
    elements.closeLogs.addEventListener("click", closeLogDrawer);
    elements.drawerBackdrop.addEventListener("click", closeLogDrawer);
    elements.refreshLogs.addEventListener("click", loadLogs);

    elements.authForm.addEventListener("submit", submitAuth);
    elements.logoutButton.addEventListener("click", logoutPanel);
    elements.cancelLogin.addEventListener("click", closeLoginPage);
    elements.exportAccounts.addEventListener("click", () => openBackupDialog("export"));
    elements.importAccounts.addEventListener("click", () => openBackupDialog("import"));
    elements.confirmExport.addEventListener("click", exportAccounts);
    elements.confirmImport.addEventListener("click", importAccounts);
    elements.backupFile?.addEventListener("change", updateImportFormatHint);
    elements.exportSwitch?.addEventListener("click", () => elements.switchDialog?.showModal());
    elements.confirmExportSwitch?.addEventListener("click", exportSwitchAccounts);
    elements.retryLogin.addEventListener("click", startLogin);
    elements.loginVersionButtons.forEach((button) => {
      button.addEventListener("click", () => {
        const next = button.dataset.loginVersion === "intl" ? "intl" : "cn";
        if (next === app.loginVersion) return;
        app.loginVersion = next;
        renderLoginVersion();
        startLogin();
      });
    });
    elements.copyLoginLink.addEventListener("click", copyLoginLink);

    elements.editForm.addEventListener("submit", saveNickname);
    elements.closeEdit.addEventListener("click", closeEditDialog);
    elements.cancelEdit.addEventListener("click", closeEditDialog);
    elements.editDialog.addEventListener("cancel", () => {
      app.editingAccountId = "";
    });
    elements.editDialog.addEventListener("close", () => {
      app.editingAccountId = "";
    });

    elements.closePackages?.addEventListener("click", closePackagesDialog);
    // 点击遮罩关闭（dialog 自身就是遮罩区域，卡片内的点击不会命中）
    elements.packagesDialog?.addEventListener("click", (event) => {
      if (event.target === elements.packagesDialog) closePackagesDialog();
    });
    elements.packagesDialog?.addEventListener("close", () => {
      app.packagesAccountId = "";
    });

    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && elements.drawer.classList.contains("is-open")) closeLogDrawer();
    });
  }

  bindEvents();
  refreshIcons();
  // 令牌有效期在服务端不会主动推送；页面首次渲染后按小时更新剩余时间文案。
  app.tokenExpiryTimer = window.setInterval(refreshTokenExpiry, 60 * 60 * 1000);
  window.setInterval(refreshTravelMeta, 60 * 1000);
  bootApp();
})();
