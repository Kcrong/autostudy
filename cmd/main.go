package main

import (
	"math/rand"
	"time"

	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"

	"github.com/Kcrong/autostudy/pkg/config"
	"github.com/Kcrong/autostudy/pkg/driver"
	"github.com/Kcrong/autostudy/pkg/univ"
)

func NewReportFunc(telegramBot *tgbotapi.BotAPI, chatID int64, wd selenium.WebDriver, nowFunc func() time.Time) func(error) {
	return func(err error) {
		log.Errorf("%+v", err)

		sentry.CaptureException(err)

		if _, err := telegramBot.Send(tgbotapi.NewMessage(chatID, "에러가 발생했습니다.")); err != nil {
			sentry.CaptureException(err)
		}
		if _, err := telegramBot.Send(tgbotapi.NewMessage(chatID, err.Error())); err != nil {
			sentry.CaptureException(err)
		}

		if screenshot, err := wd.Screenshot(); err == nil {
			_, _ = telegramBot.Send(tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
				Name:  nowFunc().String() + ".png",
				Bytes: screenshot,
			}))
		}
	}
}

func LetsStudy(c config.Config, wd selenium.WebDriver, reportFunc func(error)) {
	if err := univ.Login(wd, c.Url.Main, c.UnivID, c.UnivPW, &c.Url.MyProfile); err != nil {
		reportFunc(err)
	}

	if _, err := univ.GetSubjects(c.Url.Lecture, wd, true, univ.NewWatchFunc(wd, c.Url.Lecture)); err != nil {
		reportFunc(err)
	}
}

func main() {
	c, err := config.NewConfig()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// Randomize seed.
	rand.Seed(time.Now().Unix())

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              c.SentryDSN,
		Environment:      c.ENV,
		Release:          c.CommitHash,
		Debug:            !c.IsProduction,
		TracesSampleRate: 1,
	}); err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "sentry.Init()"))
	}
	defer sentry.Flush(2 * time.Second)

	bot, err := tgbotapi.NewBotAPI(c.TelegramToken)
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "tgbotapi.NewBotAPI()"))
	}

	var opt *driver.InitOption
	if c.UseLocalBrowser {
		opt = &driver.InitOption{
			ShouldRunService: true,
			LocalBrowserPath: c.LocalBrowserPath,
		}
	}
	wd, closeFunc, err := driver.Init(c.SeleniumWebDriverHost, c.ShouldRunHeadless, opt)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	reportFunc := NewReportFunc(bot, c.TelegramChatID, wd, time.Now)

	defer func() {
		if err := closeFunc(); err != nil {
			reportFunc(err)
		}
	}()

	for range time.Tick(time.Hour) {
		if _, err := bot.Send(tgbotapi.NewMessage(c.TelegramChatID, "Start")); err != nil {
			log.Fatal(errors.Wrap(err, "bot.Send()"))
		}

		LetsStudy(*c, wd, reportFunc)
		sentry.Flush(2 * time.Second)
	}
}
