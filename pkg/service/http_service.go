package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/optiop/grafana-whatsapp-webhook/pkg/entity"
)

const maxBodyBytes = 1 << 20 // 1 MB

// MessageSender is satisfied by *whatsapp.WhatsappService and can be mocked in tests.
type MessageSender interface {
	SendNewWhatsAppMessageToUser(msg entity.Message)
	SendNewWhatsAppMessageToGroup(msg entity.Message)
}

var appToken = os.Getenv("WEBHOOK_SECRET")

// authenticate checks the Authorization header against appToken.
// It returns true and the bearer token when valid, false otherwise.
func authenticate(r *http.Request) bool {
	header := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token != appToken {
		return false
	}
	return true
}

func sendNewGrafanaAlertWhatsAppMessageToUser(ms MessageSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		phoneNumber := r.PathValue("user_id")
		if phoneNumber == "" {
			http.Error(w, "phone number is required", http.StatusBadRequest)
			return
		}

		if phoneNumber[0] == '+' {
			phoneNumber = phoneNumber[1:]
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var alert GrafanaAlert
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			http.Error(w, "error decoding alert", http.StatusBadRequest)
			return
		}

		if alert.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		ms.SendNewWhatsAppMessageToUser(entity.Message{
			To:   phoneNumber,
			Type: "user",
			Body: alert.Message,
		})

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Message sent to " + phoneNumber))
	}
}

func sendNewGrafanaAlertWhatsAppMessageToGroup(ms MessageSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		groupId := r.PathValue("group_id")
		if groupId == "" {
			http.Error(w, "group_id is required", http.StatusBadRequest)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var alert GrafanaAlert
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			http.Error(w, "error decoding alert", http.StatusBadRequest)
			return
		}

		if alert.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		ms.SendNewWhatsAppMessageToGroup(entity.Message{
			To:   groupId,
			Type: "group",
			Body: alert.Message,
		})

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Message sent to " + groupId))
	}
}

func Run(
	ctx context.Context,
	ms MessageSender,
	wg *sync.WaitGroup,
) {
	httpMux := http.NewServeMux()

	httpMux.HandleFunc("GET /healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	httpMux.HandleFunc("POST /whatsapp/send/grafana-alert/user/{user_id}", sendNewGrafanaAlertWhatsAppMessageToUser(ms))
	httpMux.HandleFunc("POST /whatsapp/send/grafana-alert/group/{group_id}", sendNewGrafanaAlertWhatsAppMessageToGroup(ms))

	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
		})
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: loggingMiddleware(corsMiddleware(httpMux)),
	}

	go func() {
		defer wg.Done()
		fmt.Println("Starting server on :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
		fmt.Println("HTTP server stopped")
	}()

	go func() {
		defer wg.Done()
		<-ctx.Done()
		fmt.Println("Shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Error during server shutdown: %v\n", err)
		}
	}()
}

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
