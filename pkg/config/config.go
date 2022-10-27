package config

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
)

const (
	EnvProduction  = "production"
	EnvDevelopment = "development"
)

type UrlConfig struct {
	Main      string
	MyProfile string
	Login     string
	Lecture   string
}

type Config struct {
	ENV        string
	CommitHash string

	SeleniumWebDriverHost string

	UnivID string
	UnivPW string
	Url    UrlConfig

	TelegramToken  string
	TelegramChatID int64

	SentryDSN string

	// IsProduction is true if ENV is EnvProduction
	IsProduction bool
	// Set to true if you want to run browser locally
	UseLocalBrowser  bool
	LocalBrowserPath string
	// Set to true if you want to run browser headless
	ShouldRunHeadless bool
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func NewConfig() (*Config, error) {
	env := getEnv("ENV", EnvDevelopment)
	isProd := env == EnvProduction

	chatID, err := strconv.ParseInt(getEnv("TELEGRAM_CHAT_ID", ""), 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid chat_id: %s", getEnv("TELEGRAM_CHAT_ID", ""))
	}

	useLocalBrowser := !isProd
	if v := getEnv("USE_LOCAL_BROWSER", ""); v != "" {
		useLocalBrowser, err = strconv.ParseBool(v)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid USE_LOCAL_BROWSER: %s", v)
		}
	}

	shouldRunHeadless := isProd
	if v := getEnv("SHOULD_RUN_HEADLESS", ""); v != "" {
		shouldRunHeadless, err = strconv.ParseBool(v)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid SHOULD_RUN_HEADLESS: %s", v)
		}
	}

	return &Config{
		ENV:                   env,
		CommitHash:            getEnv("COMMIT_HASH", "not-available"),
		SeleniumWebDriverHost: getEnv("SELENIUM_WEB_DRIVER_HOST", ""),
		UnivID:                getEnv("UNIV_ID", ""),
		UnivPW:                getEnv("UNIV_PW", ""),
		Url: UrlConfig{
			Main:      getEnv("URL_MAIN", ""),
			MyProfile: getEnv("URL_MY_PROFILE", ""),
			Login:     getEnv("URL_LOGIN", ""),
			Lecture:   getEnv("URL_LECTURE_PAGE", ""),
		},
		TelegramToken:     getEnv("TELEGRAM_API_TOKEN", ""),
		TelegramChatID:    chatID,
		SentryDSN:         getEnv("SENTRY_DSN", ""),
		IsProduction:      isProd,
		UseLocalBrowser:   useLocalBrowser,
		LocalBrowserPath:  getEnv("LOCAL_BROWSER_PATH", "./chromedriver"),
		ShouldRunHeadless: shouldRunHeadless,
	}, nil
}
