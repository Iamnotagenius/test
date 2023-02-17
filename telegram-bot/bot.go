package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

	handlersMap := RegisterCommands(bot, commands...)

	provider, err := oidc.NewProvider(context.Background(), "https://id.itmo.ru/auth/realms/itmo")

	if err != nil {
		log.Panicf("Invalid provider: %v", err)
	}
	oauth2Config.Endpoint = provider.Endpoint()

	state, err := generateState()
	if err != nil {
		log.Panicf("Error when generating state: %v", err)
	}
	sessions := make(map[int64]Session)
	authChan := make(chan int64)
	http.Handle("/", &AuthHandler{
		bot:      bot,
		isuChan:  authChan,
		provider: provider,
		config:   oauth2Config,
		state:    state,
	})

	go http.ListenAndServe(":8080", nil)

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		log.Printf("CallbackData: %v", update.CallbackData())
		if msg := update.Message; msg != nil && msg.Chat.IsPrivate() {
			session, ok := sessions[msg.Chat.ID]
			if msg.Command() == "start" || !ok {
				currentSession, err := Authentication(bot, msg, state, authChan, handlersMap)
				if err != nil {
					log.Printf("Authentication error: %v", err)
					continue
				}
				sessions[msg.Chat.ID] = currentSession
				go currentSession.Handle()
				continue
			}

			session.ChatChannel <- msg
		}
	}
}

func Authentication(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, state string, authChan chan int64, hMap HandlersMap) (Session, error) {
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
		return Session{}, errors.New("Authentication timeout")
	}
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Hello with isu number %v. I added you to my database.", isu)))

	return Session{
		ChatID:      msg.Chat.ID,
		Isu:         isu,
		ChatChannel: make(chan *tgbotapi.Message),
		Bot:         bot,
		Handlers:    hMap,
	}, nil
}
