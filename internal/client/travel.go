package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"workbuddy-helper/internal/model"
)

// 成长中心（国内版）接口前缀与站点。派猫猫旅行与 Buddy 领养都挂在这里。
const (
	growthPrefix = "/activity/growth"
	travelPrefix = growthPrefix + "/buddy/travel"
	growthOrigin = "https://www.codebuddy.cn"
	growthRefer  = growthOrigin + "/profile/growth-center"
	webUA        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// TravelError 表示成长中心接口返回的业务错误，调用方按 Message 分类处理。
type TravelError struct {
	Status  int
	Code    int
	Message string
}

func (e *TravelError) Error() string {
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Sprintf("成长中心接口错误 code=%d", e.Code)
	}
	return e.Message
}

// IsTravelMessage 判断业务错误文案是否包含任一关键字（小写比较）。
func IsTravelMessage(err error, keywords ...string) bool {
	var te *TravelError
	if !errors.As(err, &te) {
		return false
	}
	lower := strings.ToLower(te.Message)
	for _, keyword := range keywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

type TravelLocation struct {
	ID   json.RawMessage `json:"id"`
	Name string          `json:"name"`
}

// TravelConfig 是 /buddy/travel/config 的响应。
//
// 注意：官网并不读取响应里的 enabled 字段，而是直接用
// `enabled = locations.length > 0` 推导（见 GrowthCenterPage 的 fetchConfig）。
// 这里保留指针：字段缺省时不能当成「功能关闭」，否则 locations 有值也会被误判。
type TravelConfig struct {
	Enabled   *bool            `json:"enabled"`
	Locations []TravelLocation `json:"locations"`
}

// Disabled 表示服务端显式关闭了旅行功能（字段存在且为 false）。
func (c TravelConfig) Disabled() bool {
	return c.Enabled != nil && !*c.Enabled
}

// Available 与官网一致：有地点就认为旅行可用。
func (c TravelConfig) Available() bool {
	return !c.Disabled() && len(c.Locations) > 0
}

type TravelStatus struct {
	State             string          `json:"state"`
	Location          TravelLocation  `json:"location"`
	DepartAt          int64           `json:"depart_at"`
	ArriveAt          int64           `json:"arrive_at"`
	DailyLimitReached bool            `json:"daily_limit_reached"`
	BuddyID           int64           `json:"buddy_id"`
	RecordID          int64           `json:"record_id"`
	ServerNow         int64           `json:"server_now"`
	RewardCredit      json.RawMessage `json:"reward_credit"`
}

// RewardValue 把 reward_credit 的数字/字符串表示归一为整数。
func (s TravelStatus) RewardValue() int64 {
	return parseCredit(s.RewardCredit)
}

type TravelClaimResult struct {
	RewardCredit int64
}

type Buddy struct {
	InstanceID   int64  `json:"instance_id"`
	TemplateID   int64  `json:"template_id"`
	Name         string `json:"name"`
	Rarity       string `json:"rarity"`
	CurrentBuddy bool   `json:"current_buddy"`
}

type BuddyQuota struct {
	CostPerOpen int64 `json:"cost_per_open"`
	Affordable  int64 `json:"affordable"`
	MaxOpen     int64 `json:"max_open_count"`
}

func parseCredit(raw json.RawMessage) int64 {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return 0
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return int64(number)
	}
	var quoted string
	if err := json.Unmarshal(raw, &quoted); err == nil {
		var parsed float64
		if _, err := fmt.Sscanf(strings.TrimSpace(quoted), "%f", &parsed); err == nil {
			return int64(parsed)
		}
	}
	return 0
}

