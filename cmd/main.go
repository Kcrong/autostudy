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

func NewReportFunc(telegramBot *tgbotapi.BotAPI, chatID int64, nowFunc func() time.Time) func(error, selenium.WebDriver) {
	return func(err error, wd selenium.WebDriver) {
		if err == nil {
			return
		}

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

func main() {
	c, err := config.NewConfig()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	kst, _ := time.LoadLocation("Asia/Seoul")
	nowFunc := func() time.Time {
		return time.Now().In(kst)
	}

	// Randomize seed.
	rand.Seed(nowFunc().Unix())

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

	reportFunc := NewReportFunc(bot, c.TelegramChatID, nowFunc)

	var opt *driver.InitOption
	if c.UseLocalBrowser {
		opt = &driver.InitOption{
			ShouldRunService: true,
			LocalBrowserPath: c.LocalBrowserPath,
		}
	}

	for range time.Tick(time.Hour) {
		wd, closeFunc, err := driver.Init(c.SeleniumWebDriverHost, c.ShouldRunHeadless, opt)
		if err != nil {
			log.Fatalf("%+v", err)
		}

		if err := univ.Login(wd, c.Url.Main, c.UnivID, c.UnivPW, &c.Url.MyProfile); err != nil {
			reportFunc(err, wd)
		}
		if _, err := univ.GetSubjects(c.Url.Lecture, wd, true, univ.NewWatchFunc(wd, c.Url.Lecture)); err != nil {
			reportFunc(err, wd)
		}

		reportFunc(closeFunc(), wd)
		sentry.Flush(2 * time.Second)
	}
}
