package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	authMutex = sync.Mutex{}
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_API"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	cmd := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "Initiate authentication sequence"},
	)
	if _, err := bot.Request(cmd); err != nil {
		log.Panic(err)
	}

	provider, err := oidc.NewProvider(context.Background(), "https://id.itmo.ru/auth/realms/itmo")

	if err != nil {
		log.Panicf("Invalid provider: %v", err)
	}
	oauth2Config.Endpoint = provider.Endpoint()

	state, err := generateState()
	if err != nil {
		log.Panicf("Error when generating state: %v", err)
	}
	sessions := make(map[int64]int64)
	authChan := make(chan int64)
	http.Handle("/", &AuthHandler{
		bot:      bot,
		isuChan:  authChan,
		provider: provider,
		config:   oauth2Config,
		sessions: sessions,
		state:    state,
	})

	go http.ListenAndServe(":8080", nil)

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		log.Printf("Sessions: %v", sessions)
		if msg := update.Message; msg != nil {
			newMsg := tgbotapi.NewMessage(msg.Chat.ID, "")
			switch msg.Command() {
			case "help":
				newMsg.Text = "I know /help and /hello"
			case "hello":
				newMsg.Text = fmt.Sprintf("Hello, %v", msg.From.UserName)
				break
			case "start":
				authURL, err := GetAuthCodeURL(msg.Chat.ID, state)
				if err != nil {
					log.Panicf("Failed to get auth code url: %v", err)
				}

				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Please authenticate with this link %v", authURL)))
				var isu int64
				select {
				case isu = <-authChan:
				case <-time.After(time.Minute):
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Authentication timed out. Please, try again by issuing /start."))
					continue
				}
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Hello with isu number %v. I added you to my database.", isu)))
				continue
			default:
				newMsg.Text = "I don't know this"
			}

			bot.Send(newMsg)
		}
	}
}
