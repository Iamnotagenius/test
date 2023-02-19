package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Iamnotagenius/test/db/service"
	"github.com/coreos/go-oidc/v3/oidc"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authHandler struct {
	sessions    map[int64]Session
	provider    *oidc.Provider
	state       string
	bot         *tgbotapi.BotAPI
	handlersMap handlersMap
	dbClient    service.DatabaseTestClient
}

type stateWithParams struct {
	base      string
	chatID    int64
	firstName string
	lastName  string
}

const (
	stateLength        = 25
	callbackEndpoint   = "/auth/itmoid/callback"
	chatIDEncodingBase = 16
)

var (
	oauth2Config = oauth2.Config{
		ClientID:     os.Getenv("ITMOID_CLIENT_ID"),
		ClientSecret: os.Getenv("ITMOID_CLIENT_SECRET"),
		Scopes:       []string{oidc.ScopeOpenID},
		RedirectURL:  "http://localhost:8080/",
	}
)

// Have to use "state" for chatID because custom parameters don't persist across redirect
func (s stateWithParams) encodeState() string {
	return s.base + strconv.FormatInt(s.chatID, chatIDEncodingBase) + ":" + s.firstName + ":" + s.lastName
}

func decodeState(encoded string, baseLength int) (s stateWithParams, err error) {
	s.base = encoded[:baseLength]
	rem := strings.Split(encoded[baseLength:], ":")
	s.chatID, err = strconv.ParseInt(rem[0], chatIDEncodingBase, 64)
	s.firstName = rem[1]
	s.lastName = rem[2]
	return
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Handle redirect")
	if r.URL.Query().Get("code") == "" {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "No code in url")
		return
	}

	token, err := oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("Exchange failed: %v", err)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Panicln("Missing token")
		return
	}

	idToken, err := h.provider.Verifier(
		&oidc.Config{ClientID: os.Getenv("ITMOID_CLIENT_ID")}).Verify(r.Context(), rawIDToken)
	if err != nil {
		log.Printf("Token parse failed: %v", err)
		return
	}
	combinedState, err := decodeState(r.URL.Query().Get("state"), len(h.state))
	if err != nil {
		log.Printf("Error decoding state: %v", err)
		return
	}

	if h.state != combinedState.base {
		log.Println("States did not match. CSRF attack?")
		return
	}

	var claims struct {
		Sub string `json:"sub"`
		Isu int64  `json:"isu"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("Token unmarshal failed: %v", err)
		return
	}

	if err != nil {
		log.Printf("Error parsing chat_id: %v", err)
		return
	}

	currentSession := Session{
		ChatID:      combinedState.chatID,
		Isu:         claims.Isu,
		ChatChannel: make(chan *tgbotapi.Message),
		Bot:         h.bot,
	}
	h.sessions[combinedState.chatID] = currentSession
	h.bot.Send(tgbotapi.NewMessage(combinedState.chatID, fmt.Sprintf("Hello with isu number %v.", claims.Isu)))

	currentSession.Handlers = h.handlersMap
	currentSession.DBClient = h.dbClient

	user, err := h.dbClient.GetUserByID(context.Background(), &service.UserByIDRequest{Id: currentSession.Isu})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			user = &service.User{
				Id:   currentSession.Isu,
				Name: fmt.Sprintf("%v %v", combinedState.firstName, combinedState.lastName),
				Role: service.Role_ROLE_USER,
			}
		} else {
			log.Printf("Database error: %v", err)
		}
	}

	_, err = h.dbClient.AddOrUpdateUser(context.Background(), user)
	if err != nil {
		log.Printf("Error calling db service: %v", err)
	}

	go currentSession.Handle()

}

func authentication(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, state string, firstName string, lastName string) {
	authURL := getAuthCodeURL(msg.Chat.ID, state, firstName, lastName)
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf(`Please authenticate with <a href="%v">this link</a>.`, authURL))
	msgConfig.ParseMode = html
	bot.Send(msgConfig)
}

func getAuthCodeURL(chatID int64, state string, firstName string, lastName string) string {
	return oauth2Config.AuthCodeURL(stateWithParams{
		base:      state,
		chatID:    chatID,
		firstName: firstName,
		lastName:  lastName,
	}.encodeState())
}

func generateState() (string, error) {
	buffer := make([]byte, stateLength)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
