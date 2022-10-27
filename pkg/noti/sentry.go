package noti

import (
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"

	"github.com/Kcrong/autostudy/pkg/config"
)

func InitSentry(c config.Config) error {
	return errors.Wrap(sentry.Init(sentry.ClientOptions{
		Dsn:              c.SentryDSN,
		Environment:      c.ENV,
		Release:          c.CommitHash,
		Debug:            !c.IsProduction,
		TracesSampleRate: 1,
	}), "sentry.Init")
}
