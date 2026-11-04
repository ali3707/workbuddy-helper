package service

import (
	"fmt"
	"strings"
	"time"

	"workbuddy-helper/internal/client"
	"workbuddy-helper/internal/model"
)

const (
	travelRetryInterval = 30 * time.Minute
	travelAdoptCooldown = 10 * time.Minute
)

// Travel 手动派猫猫旅行（仅国内版）：按服务端状态决定领奖、等待或派发。
func (s *Service) Travel(id string) (model.PublicAccount, error) {
	lock := s.accountLock(id)
	lock.Lock()
	defer lock.Unlock()
	account, err := s.getAccount(id)
	if err != nil {
		return account.Public(), err
	}
	if account.VersionMode() != model.ModeCN {
		return account.Public(), fmt.Errorf("%w：自动旅行仅支持国内版账号", ErrAccountModeMismatch)
	}
	a, err := s.prepareAccount(id)
	if err != nil {
		return a.Public(), err
	}
	return s.travelAccount(a)
}

// RunTravelAll 对全部启用的国内版账号执行一轮旅行（手动「全部旅行」）。
func (s *Service) RunTravelAll() []model.PublicAccount {
	s.travelMu.Lock()
	defer s.travelMu.Unlock()

	s.mu.RLock()
	ids := make([]string, 0, len(s.state.Accounts))
	for _, a := range s.state.Accounts {
		if a.Enabled && a.VersionMode() == model.ModeCN {
			ids = append(ids, a.ID)
		}
	}
	s.mu.RUnlock()

	results := make([]model.PublicAccount, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			time.Sleep(1200 * time.Millisecond)
		}
		account, err := s.Travel(id)
		if err != nil {
			if latest, getErr := s.getAccount(id); getErr == nil {
				account = latest.Public()
			}
		}
		results = append(results, account)
	}
	return results
}

// travelAccount 读取服务端状态并执行对应动作（领奖 / 等待 / 派发 / 自动领养）。
func (s *Service) travelAccount(a model.Account) (model.PublicAccount, error) {
	// 先记录尝试时间，避免调度器在请求进行中重复触发同一账号。
	_ = s.updateAccount(a.ID, func(dst *model.Account) {
		dst.TravelLastAttempt = time.Now()
	})
	a.TravelLastAttempt = time.Now()

	status, err := travelCall(s, &a, func(c *client.Client) (client.TravelStatus, error) {
		return c.TravelStatus(&a)
	})
	if err != nil {
		s.setTravelFailure(a.ID, err)
		return a.Public(), err
	}

	switch status.State {
	case "arrived":
		return s.claimTravel(&a, status)
	case "traveling":
		s.applyTravelStatus(a.ID, status, "", "旅行中")
		updated, _ := s.getAccount(a.ID)
		return updated.Public(), nil
	case "idle":
		if status.DailyLimitReached {
			s.markTravelDone(a.ID, "今日已派，明天再来")
			updated, _ := s.getAccount(a.ID)
			return updated.Public(), nil
		}
		return s.departTravel(&a)
	default:
		err := fmt.Errorf("未知旅行状态：%s", status.State)
		s.setTravelFailure(a.ID, err)
		return a.Public(), err
	}
}

