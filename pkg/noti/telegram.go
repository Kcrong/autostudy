package noti

import (
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	CommandReport = "report"
	CommandRun    = "run"
)

func IsValidCommand(c string) bool {
	switch c {
	case CommandReport, CommandRun:
		return true
	default:
		return false
	}
}

var (
	keyboardMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/"+CommandRun),
			tgbotapi.NewKeyboardButton("/"+CommandReport),
		),
	)
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
