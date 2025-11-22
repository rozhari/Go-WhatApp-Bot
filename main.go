package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "gobot/plugins"

	_ "github.com/mattn/go-sqlite3"

	"gobot/lib"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var startTime = time.Now()
var syncCompleted = false
var syncMutex sync.Mutex

func main() {
	lib.LoadConfig()
	ctx := context.Background()
	dbLog := waLog.Noop
	container, err := sqlstore.New(ctx, "sqlite3", "file:auth.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Noop
	fmt.Println("Connecting to WhatsApp...")
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		fmt.Println("Connected")
		if err != nil {
			panic(err)
		}
	}

	if client.Store.ID != nil {
		time.Sleep(5 * time.Second)
		sudos := strings.Split(lib.Config.SUDO, ",")
		sudo := sudos[0]
		if sudo == "" {
			sudo = client.Store.ID.User
		}
		jid := types.NewJID(sudo, types.DefaultUserServer)
		prefix := lib.GetPrefix()
		client.SendMessage(context.Background(), jid, &waE2E.Message{
			Conversation: proto.String(fmt.Sprintf("*BOT CONNECTED*\n\n```PREFIX : %s\nPLUGINS : %d\nVERSION : %s```", prefix, len(lib.Commands), "1.0.0")),
		})
	}

	lib.Client = client
	lib.StartTime = startTime

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

func eventHandler(evt interface{}) {
	switch v := evt.(type) {

	case *events.OfflineSyncPreview:
		fmt.Printf("\n\x1b[36m[Offline Sync Preview]\x1b[39m\n")
		fmt.Printf("Messages: %d\n", v.Messages)
		fmt.Printf("Notifications: %d\n", v.Notifications)
		fmt.Printf("Receipts: %d\n", v.Receipts)
		fmt.Println("\x1b[33mWaiting for offline sync to complete...\x1b[39m")

	case *events.OfflineSyncCompleted:
		syncMutex.Lock()
		syncCompleted = true
		syncMutex.Unlock()
		fmt.Println("\x1b[32m[Offline Sync Completed] - Bot is now ready to process commands\x1b[39m")

	case *events.Message:
		handleMessage(v)
	}
}

func handleMessage(evt *events.Message) {
	ctx := context.Background()
	if evt.Message == nil {
		return
	}

	syncMutex.Lock()
	isSyncCompleted := syncCompleted
	syncMutex.Unlock()

	message := lib.NewMessage(lib.Client, evt)

	if lib.Config.LOG_MSG {
		fmt.Printf("[%s] : %s\n", message.PushName, message.Text)
	}

	if !isSyncCompleted {
		return
	}

	if lib.Config.READ_MSG && evt.Info.Chat.Server != types.BroadcastServer {
		lib.Client.MarkRead(ctx, []types.MessageID{evt.Info.ID}, time.Now(), evt.Info.Chat, evt.Info.Sender)
	}

	for _, command := range lib.Commands {
		isMatch := false
		if command.On != "" {
			switch command.On {
			case "image":
				isMatch = message.Type == "imageMessage"
			case "video":
				isMatch = message.Type == "videoMessage"
			case "sticker":
				isMatch = message.Type == "stickerMessage"
			case "audio":
				isMatch = message.Type == "audioMessage"
			case "text":
				isMatch = message.Text != ""
			case "message":
				isMatch = true
			}
		} else if command.Pattern != nil {
			isMatch = command.Pattern.MatchString(message.Text)
		}

		if isMatch {
			if command.FromMe && !message.FromMe {
				continue
			}
			if command.OnlyGroup && !message.IsGroup {
				continue
			}
			if command.OnlyPm && !message.IsPm {
				continue
			}


			if command.Pattern != nil && lib.Config.READ_CMD {
				lib.Client.MarkRead(ctx, []types.MessageID{evt.Info.ID}, time.Now(), evt.Info.Chat, evt.Info.Sender)
			}

			match := ""
			if command.Pattern != nil {
				matches := command.Pattern.FindStringSubmatch(message.Text)
				if len(matches) >= 2 {
					if len(matches) == 6 {
						if matches[3] != "" {
							match = matches[3]
						} else {
							match = matches[4]
						}
					} else {
						if matches[2] != "" {
							match = matches[2]
						} else if len(matches) > 3 {
							match = matches[3]
						}
					}
				}
			}

			go func() {
				defer func() {
					if r := recover(); r != nil {
						if lib.Config.ERROR_MSG {
							fmt.Println("Error:", r)
							sudos := strings.Split(lib.Config.SUDO, ",")
							sudo := sudos[0]
							if sudo == "" {
								sudo = lib.Client.Store.ID.User
							}
							jid := types.NewJID(sudo, types.DefaultUserServer)
							lib.Client.SendMessage(context.Background(), jid, &waE2E.Message{
								Conversation: proto.String(fmt.Sprintf("```─━❲ ERROR REPORT ❳━─\n\nMessage : %s\nError : %v\nJid : %s```", message.Text, r, message.Chat.String())),
							})
						}
					}
				}()
				command.Function(message, match)
			}()
		}
	}
}