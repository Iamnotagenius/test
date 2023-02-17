package main

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlersMap = map[string]func(*Session, *tgbotapi.Message) error

type Session struct {
	ChatID      int64
	Isu         int64
	ChatChannel chan *tgbotapi.Message
	Bot         *tgbotapi.BotAPI
	Handlers    HandlersMap
}

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

func (s *Session) WaitForNewMessage() (string, bool) {
	msg, ok := <-s.ChatChannel
	return msg.Text, ok
}

func (s *Session) SendMessage(msg string) {
	s.Bot.Send(tgbotapi.NewMessage(s.ChatID, msg))
}