func (s *Service) departTravel(a *model.Account) (model.PublicAccount, error) {
	cfg, err := travelCall(s, a, func(c *client.Client) (client.TravelConfig, error) {
		return c.TravelConfig(a)
	})
	if err != nil {
		s.setTravelFailure(a.ID, err)
		return a.Public(), err
	}
	if !cfg.Enabled || len(cfg.Locations) == 0 {
		err := fmt.Errorf("旅行地点不可用")
		s.setTravelFailure(a.ID, err)
		return a.Public(), err
	}

	unavailable := 0
	for _, location := range cfg.Locations {
		_, derr := travelCall(s, a, func(c *client.Client) (string, error) {
			return c.TravelDepart(a, location.ID)
		})
		if derr == nil {
			msg := "已派往" + fallback(location.Name, "目的地")
			if status, serr := travelCall(s, a, func(c *client.Client) (client.TravelStatus, error) {
				return c.TravelStatus(a)
			}); serr == nil {
				s.applyTravelStatus(a.ID, status, location.Name, msg)
			} else {
				s.applyTravelStatus(a.ID, client.TravelStatus{State: "traveling", Location: location}, location.Name, msg)
			}
			s.appendLog("info", a.ID, "派猫猫旅行："+msg)
			updated, _ := s.getAccount(a.ID)
			return updated.Public(), nil
		}

		switch {
		case client.IsTravelMessage(derr, "already traveling"):
			s.applyTravelStatus(a.ID, client.TravelStatus{State: "traveling"}, "", "已在旅行中")
			updated, _ := s.getAccount(a.ID)
			return updated.Public(), nil
		case client.IsTravelMessage(derr, "daily limit", "daily_limit"):
			s.markTravelDone(a.ID, "今日已派，明天再来")
			updated, _ := s.getAccount(a.ID)
			return updated.Public(), nil
		case client.IsTravelMessage(derr, "no active buddy"):
			adopted, adoptErr := s.ensureBuddy(a)
			if adoptErr != nil {
				s.setTravelFailure(a.ID, adoptErr)
				updated, _ := s.getAccount(a.ID)
				return updated.Public(), adoptErr
			}
			if !adopted {
				err := fmt.Errorf("该账号没有可用 Buddy，且自动领养未成功")
				s.setTravelFailure(a.ID, err)
				updated, _ := s.getAccount(a.ID)
				return updated.Public(), err
			}
			// 领养成功后立刻重试当前地点
			if _, retryErr := travelCall(s, a, func(c *client.Client) (string, error) {
				return c.TravelDepart(a, location.ID)
			}); retryErr == nil {
				msg := "已领养 Buddy 并派往" + fallback(location.Name, "目的地")
				if status, serr := travelCall(s, a, func(c *client.Client) (client.TravelStatus, error) {
					return c.TravelStatus(a)
				}); serr == nil {
					s.applyTravelStatus(a.ID, status, location.Name, msg)
				}
				s.appendLog("info", a.ID, "派猫猫旅行："+msg)
				updated, _ := s.getAccount(a.ID)
				return updated.Public(), nil
			}
			continue
		case client.IsTravelMessage(derr, "location not available"):
			unavailable++
			continue
		default:
			s.setTravelFailure(a.ID, derr)
			return a.Public(), derr
		}
	}

	var errAll error
	if unavailable > 0 {
		errAll = fmt.Errorf("所有旅行地点暂不可用")
	} else {
		errAll = fmt.Errorf("派发旅行失败")
	}
	s.setTravelFailure(a.ID, errAll)
	updated, _ := s.getAccount(a.ID)
	return updated.Public(), errAll
}

func (s *Service) claimTravel(a *model.Account, status client.TravelStatus) (model.PublicAccount, error) {
	result, err := travelCall(s, a, func(c *client.Client) (client.TravelClaimResult, error) {
		return c.TravelClaim(a, status.RecordID)
	})
	if err != nil {
		switch {
		case client.IsTravelMessage(err, "no unclaimed travel", "daily_limit", "already"):
			s.markTravelClaimed(a.ID, 0, "奖励已在网页端领取")
			updated, _ := s.getAccount(a.ID)
			return updated.Public(), nil
		case client.IsTravelMessage(err, "not arrived yet"):
			s.applyTravelStatus(a.ID, status, "", "已到达，等待领取")
			updated, _ := s.getAccount(a.ID)
			return updated.Public(), nil
		default:
			s.setTravelFailure(a.ID, err)
			return a.Public(), err
		}
	}
	s.markTravelClaimed(a.ID, result.RewardCredit, "旅行奖励已领取")
	s.appendLog("info", a.ID, fmt.Sprintf("旅行奖励已领取：+%d 积分", result.RewardCredit))
	updated, _ := s.getAccount(a.ID)
	return updated.Public(), nil
}

