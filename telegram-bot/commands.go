package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/Iamnotagenius/test/db/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/grpc/status"
)

// Command represents a command handled by a bot
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
	{
		Name:        "search",
		Description: "Search users by part or whole name",
		Handler:     searchHandler,
	},
}

// RegisterCommands makes a request to notify about the declared commands
func RegisterCommands(bot *tgbotapi.BotAPI, cmds ...Command) handlersMap {
	botCmds := make([]tgbotapi.BotCommand, 0, len(cmds))
	botCmds = append(botCmds, tgbotapi.BotCommand{Command: "start", Description: "Initiate authentication sequence"})
	hMap := make(handlersMap)
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

	user, err := s.DBClient.GetUserByID(
		context.Background(),
		&service.UserByIDRequest{Id: s.Isu})
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	user.PhoneNumber = &phone
	s.DBClient.AddOrUpdateUser(context.Background(), user)
	return nil
}

func searchHandler(s *Session, msg *tgbotapi.Message) error {
	stream, err := s.DBClient.SearchUsersByName(context.Background(), &service.SearchByNameRequest{Query: msg.CommandArguments()})
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}

	tableString := &strings.Builder{}
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		phone := user.GetPhoneNumber()
		if phone == "" {
			phone = "Unset"
		}
		tableString.WriteString(fmt.Sprintf("\nISU: %v\nName: %v\nPhone Number: %v",
			strconv.FormatInt(user.GetId(), 10),
			user.GetName(),
			phone))
	}
	if tableString.Len() == 0 {
		s.SendMessage("No users found.")
		return nil
	}
	s.SendMessage(tableString.String()[1:])
	return nil
}
