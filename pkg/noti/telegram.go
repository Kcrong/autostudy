package noti

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	TelegramCommandPrefix = "/"

	CommandReport = "report"
	CommandRun    = "run"
)

var (
	validCommands = []string{CommandReport, CommandRun}
)

func IsValidCommand(c string) bool {
	for _, validCommand := range validCommands {
		if c == validCommand {
			return true
		}
	}
	return false
}

var (
	keyboardMarkup = func() tgbotapi.ReplyKeyboardMarkup {
		rows := make([]tgbotapi.KeyboardButton, len(validCommands))
		for i, c := range validCommands {
			rows[i] = tgbotapi.NewKeyboardButton(TelegramCommandPrefix + c)
		}

		return tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				rows...,
			),
		)
	}()
)

type TelegramBot struct {
	bot     *tgbotapi.BotAPI
	chatID  int64
	nowFunc func() time.Time
}

func (b TelegramBot) SendMessage(msg string) error {
	m := tgbotapi.NewMessage(b.chatID, msg)
	m.ReplyMarkup = keyboardMarkup
	_, err := b.bot.Send(m)
	return errors.Wrap(err, "b.bot.Send(tgbotapi.NewMessage(b.chatID, msg))")
}

func (b TelegramBot) SendPhoto(photo []byte) error {
	m := tgbotapi.NewPhoto(b.chatID, tgbotapi.FileBytes{
		Name:  b.nowFunc().String() + ".png",
		Bytes: photo,
	})
	m.ReplyMarkup = keyboardMarkup
	_, err := b.bot.Send(m)
	return errors.Wrap(err, "b.bot.Send(tgbotapi.NewPhoto(b.chatID, tgbotapi.FileBytes{...}))")
}

func (b TelegramBot) Updates() tgbotapi.UpdatesChannel {
	return b.bot.GetUpdatesChan(tgbotapi.UpdateConfig{
		Timeout: 60,
	})
}

func NewTelegramBot(token string, chatID int64, nowFunc func() time.Time) (*TelegramBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "tgbotapi.NewBotAPI(token)")
	}

	return &TelegramBot{
		bot:     bot,
		chatID:  chatID,
		nowFunc: nowFunc,
	}, nil
}