// ensureBuddy 在没有可用 Buddy 时自动领养（列表切换 → 新手领取 → 同意协议 → 开盲盒）。
func (s *Service) ensureBuddy(a *model.Account) (bool, error) {
	enabled, err := s.travelAutoAdopt()
	if err != nil || !enabled {
		return false, fmt.Errorf("该账号没有可用 Buddy；可在设置中开启「无 Buddy 时自动领养」")
	}
	// 冷却：避免每次派发都触发一轮开盒
	if last := a.TravelLastAdopt; !last.IsZero() && time.Since(last) < travelAdoptCooldown {
		return false, fmt.Errorf("刚刚已尝试领取 Buddy，请稍后再试")
	}
	_ = s.updateAccount(a.ID, func(dst *model.Account) { dst.TravelLastAdopt = time.Now() })

	if buddy, infoErr := travelCall(s, a, func(c *client.Client) (*client.Buddy, error) {
		return c.BuddyInfo(a)
	}); infoErr == nil && buddy != nil {
		return false, fmt.Errorf("Buddy 已存在但服务端未认定为活跃，请稍后重试")
	}

	list, listErr := travelCall(s, a, func(c *client.Client) ([]client.Buddy, error) {
		return c.BuddyList(a)
	})
	if listErr == nil {
		for _, buddy := range list {
			if buddy.InstanceID == 0 {
				continue
			}
			if err := travelCall0(s, a, func(c *client.Client) error {
				return c.BuddySwitch(a, buddy.InstanceID)
			}); err == nil {
				s.appendLog("info", a.ID, "已切换为现有 Buddy："+fallback(buddy.Name, "未命名"))
				return true, nil
			}
		}
	}

	if buddy, firstErr := travelCall(s, a, func(c *client.Client) (*client.Buddy, error) {
		return c.BuddyFirst(a)
	}); firstErr == nil && buddy != nil && buddy.InstanceID > 0 {
		_ = travelCall0(s, a, func(c *client.Client) error {
			return c.BuddySwitch(a, buddy.InstanceID)
		})
		s.appendLog("info", a.ID, "已领取新手 Buddy："+fallback(buddy.Name, "未命名"))
		return true, nil
	}

	agreed, agreeErr := travelCall(s, a, func(c *client.Client) (bool, error) {
		return c.BuddyAgreement(a)
	})
	if agreeErr == nil && !agreed {
		if err := travelCall0(s, a, func(c *client.Client) error {
			return c.SignBuddyAgreement(a)
		}); err != nil {
			return false, fmt.Errorf("自动同意活动协议失败：%w", err)
		}
		s.appendLog("warn", a.ID, "已代同意成长中心活动协议（用于自动领养 Buddy）")
	}

	quota, quotaErr := travelCall(s, a, func(c *client.Client) (client.BuddyQuota, error) {
		return c.BuddyQuota(a)
	})
	if quotaErr != nil {
		return false, quotaErr
	}
	if quota.Affordable <= 0 {
		return false, fmt.Errorf("没有可用 Buddy，且开盲盒能量不足（需要 %d）", quota.CostPerOpen)
	}
	opened, openErr := travelCall(s, a, func(c *client.Client) ([]client.Buddy, error) {
		return c.BuddyOpen(a, 1)
	})
	if openErr != nil {
		return false, openErr
	}
	for _, buddy := range opened {
		if buddy.InstanceID == 0 {
			continue
		}
		if err := travelCall0(s, a, func(c *client.Client) error {
			return c.BuddySwitch(a, buddy.InstanceID)
		}); err != nil {
			return false, fmt.Errorf("开盲盒成功但切换 Buddy 失败：%w", err)
		}
		s.appendLog("info", a.ID, "已自动领养并启用 Buddy："+fallback(buddy.Name, "未命名"))
		return true, nil
	}
	return false, fmt.Errorf("开盲盒未返回可用 Buddy")
}

func (s *Service) travelAutoAdopt() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Settings.TravelAutoAdopt, nil
}

