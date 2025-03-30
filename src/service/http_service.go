package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/optiop/grafana-whatsapp-webhook/entity"
	"github.com/optiop/grafana-whatsapp-webhook/whatsapp"
)

var appToken = os.Getenv("WHATSAPP_APP_TOKEN")

func sendNewGrafanaAlertWhatsAppMessageToUser(ws *whatsapp.WhatsappService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if token != appToken {
			http.Error(w, "token is required", http.StatusBadRequest)
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

		var alert GrafanaAlert
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			http.Error(w, "Error decoding alert", http.StatusBadRequest)
			return
		}

		if alert.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		message := entity.Message{
			To:   phoneNumber,
			Type: "user",
			Body: alert.Message,
		}

		ws.SendNewWhatsAppMessageToUser(message)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Message sent to " + message.To))
	}
}

func sendNewGrafanaAlertWhatsAppMessageToGroup(ws *whatsapp.WhatsappService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if token != appToken {
			http.Error(w, "token is required", http.StatusBadRequest)
			return
		}

		groupId := r.PathValue("group_id")
		if groupId == "" {
			http.Error(w, "group_id is required", http.StatusBadRequest)
			return
		}

		var alert GrafanaAlert
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			http.Error(w, "Error decoding alert", http.StatusBadRequest)
			return
		}

		if alert.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		message := entity.Message{
			To:   groupId,
			Type: "group",
			Body: alert.Message,
		}

		ws.SendNewWhatsAppMessageToGroup(message)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Message sent to " + message.To))
	}
}

func Run(
	ctx context.Context,
	ws *whatsapp.WhatsappService,
	wg *sync.WaitGroup,
) {
	httpMux := http.NewServeMux()

	httpMux.HandleFunc("GET /healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	httpMux.HandleFunc("POST /whatsapp/send/grafana-alert/user/{user_id}/{token}", sendNewGrafanaAlertWhatsAppMessageToUser(ws))
	httpMux.HandleFunc("POST /whatsapp/send/grafana-alert/group/{group_id}/{token}", sendNewGrafanaAlertWhatsAppMessageToGroup(ws))
	
	// Apply CORS middleware for all routes
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}

	// Wrap the HTTP mux with the CORS middleware
	server := &http.Server{
		Addr:    ":8080",
		Handler: corsMiddleware(httpMux),
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
