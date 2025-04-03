package whatsapp

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"github.com/optiop/grafana-whatsapp-webhook/pkg/entity"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type WhatsappService struct {
	client *whatsmeow.Client

	cUserMessage,
	cGroupMessage chan entity.Message
}

func New(ctx context.Context, wg *sync.WaitGroup) *WhatsappService {
	if _, err := os.Stat("data"); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir("data", os.ModePerm); err != nil && !os.IsExist(err) {
			log.Fatal(err)
		}
	}

	wa := &WhatsappService{
		cUserMessage:  make(chan entity.Message, 1024),
		cGroupMessage: make(chan entity.Message, 1024),
	}

	wa.setupWhatsappService(ctx, wg)

	return wa
}

func (*WhatsappService) eventHandler(evt any) {
	switch v := evt.(type) {
	case *events.Message:
		log.Println("Received a message: ", v.Message.GetConversation())
	}
}

// setupWhatsappService initializes and sets up the WhatsApp service for the given WhatsappService instance.
// It configures logging, connects to the database, retrieves the device store, and establishes a connection
// to the WhatsApp client. It also handles QR code generation for new logins and retrieves the list of joined groups.
// Additionally, it starts goroutines to handle sending user and group messages and disconnecting the client.
//
// Parameters:
//   - ctx: The context for managing the lifecycle of the service.
//   - wg: A wait group to synchronize the shutdown process.
//
// The function performs the following steps:
//  1. Configures logging based on the APP_DEBUG environment variable.
//  2. Connects to the SQLite database and retrieves the device store.
//  3. Initializes the WhatsApp client and sets up event handlers.
//  4. Handles QR code generation for new logins if the client is not already authenticated.
//  5. Connects the client to the WhatsApp service.
//  6. Retrieves and logs the list of joined groups.
//  7. Starts goroutines for handling user messages, group messages, and client disconnection.
//
// If any error occurs during the setup process, the function logs the error and panics.
func (ws *WhatsappService) setupWhatsappService(
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	debug := strings.ToLower(os.Getenv("APP_DEBUG")) == "true"
	level := "INFO"

	if debug {
		level = "DEBUG"
	}

	dbLog := waLog.Stdout("Database", level, true)
	container, err := sqlstore.New("sqlite3", "data/sqlite3.db?_foreign_keys=on", dbLog)
	if err != nil {
		log.Panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		log.Panic(err)
	}

	clientLog := waLog.Stdout("Client", level, true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(ws.eventHandler)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			log.Panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				if _, err := os.Stat("out"); errors.Is(err, os.ErrNotExist) {
					if err := os.Mkdir("out", os.ModePerm); err != nil && !os.IsExist(err) {
						log.Println("Error creating 'out' directory: ", err)
					}
				}
				err := qrcode.WriteFile(evt.Code, qrcode.Medium, 256, "out/qr.png")
				if err != nil {
					log.Println("Error: to create qr code png file: ", err)
				}
			} else {
				log.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			log.Panic(err)
		}
	}

	groups, err := client.GetJoinedGroups()
	if err != nil {
		log.Panic(err)
	}

	log.Println("Joined groups:")
	for _, group := range groups {
		log.Println("Name: ", group.Name)
		log.Println("Jid: ", group.JID)
		log.Println("----------------")
	}

	go ws.handelSendUserMessages(ctx)
	go ws.handelSendGroupMessages(ctx)
	go ws.disconnect(ctx, wg)

	ws.client = client
}

func (ws *WhatsappService) SendNewWhatsAppMessageToUser(msg entity.Message) {
	ws.cUserMessage <- msg
}

func (ws *WhatsappService) SendNewWhatsAppMessageToGroup(msg entity.Message) {
	ws.cGroupMessage <- msg
}

func (ws *WhatsappService) handelSendUserMessages(ctx context.Context) {
	for msg := range ws.cUserMessage {
		to := types.NewJID(msg.To, "s.whatsapp.net")
		message := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: &msg.Body,
			},
		}

		_, err := ws.client.SendMessage(ctx, to, message)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (ws *WhatsappService) handelSendGroupMessages(ctx context.Context) {
	for msg := range ws.cGroupMessage {
		groupJID := types.NewJID(msg.To, "g.us")
		message := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: &msg.Body,
			},
		}

		_, err := ws.client.SendMessage(ctx, groupJID, message)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (ws *WhatsappService) disconnect(
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	<-ctx.Done()
	close(ws.cUserMessage)
	close(ws.cGroupMessage)

	<-time.After(3 * time.Second)

	ws.client.Disconnect()
}
