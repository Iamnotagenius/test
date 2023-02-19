// Telegram bot with authorization by ITMO.ID,
// searching users and ability to specify a phone number
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

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

	sessions := make(map[int64]Session)
	grpcConn, err := grpc.Dial(*grpcDbServiceAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Panicf("Failed to establish connection with database service: %v", err)
	}
	http.Handle("/", &authHandler{
		sessions:    sessions,
		provider:    provider,
		state:       state,
		handlersMap: RegisterCommands(bot, commands...),
		dbClient:    service.NewDatabaseTestClient(grpcConn),
		bot:         bot,
	})
	go http.ListenAndServe(":8080", nil)

	defer grpcConn.Close()

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if msg := update.Message; msg != nil && msg.Chat.IsPrivate() {
			session, ok := sessions[msg.Chat.ID]
			if msg.Command() == "start" || !ok {
				authentication(bot, msg, state, msg.From.FirstName, msg.From.LastName)
				continue
			}

			session.ChatChannel <- msg
		}
	}
}
