package noti

import (
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

type TelegramBot struct {
	bot     *tgbotapi.BotAPI
	chatID  int64
	nowFunc func() time.Time
}

func (b TelegramBot) SendMessage(msg string) error {
	_, err := b.bot.Send(tgbotapi.NewMessage(b.chatID, msg))
	return errors.Wrap(err, "b.bot.Send(tgbotapi.NewMessage(b.chatID, msg))")
}

func (b TelegramBot) SendPhoto(photo []byte) error {
	_, err := b.bot.Send(tgbotapi.NewPhoto(b.chatID, tgbotapi.FileBytes{
		Name:  b.nowFunc().String() + ".png",
		Bytes: photo,
	}))
	return errors.Wrap(err, "b.bot.Send(tgbotapi.NewPhoto(b.chatID, tgbotapi.FileBytes{...}))")
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
