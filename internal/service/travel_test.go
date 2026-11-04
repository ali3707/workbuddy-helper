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
			call := departCalls
			mu.Unlock()
			if call == 1 {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"code":4001,"message":"no active buddy"}`))
				return
			}
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
	if calls != 2 {
		t.Fatalf("expected depart retry after adoption, got %d calls", calls)
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