// growthDo 调用成长中心接口。业务错误以 *TravelError 返回，认证失效仍用 *Error。
func (c *Client) growthDo(a *model.Account, method, path string, body any) (json.RawMessage, error) {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.base(a, true)+growthPrefix+path, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	// 成长中心按 web 端识别：不发送桌面客户端 UA，改为页面来源头。
	req.Header.Set("User-Agent", webUA)
	req.Header.Set("x-client-platform", "web")
	req.Header.Set("Origin", growthOrigin)
	req.Header.Set("Referer", growthRefer)
	if a.UID != "" {
		req.Header.Set("X-User-Id", a.UID)
	}
	if a.EnterpriseID != "" {
		req.Header.Set("X-Enterprise-Id", a.EnterpriseID)
		req.Header.Set("X-Tenant-Id", a.EnterpriseID)
	}
	if a.Domain != "" {
		req.Header.Set("X-Domain", a.Domain)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &Error{Kind: KindTransport, Msg: err.Error()}
	}
	defer resp.Body.Close()
	raw, readErr := readResponseBody(resp.Body, 2<<20)
	if readErr != nil {
		return nil, &Error{Kind: KindTransport, Status: resp.StatusCode, Msg: "读取成长中心响应失败"}
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &Error{Kind: KindAuthDead, Status: resp.StatusCode, Msg: "成长中心登录态失效"}
	}

	var env struct {
		Code    int             `json:"code"`
		Msg     string          `json:"msg"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil, &Error{Kind: KindProtocol, Status: resp.StatusCode, Msg: "成长中心响应不是有效 JSON"}
		}
		return nil, classifyHTTP(resp.StatusCode, string(raw))
	}
	message := strings.TrimSpace(env.Message)
	if message == "" {
		message = strings.TrimSpace(env.Msg)
	}
	if env.Code == 0 || env.Code == 200 {
		if resp.StatusCode >= 400 {
			return nil, &TravelError{Status: resp.StatusCode, Code: env.Code, Message: message}
		}
		return env.Data, nil
	}
	return nil, &TravelError{Status: resp.StatusCode, Code: env.Code, Message: message}
}

// TravelConfig 读取旅行地点与开关。
func (c *Client) TravelConfig(a *model.Account) (TravelConfig, error) {
	data, err := c.growthDo(a, http.MethodGet, "/buddy/travel/config", nil)
	if err != nil {
		return TravelConfig{}, err
	}
	var cfg TravelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return TravelConfig{}, &Error{Kind: KindProtocol, Msg: "旅行配置格式不正确"}
	}
	return cfg, nil
}

// TravelStatus 查询当前旅行状态。
func (c *Client) TravelStatus(a *model.Account) (TravelStatus, error) {
	data, err := c.growthDo(a, http.MethodGet, "/buddy/travel/status", nil)
	if err != nil {
		return TravelStatus{}, err
	}
	var status TravelStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return TravelStatus{}, &Error{Kind: KindProtocol, Msg: "旅行状态格式不正确"}
	}
	if strings.TrimSpace(status.State) == "" {
		return TravelStatus{}, &Error{Kind: KindProtocol, Msg: "旅行状态缺少 state 字段"}
	}
	return status, nil
}

// TravelDepart 派猫猫出行，返回服务端确认的状态。
// locationID 为空时按官网「未指定地点」的形式发送空对象，由服务端自行选点。
func (c *Client) TravelDepart(a *model.Account, locationID json.RawMessage) (string, error) {
	body := map[string]any{}
	if raw := strings.TrimSpace(string(locationID)); raw != "" && raw != "null" {
		body["location_id"] = locationID
	}
	data, err := c.growthDo(a, http.MethodPost, "/buddy/travel/depart", body)
	if err != nil {
		return "", err
	}
	var result struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(data, &result); err != nil || strings.TrimSpace(result.State) == "" {
		return "traveling", nil
	}
	return result.State, nil
}

// TravelClaim 领取到达后的旅行奖励。
func (c *Client) TravelClaim(a *model.Account, recordID int64) (TravelClaimResult, error) {
	body := map[string]any{}
	if recordID > 0 {
		body["record_id"] = recordID
	}
	data, err := c.growthDo(a, http.MethodPost, "/buddy/travel/claim", body)
	if err != nil {
		return TravelClaimResult{}, err
	}
	var result struct {
		RewardCredit json.RawMessage `json:"reward_credit"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return TravelClaimResult{}, nil
	}
	return TravelClaimResult{RewardCredit: parseCredit(result.RewardCredit)}, nil
}

// BuddyInfo 返回当前生效的 Buddy（没有则返回 nil）。
// 服务端可能把 Buddy 放在 buddy / current_buddy 字段，或直接返回对象本身，这里都兼容。
func (c *Client) BuddyInfo(a *model.Account) (*Buddy, error) {
	data, err := c.growthDo(a, http.MethodGet, "/buddy/info", nil)
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"buddy", "current_buddy", "currentBuddy"} {
		var envelope map[string]json.RawMessage
		if json.Unmarshal(data, &envelope) != nil {
			break
		}
		raw, ok := envelope[key]
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(string(raw))
		if trimmed == "" || trimmed == "null" {
			return nil, nil
		}
		if strings.HasPrefix(trimmed, "{") {
			var buddy Buddy
			if json.Unmarshal(raw, &buddy) == nil && buddy.InstanceID > 0 {
				return &buddy, nil
			}
			return nil, nil
		}
		if id := parseCredit(raw); id > 0 {
			return &Buddy{InstanceID: id}, nil
		}
		return nil, nil
	}
	var direct Buddy
	if json.Unmarshal(data, &direct) == nil && direct.InstanceID > 0 {
		return &direct, nil
	}
	return nil, nil
}