func (s *Service) applyTravelStatus(id string, status client.TravelStatus, locationName, message string) {
	_ = s.updateAccount(id, func(dst *model.Account) {
		if status.State != "" {
			dst.TravelState = status.State
		}
		if name := strings.TrimSpace(locationName); name != "" {
			dst.TravelLocation = name
		} else if status.Location.Name != "" {
			dst.TravelLocation = status.Location.Name
		}
		if status.ArriveAt > 0 {
			dst.TravelArriveAt = normalizeUnix(status.ArriveAt)
		}
		if reward := status.RewardValue(); reward > 0 {
			dst.TravelReward = reward
		}
		dst.TravelMsg = message
		dst.TravelUpdatedAt = time.Now()
	})
}

func (s *Service) markTravelDone(id, message string) {
	_ = s.updateAccount(id, func(dst *model.Account) {
		dst.TravelState = "done"
		dst.TravelDate = today()
		dst.TravelMsg = message
		dst.TravelUpdatedAt = time.Now()
	})
}

func (s *Service) markTravelClaimed(id string, reward int64, message string) {
	_ = s.updateAccount(id, func(dst *model.Account) {
		dst.TravelState = "done"
		dst.TravelDate = today()
		dst.TravelReward = reward
		dst.TravelMsg = message
		dst.TravelUpdatedAt = time.Now()
		dst.TravelArriveAt = 0
	})
}

func (s *Service) setTravelFailure(id string, err error) {
	msg := "旅行失败"
	if err != nil {
		msg = err.Error()
	}
	_ = s.updateAccount(id, func(dst *model.Account) {
		if dst.TravelState == "" || dst.TravelState == "idle" || dst.TravelState == "traveling" {
			dst.TravelState = "error"
		}
		dst.TravelMsg = sanitize(msg)
		dst.TravelUpdatedAt = time.Now()
	})
	s.appendLog("warn", id, "自动旅行："+msg)
}

// runTravelIfDue 由调度器周期调用：到点领奖、旅行中到点复核、未派出则重试。
func (s *Service) runTravelIfDue() {
	if !s.travelMu.TryLock() {
		return
	}
	defer s.travelMu.Unlock()

	s.mu.RLock()
	enabled := s.state.Settings.TravelEnabled
	accounts := make([]model.Account, 0, len(s.state.Accounts))
	for _, a := range s.state.Accounts {
		if a.Enabled && a.VersionMode() == model.ModeCN {
			accounts = append(accounts, a)
		}
	}
	s.mu.RUnlock()
	if !enabled {
		return
	}

	now := time.Now()
	day := today()
	for _, a := range accounts {
		if a.TravelDate == day && a.TravelState == "done" {
			continue
		}
		if !travelDue(a, now) {
			continue
		}
		lock := s.accountLock(a.ID)
		if !lock.TryLock() {
			continue
		}
		current, err := s.getAccount(a.ID)
		if err == nil {
			_, _ = s.travelAccount(current)
		}
		lock.Unlock()
	}
}

func travelDue(a model.Account, now time.Time) bool {
	switch a.TravelState {
	case "arrived":
		return true
	case "traveling":
		if a.TravelArriveAt > 0 {
			return now.Unix() >= a.TravelArriveAt
		}
		return now.Sub(a.TravelLastAttempt) >= travelRetryInterval
	default:
		return now.Sub(a.TravelLastAttempt) >= travelRetryInterval
	}
}

func normalizeUnix(value int64) int64 {
	if value > 1_000_000_000_000 {
		return value / 1000
	}
	return value
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

// travelCall 在登录态失效时刷新一次令牌并重试。
func travelCall[T any](s *Service, a *model.Account, fn func(*client.Client) (T, error)) (T, error) {
	var zero T
	result, err := fn(s.apiclientFor(a))
	if client.IsKind(err, client.KindAuthDead) {
		if refreshErr := s.refreshAccount(a.ID, a); refreshErr != nil {
			return zero, refreshErr
		}
		return fn(s.apiclientFor(a))
	}
	return result, err
}

func travelCall0(s *Service, a *model.Account, fn func(*client.Client) error) error {
	err := fn(s.apiclientFor(a))
	if client.IsKind(err, client.KindAuthDead) {
		if refreshErr := s.refreshAccount(a.ID, a); refreshErr != nil {
			return refreshErr
		}
		return fn(s.apiclientFor(a))
	}
	return err
}
