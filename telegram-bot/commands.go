package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Command struct {
	Name        string
	Description string
	Handler     func(*Session, *tgbotapi.Message) error
}

var phoneRegex = regexp.MustCompile("\\+[0-9]+ \\([0-9]{3}\\) [0-9]{3}-[0-9]{2}-[0-9]{2}")

var commands = []Command{
	{
		Name:        "hello",
		Description: "Greet the user",
		Handler:     helloHandler,
	},
	{
		Name:        "phone",
		Description: "Add or set phone number for the user",
		Handler:     phoneHandler,
	},
}

func RegisterCommands(bot *tgbotapi.BotAPI, cmds ...Command) HandlersMap {
	botCmds := make([]tgbotapi.BotCommand, 0, len(cmds))
	botCmds = append(botCmds, tgbotapi.BotCommand{Command: "start", Description: "Initiate authentication sequence"})
	hMap := make(HandlersMap)
	for _, cmd := range cmds {
		botCmds = append(botCmds, tgbotapi.BotCommand{Command: cmd.Name, Description: cmd.Description})
		hMap[cmd.Name] = cmd.Handler
	}

	cmd := tgbotapi.NewSetMyCommands(botCmds...)
	if _, err := bot.Request(cmd); err != nil {
		log.Panic(err)
	}

	return hMap
}

func helloHandler(s *Session, msg *tgbotapi.Message) error {
	s.SendMessage(fmt.Sprintf("Hello, %v [%v]", msg.From.FirstName, s.Isu))
	return nil
}

func phoneHandler(s *Session, msg *tgbotapi.Message) error {
	phone := msg.CommandArguments()
	if phone == "" {
		s.SendMessage("Enter a phone number in format '+x (xxx) xxx-xx-xx'")
		var ok bool
		phone, ok = s.WaitForNewMessage()
		if !ok {
			return errors.New("Session was closed")
		}
	}
	if !phoneRegex.MatchString(phone) {
		return errors.New("Phone format did not match")
	}

	log.Printf("%v's [%v] phone: %v", msg.From.UserName, s.Isu, phone)

	return nil
}
