// Telegram bot with authorization by ITMO.ID,
// searching users and ability to specify a phone number
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Iamnotagenius/test/db/server"
	"github.com/Iamnotagenius/test/db/service"
	"github.com/coreos/go-oidc/v3/oidc"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcDbServiceAddress = flag.String("db-service-addr", "localhost:50051", "Address of grpc DB service")
)

func main() {
	flag.Parse()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_API"))
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	provider, err := oidc.NewProvider(context.Background(), "https://id.itmo.ru/auth/realms/itmo")
	if err != nil {
		log.Panicf("Invalid provider: %v", err)
	}
	oauth2Config.Endpoint = provider.Endpoint()

	state, err := generateState()
	if err != nil {
		log.Panicf("Error when generating state: %v", err)
	}
	authChan := make(chan int64)
	http.Handle("/", &authHandler{
		bot:      bot,
		isuChan:  authChan,
		provider: provider,
		config:   oauth2Config,
		state:    state,
	})
	go http.ListenAndServe(":8080", nil)

	grpcConn, err := grpc.Dial(*grpcDbServiceAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Panicf("Failed to establish connection with database service: %v", err)
	}
	defer grpcConn.Close()
	dbClient := service.NewDatabaseTestClient(grpcConn)

	sessions := make(map[int64]Session)
	handlersMap := RegisterCommands(bot, commands...)
	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		log.Printf("CallbackData: %v", update.CallbackData())
		if msg := update.Message; msg != nil && msg.Chat.IsPrivate() {
			session, ok := sessions[msg.Chat.ID]
			if msg.Command() == "start" || !ok {
				currentSession, err := authentication(bot, msg, state, authChan)
				if err != nil {
					log.Printf("Authentication error: %v", err)
					continue
				}
				currentSession.Handlers = handlersMap
				currentSession.DBClient = dbClient
				sessions[msg.Chat.ID] = currentSession

				user, err := dbClient.GetUserByID(context.Background(), &service.UserByIDRequest{Id: currentSession.Isu})
				if err != nil {
					if err == server.ErrUserNotFound {
						user = &service.User{
							Id:   currentSession.Isu,
							Name: fmt.Sprintf("%v %v (%v)", msg.From.FirstName, msg.From.LastName, msg.From.UserName),
							Role: service.Role_ROLE_USER,
						}
					} else {
						log.Printf("Database error: %v", err)
					}
				}

				_, err = dbClient.AddOrUpdateUser(context.Background(), user)
				if err != nil {
					log.Printf("Error calling db service: %v", err)
				}

				go currentSession.Handle()

				continue
			}

			session.ChatChannel <- msg
		}
	}
}

func authentication(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, state string, authChan chan int64) (Session, error) {
	authURL := getAuthCodeURL(msg.Chat.ID, state)
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf(`Please authenticate with <a href="%v">this link</a>.`, authURL))
	msgConfig.ParseMode = html
	bot.Send(msgConfig)
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
	}, nil
}
