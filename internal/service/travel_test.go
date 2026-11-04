package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"workbuddy-helper/internal/model"
)

func growthEnvelope(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":` + data + `}`))
}

func TestTravelAutoAdoptsBuddyAndDeparts(t *testing.T) {
	var mu sync.Mutex
	departCalls := 0
	switched := int64(0)
	state := "idle"

	svc, closeUpstream := newAccountService(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/activity/growth/buddy/travel/status":
			growthEnvelope(w, `{"state":"`+state+`","location":{"id":"loc1","name":"大理"},"arrive_at":0,"daily_limit_reached":false,"record_id":0}`)
		case "/activity/growth/buddy/travel/config":
			growthEnvelope(w, `{"enabled":true,"locations":[{"id":"loc1","name":"大理"}]}`)
		case "/activity/growth/buddy/travel/depart":
			mu.Lock()
			departCalls++
			mu.Unlock()
			state = "traveling"
			growthEnvelope(w, `{"state":"traveling"}`)
		case "/activity/growth/buddy/info":
			growthEnvelope(w, `{"buddy":null}`)
		case "/activity/growth/buddy/list":
			growthEnvelope(w, `{"buddies":[]}`)
		case "/activity/growth/buddy/first":
			growthEnvelope(w, `{"buddy":null}`)
		case "/activity/growth/buddy/agreement":
			if r.Method == http.MethodPost {
				growthEnvelope(w, `{}`)
				return
			}
			growthEnvelope(w, `{"agreed":false}`)
		case "/activity/growth/buddy/quota":
			growthEnvelope(w, `{"cost_per_open":10,"affordable":2,"max_open_count":5}`)
		case "/activity/growth/buddy/open":
			growthEnvelope(w, `{"results":[{"instance_id":99,"name":"奶糖","current_buddy":true}]}`)
		case "/activity/growth/buddy/switch":
			var body struct {
				InstanceID int64 `json:"instance_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			switched = body.InstanceID
			mu.Unlock()
			growthEnvelope(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer closeUpstream()

	yes := true
	if _, err := svc.UpdateSettings(model.SettingsPatch{TravelAutoAdopt: &yes, TravelEnabled: &yes}); err != nil {
		t.Fatal(err)
	}

	account, err := svc.Travel("a1")
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	calls, gotSwitch := departCalls, switched
	mu.Unlock()
	if calls != 1 {
		t.Fatalf("expected single depart after pre-adoption, got %d calls", calls)
	}
	if gotSwitch != 99 {
		t.Fatalf("opened buddy was not activated: instance=%d", gotSwitch)
	}
	if account.TravelState != "traveling" || account.TravelLocation != "大理" {
		t.Fatalf("unexpected travel state: %+v", account)
	}
}

func TestTravelClaimsArrivedReward(t *testing.T) {
	claimed := false
	svc, closeUpstream := newAccountService(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/activity/growth/buddy/travel/status":
			growthEnvelope(w, `{"state":"arrived","location":{"id":"loc1","name":"敦煌"},"arrive_at":1,"daily_limit_reached":false,"record_id":7,"reward_credit":30}`)
		case "/activity/growth/buddy/travel/claim":
			claimed = true
			growthEnvelope(w, `{"reward_credit":30}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer closeUpstream()

	account, err := svc.Travel("a1")
	if err != nil {
		t.Fatal(err)
	}
	if !claimed {
		t.Fatal("arrived travel did not claim reward")
	}
	if account.TravelState != "done" || account.TravelReward != 30 {
		t.Fatalf("unexpected travel result: %+v", account)
	}
	loaded, err := svc.getAccount("a1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TravelDate != today() {
		t.Fatalf("travel date not recorded: %+v", loaded)
	}
}

func TestTravelStopsAtDailyLimit(t *testing.T) {
	departCalls := 0
	svc, closeUpstream := newAccountService(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/activity/growth/buddy/travel/status":
			growthEnvelope(w, `{"state":"idle","daily_limit_reached":true}`)
		case "/activity/growth/buddy/travel/depart":
			departCalls++
			growthEnvelope(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer closeUpstream()

	account, err := svc.Travel("a1")
	if err != nil {
		t.Fatal(err)
	}
	if departCalls != 0 {
		t.Fatalf("daily-limit account should not depart: %d", departCalls)
	}
	if account.TravelState != "done" {
		t.Fatalf("unexpected state: %+v", account)
	}
}

func TestTravelRejectsInternationalAccount(t *testing.T) {
	svc, closeUpstream := newAccountService(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	defer closeUpstream()
	if err := svc.updateAccount("a1", func(a *model.Account) {
		a.Version = model.ModeIntl
		a.Domain = "www.workbuddy.ai"
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Travel("a1"); err == nil || !strings.Contains(err.Error(), "国内版") {
		t.Fatalf("international account was not rejected: %v", err)
	}
}

func TestTravelDueSchedule(t *testing.T) {
	now := time.Now()
	if !travelDue(model.Account{TravelState: "arrived"}, now) {
		t.Fatal("arrived travel should be due")
	}
	if travelDue(model.Account{TravelState: "traveling", TravelArriveAt: now.Add(time.Hour).Unix()}, now) {
		t.Fatal("traveling before arrival should wait")
	}
	if !travelDue(model.Account{TravelState: "traveling", TravelArriveAt: now.Add(-time.Minute).Unix()}, now) {
		t.Fatal("traveling after arrival should be rechecked")
	}
	if !travelDue(model.Account{TravelState: "idle", TravelLastAttempt: now.Add(-31 * time.Minute)}, now) {
		t.Fatal("idle past retry interval should be due")
	}
	if travelDue(model.Account{TravelState: "idle", TravelLastAttempt: now.Add(-time.Minute)}, now) {
		t.Fatal("idle within retry interval should wait")
	}
}

// 官网的 config 不返回 enabled 字段，只用 locations 判断能否旅行。
// 旧实现要求 enabled==true，导致有地点也被误判为「旅行地点不可用」。
func TestTravelDepartsWhenConfigOmitsEnabled(t *testing.T) {
	var mu sync.Mutex
	departBodies := make([]string, 0, 1)
	state := "idle"

	svc, closeUpstream := newAccountService(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/activity/growth/buddy/travel/status":
			growthEnvelope(w, `{"state":"`+state+`","daily_limit_reached":false}`)
		case "/activity/growth/buddy/travel/config":
			growthEnvelope(w, `{"locations":[{"id":"loc7","name":"敦煌"}]}`)
		case "/activity/growth/buddy/info":
			growthEnvelope(w, `{"buddy":{"instance_id":11,"name":"奶糖"}}`)
		case "/activity/growth/buddy/travel/depart":
			var body struct {
				LocationID string `json:"location_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			departBodies = append(departBodies, body.LocationID)
			state = "traveling"
			mu.Unlock()
			growthEnvelope(w, `{"state":"traveling"}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer closeUpstream()

	account, err := svc.Travel("a1")
	if err != nil {
		t.Fatalf("depart failed without enabled field: %v", err)
	}
	mu.Lock()
	got := append([]string(nil), departBodies...)
	mu.Unlock()
	if len(got) != 1 || got[0] != "loc7" {
		t.Fatalf("unexpected depart bodies: %v", got)
	}
	if account.TravelState != "traveling" || account.TravelLocation != "敦煌" {
		t.Fatalf("unexpected travel state: %+v", account)
	}
}

// 没有 Buddy 且未开启自动领养时，手动点击应立即给出可读提示，而不是先派发再报错。
func TestTravelWithoutBuddyAndAdoptDisabled(t *testing.T) {
	var mu sync.Mutex
	departCalls := 0

	svc, closeUpstream := newAccountService(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/activity/growth/buddy/travel/status":
			growthEnvelope(w, `{"state":"idle","daily_limit_reached":false}`)
		case "/activity/growth/buddy/travel/config":
			growthEnvelope(w, `{"locations":[{"id":"loc1","name":"大理"}]}`)
		case "/activity/growth/buddy/info":
			growthEnvelope(w, `{"buddy":null}`)
		case "/activity/growth/buddy/list":
			growthEnvelope(w, `{"buddies":[]}`)
		case "/activity/growth/buddy/travel/depart":
			mu.Lock()
			departCalls++
			mu.Unlock()
			growthEnvelope(w, `{"state":"traveling"}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer closeUpstream()

	account, err := svc.Travel("a1")
	if err == nil || !strings.Contains(err.Error(), "Buddy") {
		t.Fatalf("expected buddy hint, got %v", err)
	}
	mu.Lock()
	calls := departCalls
	mu.Unlock()
	if calls != 0 {
		t.Fatalf("depart should be skipped when no buddy, got %d calls", calls)
	}
	if account.TravelState != "no_buddy" {
		t.Fatalf("unexpected travel state: %+v", account)
	}
}

// 配置没有地点时，退回「不指定地点」派发，交给服务端选点。
func TestTravelDepartsWithoutLocationsFallback(t *testing.T) {
	var mu sync.Mutex
	body := ""

	svc, closeUpstream := newAccountService(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/activity/growth/buddy/travel/status":
			growthEnvelope(w, `{"state":"idle","daily_limit_reached":false}`)
		case "/activity/growth/buddy/travel/config":
			growthEnvelope(w, `{"locations":[]}`)
		case "/activity/growth/buddy/info":
			growthEnvelope(w, `{"buddy":{"instance_id":5}}`)
		case "/activity/growth/buddy/travel/depart":
			raw := make([]byte, 0, 64)
			buf := make([]byte, 64)
			for {
				n, err := r.Body.Read(buf)
				raw = append(raw, buf[:n]...)
				if err != nil {
					break
				}
			}
			mu.Lock()
			body = string(raw)
			mu.Unlock()
			growthEnvelope(w, `{"state":"traveling"}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer closeUpstream()

	if _, err := svc.Travel("a1"); err != nil {
		t.Fatalf("fallback depart failed: %v", err)
	}
	mu.Lock()
	gotBody := body
	mu.Unlock()
	if strings.Contains(gotBody, "location_id") {
		t.Fatalf("fallback depart should omit location_id, got %s", gotBody)
	}
}
