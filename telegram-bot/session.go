package main

import (
	"fmt"
	"log"

	"github.com/Iamnotagenius/test/db/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type handlersMap = map[string]func(*Session, *tgbotapi.Message) error

// Session represents a chat session associated with one user
type Session struct {
	ChatID      int64
	Isu         int64
	ChatChannel chan *tgbotapi.Message
	Bot         *tgbotapi.BotAPI
	Handlers    handlersMap
	DBClient    service.DatabaseTestClient
}

const (
	markdown   = "Markdown"
	markdownV2 = "MarkdownV2"
	html       = "HTML"
)

// Handle starts recieving commands on a given session
func (s *Session) Handle() {
	for msg := range s.ChatChannel {
		if !msg.IsCommand() {
			continue
		}

		handler, ok := s.Handlers[msg.Command()]
		if !ok {
			s.SendMessage("I don't know this command")
		}
		err := handler(s, msg)
		if err != nil {
			log.Printf("Error when handling '%v': %v", msg.Command(), err)
			s.SendMessage(fmt.Sprintf("%v, please try again", err))
			continue
		}
	}
}

// WaitForNewMessage waits for a new message from the same chat to arrive.
// ok is false when channel is closed.
// Returns content of a message
func (s *Session) WaitForNewMessage() (content string, ok bool) {
	msg, ok := <-s.ChatChannel
	return msg.Text, ok
}

// SendMessage sends a message to chat
func (s *Session) SendMessage(msg string) {
	s.Bot.Send(tgbotapi.NewMessage(s.ChatID, msg))
}

// SendMessageWithParseMode sends a message to chat specifying parse mode
func (s *Session) SendMessageWithParseMode(msg string, parseMode string) {
	msgConfig := tgbotapi.NewMessage(s.ChatID, msg)
	msgConfig.ParseMode = parseMode
	s.Bot.Send(msgConfig)
}
