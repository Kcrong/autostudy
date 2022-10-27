package main

import (
	"math/rand"
	"time"
	_ "time/tzdata"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"

	"github.com/Kcrong/autostudy/pkg/config"
	"github.com/Kcrong/autostudy/pkg/driver"
	"github.com/Kcrong/autostudy/pkg/noti"
	"github.com/Kcrong/autostudy/pkg/univ"
)

func NewReportFunc(telegramBot *noti.TelegramBot) func(error, selenium.WebDriver) {
	return func(err error, wd selenium.WebDriver) {
		if err == nil {
			return
		}

		log.Errorf("%+v", err)
		sentry.CaptureException(err)

		if err := telegramBot.SendMessage("에러가 발생했습니다."); err != nil {
			sentry.CaptureException(err)
		}
		if err := telegramBot.SendMessage(err.Error()); err != nil {
			sentry.CaptureException(err)
		}

		if screenshot, err := wd.Screenshot(); err == nil {
			if err := telegramBot.SendPhoto(screenshot); err != nil {
				sentry.CaptureException(err)
			}
		}
	}
}

func main() {
	c, err := config.NewConfig()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	kst, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "time.LoadLocation(\"Asia/Seoul\")"))
	}

	nowFunc := func() time.Time {
		return time.Now().In(kst)
	}
	// Randomize seed.
	rand.Seed(nowFunc().Unix())

	if err := noti.InitSentry(c); err != nil {
		log.Fatalf("%+v", err)
	}
	defer sentry.Flush(2 * time.Second)

	bot, err := noti.NewTelegramBot(c.TelegramToken, c.TelegramChatID, nowFunc)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	reportFunc := NewReportFunc(bot)

	var opt *driver.InitOption
	if c.UseLocalBrowser {
		opt = &driver.InitOption{
			ShouldRunService: true,
			LocalBrowserPath: c.LocalBrowserPath,
		}
	}

	for range time.Tick(time.Millisecond) {
		wd, closeFunc, err := driver.Init(c.SeleniumWebDriverHost, c.ShouldRunHeadless, opt)
		if err != nil {
			log.Fatalf("%+v", err)
		}

		if err := univ.Login(wd, c.Url.Main, c.UnivID, c.UnivPW, &c.Url.MyProfile); err != nil {
			reportFunc(err, wd)
		}
		if _, err := univ.GetSubjects(c.Url.Lecture, wd, true, univ.NewWatchFunc(wd, c.Url.Lecture, bot)); err != nil {
			reportFunc(err, wd)
		}

		reportFunc(closeFunc(), wd)
		sentry.Flush(2 * time.Second)
	}
}
