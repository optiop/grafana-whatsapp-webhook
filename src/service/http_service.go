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

var appToken = os.Getenv("APP_TOKEN")

func sendNewWhatsAppMessage(ws *whatsapp.WhatsappService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if token != appToken {
			http.Error(w, "token is required", http.StatusBadRequest)
			return
		}

		var message entity.Message
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			_, _ = w.Write([]byte("Error decoding message"))
			return
		}

		if message.To == "" || message.Body == "" {
			http.Error(w, "to and body are required", http.StatusBadRequest)
			return
		}

		if message.Type == "user" {
			ws.SendNewWhatsAppMessageToUser(message)
		} else if message.Type == "group" {
			ws.SendNewWhatsAppMessageToGroup(message)
		} else {
			http.Error(w, "type must be user or group", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Message sent to " + message.To))
	}
}

func sendNewGrafanaAlertWhatsAppMessage(ws *whatsapp.WhatsappService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if token != appToken {
			http.Error(w, "token is required", http.StatusBadRequest)
			return
		}

		var alert GrafanaAlert
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			_, _ = w.Write([]byte("Error decoding alert"))
			return
		}

		if alert.CommonLabels.Phone == "" {
			http.Error(w, "phone is required", http.StatusBadRequest)
			return
		}

		if alert.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		phone := alert.CommonLabels.Phone
		if phone[0] == '+' {
			phone = phone[1:]
		}

		message := entity.Message{
			To:   phone,
			Body: alert.Message,
			Type: "user",
		}

		ws.SendNewWhatsAppMessageToUser(message)

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
	httpMux.HandleFunc("POST /whatsapp/send/message/{token}", sendNewWhatsAppMessage(ws))
	httpMux.HandleFunc("POST /whatsapp/send/grafana-alert/{token}", sendNewGrafanaAlertWhatsAppMessage(ws))

	server := &http.Server{
		Addr:    ":8080",
		Handler: httpMux,
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
