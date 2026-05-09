package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blackgrid/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestValidateChannel(t *testing.T) {
	cases := []struct {
		name        string
		channelName string
		channelType string
		config      string
		wantErr     bool
	}{
		{"valid webhook", "Ops", ChannelTypeWebhook, `{"url":"https://example.com/hook"}`, false},
		{"webhook missing url", "Ops", ChannelTypeWebhook, `{}`, true},
		{"webhook ftp scheme rejected", "Ops", ChannelTypeWebhook, `{"url":"ftp://example.com"}`, true},
		{"webhook http accepted", "Ops", ChannelTypeWebhook, `{"url":"http://example.com"}`, false},
		{"smtp valid", "Mail", ChannelTypeSMTP, `{"host":"smtp.example.com","port":587,"from":"a@b.com","to":["c@d.com"]}`, false},
		{"smtp missing host", "Mail", ChannelTypeSMTP, `{"port":587,"from":"a@b.com","to":["c@d.com"]}`, true},
		{"smtp missing from", "Mail", ChannelTypeSMTP, `{"host":"smtp","port":587,"to":["c@d.com"]}`, true},
		{"smtp missing recipients", "Mail", ChannelTypeSMTP, `{"host":"smtp","port":587,"from":"a@b.com"}`, true},
		{"missing name", "", ChannelTypeWebhook, `{"url":"https://example.com"}`, true},
		{"unsupported channel_type", "X", "slack", `{}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateChannel(tc.channelName, tc.channelType, json.RawMessage(tc.config))
			if (err != nil) != tc.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestMaskConfigSMTP(t *testing.T) {
	ch := db.NotificationChannel{
		ChannelType: ChannelTypeSMTP,
		Config:      []byte(`{"host":"smtp","password":"super-secret","from":"a@b.com"}`),
	}
	masked := MaskConfig(ch)
	if strings.Contains(string(masked), "super-secret") {
		t.Fatalf("masked config still contains password: %s", masked)
	}
	if !strings.Contains(string(masked), maskedSecret) {
		t.Fatalf("masked config missing mask placeholder: %s", masked)
	}
}

func TestMaskConfigWebhookUntouched(t *testing.T) {
	ch := db.NotificationChannel{
		ChannelType: ChannelTypeWebhook,
		Config:      []byte(`{"url":"https://x"}`),
	}
	masked := MaskConfig(ch)
	if string(masked) != `{"url":"https://x"}` {
		t.Fatalf("webhook config was mutated: %s", masked)
	}
}

func TestMergeSMTPPasswordPreservesExisting(t *testing.T) {
	existing := []byte(`{"host":"h","password":"old"}`)
	new := json.RawMessage(`{"host":"h","password":""}`)
	merged, err := mergeSMTPPassword(existing, new)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(merged, &got)
	if got["password"] != "old" {
		t.Fatalf("expected old password preserved, got %v", got["password"])
	}
}

func TestMergeSMTPPasswordReplacesMaskWithExisting(t *testing.T) {
	existing := []byte(`{"host":"h","password":"old"}`)
	new := json.RawMessage(`{"host":"h","password":"********"}`)
	merged, _ := mergeSMTPPassword(existing, new)
	var got map[string]any
	_ = json.Unmarshal(merged, &got)
	if got["password"] != "old" {
		t.Fatalf("masked sentinel should be replaced; got %v", got["password"])
	}
}

func TestBuildIncidentPayloadOpened(t *testing.T) {
	monitor := db.Monitor{Name: "PostgreSQL", MonitorType: "tcp", Target: "10.10.13.20:5432", Status: "down"}
	incident := db.Incident{Status: "open", Severity: "critical"}
	payload := buildIncidentPayload(EventIncidentOpened, incident, monitor)

	if payload["event"] != EventIncidentOpened {
		t.Errorf("event mismatch: %v", payload["event"])
	}
	if payload["severity"] != "critical" {
		t.Errorf("severity mismatch: %v", payload["severity"])
	}
	mon := payload["monitor"].(map[string]any)
	if mon["name"] != "PostgreSQL" || mon["target"] != "10.10.13.20:5432" {
		t.Errorf("monitor payload wrong: %v", mon)
	}
	if !strings.Contains(payload["message"].(string), "is down") {
		t.Errorf("expected 'is down' in message, got %v", payload["message"])
	}
}

func TestBuildIncidentPayloadResolved(t *testing.T) {
	monitor := db.Monitor{Name: "PG", Status: "up"}
	incident := db.Incident{Status: "resolved"}
	payload := buildIncidentPayload(EventIncidentResolved, incident, monitor)
	if !strings.Contains(payload["message"].(string), "recovered") {
		t.Errorf("expected 'recovered' in message, got %v", payload["message"])
	}
}

func TestSendWebhookSuccess(t *testing.T) {
	got := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		got <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := &NotificationService{httpClient: srv.Client()}
	cfg := []byte(`{"url":"` + srv.URL + `","method":"POST"}`)
	if err := svc.sendWebhook(testCtx(t), cfg, []byte(`{"event":"x"}`)); err != nil {
		t.Fatalf("sendWebhook returned error: %v", err)
	}
	select {
	case body := <-got:
		if !strings.Contains(string(body), `"event":"x"`) {
			t.Errorf("server did not receive payload: %s", body)
		}
	default:
		t.Fatal("server never received request")
	}
}

func TestSendWebhookErrorOn500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := &NotificationService{httpClient: srv.Client()}
	cfg := []byte(`{"url":"` + srv.URL + `"}`)
	err := svc.sendWebhook(testCtx(t), cfg, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestSendWebhookRejectsNonHTTP(t *testing.T) {
	svc := &NotificationService{httpClient: http.DefaultClient}
	cfg := []byte(`{"url":"file:///etc/passwd"}`)
	err := svc.sendWebhook(testCtx(t), cfg, []byte(`{}`))
	if err == nil {
		t.Fatal("expected rejection of non-http url")
	}
}

func TestUUIDToHexFormat(t *testing.T) {
	var u pgtype.UUID
	u.Bytes = [16]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00}
	u.Valid = true
	got := uuidToHex(u)
	want := "aabbccdd-eeff-1122-3344-556677889900"
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}
