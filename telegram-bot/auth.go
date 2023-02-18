package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/coreos/go-oidc/v3/oidc"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/oauth2"
)

type authHandler struct {
	bot      *tgbotapi.BotAPI
	isuChan  chan int64
	config   oauth2.Config
	provider *oidc.Provider
	state    string
}

type stateWithChatID struct {
	base   string
	chatID int64
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

func newState(base string, chatID int64) (s stateWithChatID, err error) {
	if len(base) != stateLength {
		return stateWithChatID{}, fmt.Errorf(
			"Invalid length of base string, should be %v, got %v",
			stateLength,
			len(base))
	}
	s.base = base
	s.chatID = chatID
	return s, nil
}

// Have to use "state" for chatID because custom parameters don't persist across redirect
func (s stateWithChatID) encode() string {
	return s.base + strconv.FormatInt(s.chatID, chatIDEncodingBase)
}

func decodeState(encoded string, baseLength int) (s stateWithChatID, err error) {
	s.base = encoded[:baseLength]
	s.chatID, err = strconv.ParseInt(encoded[baseLength:], chatIDEncodingBase, 64)
	return
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Handle redirect")
	if r.URL.Query().Get("code") == "" {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "No code in url")
		return
	}

	token, err := h.config.Exchange(r.Context(), r.URL.Query().Get("code"))
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

	h.isuChan <- claims.Isu
}

func getAuthCodeURL(chatID int64, state string) string {
	return oauth2Config.AuthCodeURL(stateWithChatID{base: state, chatID: chatID}.encode())
}

func generateState() (string, error) {
	buffer := make([]byte, stateLength)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