// BuddyList 返回已拥有的 Buddy 列表。
func (c *Client) BuddyList(a *model.Account) ([]Buddy, error) {
	data, err := c.growthDo(a, http.MethodGet, "/buddy/list", nil)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Buddies []Buddy `json:"buddies"`
		List    []Buddy `json:"list"`
		Items   []Buddy `json:"items"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil {
		switch {
		case len(envelope.Buddies) > 0:
			return envelope.Buddies, nil
		case len(envelope.List) > 0:
			return envelope.List, nil
		case len(envelope.Items) > 0:
			return envelope.Items, nil
		}
	}
	var direct []Buddy
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, nil
	}
	return nil, nil
}

// BuddyQuota 返回开盲盒所需能量与可开次数。
func (c *Client) BuddyQuota(a *model.Account) (BuddyQuota, error) {
	data, err := c.growthDo(a, http.MethodGet, "/buddy/quota", nil)
	if err != nil {
		return BuddyQuota{}, err
	}
	var quota BuddyQuota
	if err := json.Unmarshal(data, &quota); err != nil {
		return BuddyQuota{}, &Error{Kind: KindProtocol, Msg: "Buddy 配额格式不正确"}
	}
	return quota, nil
}

// BuddyAgreement 返回是否已同意活动协议。
func (c *Client) BuddyAgreement(a *model.Account) (bool, error) {
	data, err := c.growthDo(a, http.MethodGet, "/buddy/agreement", nil)
	if err != nil {
		return false, err
	}
	var result struct {
		Agreed bool `json:"agreed"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return false, &Error{Kind: KindProtocol, Msg: "Buddy 协议状态格式不正确"}
	}
	return result.Agreed, nil
}

// SignBuddyAgreement 代用户同意活动协议（仅在开启自动领养时调用）。
func (c *Client) SignBuddyAgreement(a *model.Account) error {
	_, err := c.growthDo(a, http.MethodPost, "/buddy/agreement", map[string]any{"agree": true})
	return err
}

// BuddyFirst 领取新手第一只 Buddy。
func (c *Client) BuddyFirst(a *model.Account) (*Buddy, error) {
	data, err := c.growthDo(a, http.MethodPost, "/buddy/first", map[string]any{})
	if err != nil {
		return nil, err
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(data, &envelope) == nil {
		for _, key := range []string{"buddy", "instance"} {
			if raw, ok := envelope[key]; ok && strings.HasPrefix(strings.TrimSpace(string(raw)), "{") {
				var buddy Buddy
				if json.Unmarshal(raw, &buddy) == nil && buddy.InstanceID > 0 {
					return &buddy, nil
				}
			}
		}
	}
	var direct Buddy
	if json.Unmarshal(data, &direct) == nil && direct.InstanceID > 0 {
		return &direct, nil
	}
	return nil, nil
}

// BuddyOpen 开盲盒，返回新获得的 Buddy 列表。
func (c *Client) BuddyOpen(a *model.Account, count int) ([]Buddy, error) {
	if count <= 0 {
		count = 1
	}
	data, err := c.growthDo(a, http.MethodPost, "/buddy/open", map[string]any{"count": count})
	if err != nil {
		return nil, err
	}
	var result struct {
		Results []Buddy `json:"results"`
		Buddies []Buddy `json:"buddies"`
		Items   []Buddy `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, &Error{Kind: KindProtocol, Msg: "开盲盒结果格式不正确"}
	}
	switch {
	case len(result.Results) > 0:
		return result.Results, nil
	case len(result.Buddies) > 0:
		return result.Buddies, nil
	default:
		return result.Items, nil
	}
}

// BuddySwitch 切换当前生效的 Buddy。
func (c *Client) BuddySwitch(a *model.Account, instanceID int64) error {
	_, err := c.growthDo(a, http.MethodPost, "/buddy/switch", map[string]any{"instance_id": instanceID})
	return err
}
