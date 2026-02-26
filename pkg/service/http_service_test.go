package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/optiop/grafana-whatsapp-webhook/pkg/entity"
)

type mockSender struct {
	userMessages  []entity.Message
	groupMessages []entity.Message
}

func (m *mockSender) SendNewWhatsAppMessageToUser(msg entity.Message) {
	m.userMessages = append(m.userMessages, msg)
}

func (m *mockSender) SendNewWhatsAppMessageToGroup(msg entity.Message) {
	m.groupMessages = append(m.groupMessages, msg)
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthy", nil)
	w := httptest.NewRecorder()

	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "OK" {
		t.Errorf("got body %q, want %q", w.Body.String(), "OK")
	}
}

func TestSendToUser(t *testing.T) {
	appToken = "testtoken"

	tests := []struct {
		name       string
		authHeader string
		userID     string
		body       GrafanaAlert
		wantStatus int
		wantSent   bool
		wantTo     string
	}{
		{
			name:       "valid request",
			authHeader: "Bearer testtoken",
			userID:     "1234567890",
			body:       GrafanaAlert{Message: "test alert"},
			wantStatus: http.StatusOK,
			wantSent:   true,
			wantTo:     "1234567890",
		},
		{
			name:       "strips leading plus from phone number",
			authHeader: "Bearer testtoken",
			userID:     "+491234567890",
			body:       GrafanaAlert{Message: "test alert"},
			wantStatus: http.StatusOK,
			wantSent:   true,
			wantTo:     "491234567890",
		},
		{
			name:       "invalid token",
			authHeader: "Bearer wrongtoken",
			userID:     "1234567890",
			body:       GrafanaAlert{Message: "test alert"},
			wantStatus: http.StatusUnauthorized,
			wantSent:   false,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			userID:     "1234567890",
			body:       GrafanaAlert{Message: "test alert"},
			wantStatus: http.StatusUnauthorized,
			wantSent:   false,
		},
		{
			name:       "token without Bearer prefix",
			authHeader: "testtoken",
			userID:     "1234567890",
			body:       GrafanaAlert{Message: "test alert"},
			wantStatus: http.StatusUnauthorized,
			wantSent:   false,
		},
		{
			name:       "empty message",
			authHeader: "Bearer testtoken",
			userID:     "1234567890",
			body:       GrafanaAlert{Message: ""},
			wantStatus: http.StatusBadRequest,
			wantSent:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sender := &mockSender{}
			handler := sendNewGrafanaAlertWhatsAppMessageToUser(sender)

			bodyBytes, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/whatsapp/send/grafana-alert/user/"+tc.userID, bytes.NewReader(bodyBytes))
			req.SetPathValue("user_id", tc.userID)
			req.Header.Set("Content-Type", "application/json")
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tc.wantStatus)
			}

			sent := len(sender.userMessages) > 0
			if sent != tc.wantSent {
				t.Errorf("message sent = %v, want %v", sent, tc.wantSent)
			}

			if tc.wantSent && sender.userMessages[0].To != tc.wantTo {
				t.Errorf("message To = %q, want %q", sender.userMessages[0].To, tc.wantTo)
			}
		})
	}
}

func TestSendToGroup(t *testing.T) {
	appToken = "testtoken"

	tests := []struct {
		name       string
		authHeader string
		groupID    string
		body       GrafanaAlert
		wantStatus int
		wantSent   bool
	}{
		{
			name:       "valid request",
			authHeader: "Bearer testtoken",
			groupID:    "120363417630801571",
			body:       GrafanaAlert{Message: "group alert"},
			wantStatus: http.StatusOK,
			wantSent:   true,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer wrongtoken",
			groupID:    "120363417630801571",
			body:       GrafanaAlert{Message: "group alert"},
			wantStatus: http.StatusUnauthorized,
			wantSent:   false,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			groupID:    "120363417630801571",
			body:       GrafanaAlert{Message: "group alert"},
			wantStatus: http.StatusUnauthorized,
			wantSent:   false,
		},
		{
			name:       "empty message",
			authHeader: "Bearer testtoken",
			groupID:    "120363417630801571",
			body:       GrafanaAlert{Message: ""},
			wantStatus: http.StatusBadRequest,
			wantSent:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sender := &mockSender{}
			handler := sendNewGrafanaAlertWhatsAppMessageToGroup(sender)

			bodyBytes, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/whatsapp/send/grafana-alert/group/"+tc.groupID, bytes.NewReader(bodyBytes))
			req.SetPathValue("group_id", tc.groupID)
			req.Header.Set("Content-Type", "application/json")
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tc.wantStatus)
			}

			sent := len(sender.groupMessages) > 0
			if sent != tc.wantSent {
				t.Errorf("message sent = %v, want %v", sent, tc.wantSent)
			}

			if tc.wantSent && sender.groupMessages[0].To != tc.groupID {
				t.Errorf("message To = %q, want %q", sender.groupMessages[0].To, tc.groupID)
			}
		})
	}
}
