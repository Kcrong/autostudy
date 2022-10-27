package driver

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

const (
	defaultDriverServicePort = 4444
)

type WaitFuncType func(selenium.Condition) error

type InitOption struct {
	ShouldRunService bool
	LocalBrowserPath string
}

func Init(path string, shouldRunHeadless bool, opt *InitOption) (selenium.WebDriver, func() error, error) {
	closeFunc := func() error { return nil }

	if opt != nil && opt.ShouldRunService {
		service, err := selenium.NewChromeDriverService(opt.LocalBrowserPath, defaultDriverServicePort)
		if err != nil {
			return nil, nil, errors.Wrap(err, "selenium.NewChromeDriverService")
		}

		closeFunc = appendFunc(service.Stop, closeFunc)
	}

	driver, err := selenium.NewRemote(defaultChromeCaps(shouldRunHeadless), path)
	if err != nil {
		return nil, closeFunc, errors.Wrap(err, "selenium.NewRemote")
	}

	closeFunc = appendFunc(driver.Quit, closeFunc)

	return driver, closeFunc, nil
}

// AssertUrl is asserting that the driver located at the given url.
func AssertUrl(url string, wd selenium.WebDriver) error {
	currentUrl, err := wd.CurrentURL()
	if err != nil {
		return errors.Wrap(err, "wd.CurrentURL")
	}

	// NOTE: This is a workaround for ignoring query strings.
	if !strings.Contains(currentUrl, url) {
		if err := wd.Get(url); err != nil {
			return errors.Wrap(err, fmt.Sprintf("wd.Get(%s)", url))
		}
	}

	return nil
}

func WaitElement(wd selenium.WebDriver, by, selector string) error {
	return errors.Wrap(wd.Wait(func(wd selenium.WebDriver) (bool, error) {
		_, err := wd.FindElement(by, selector)
		return err == nil, nil
	}), "wd.WaitElement")
}

func WaitAndFindElement(wd selenium.WebDriver, by, selector string) (selenium.WebElement, error) {
	if err := WaitElement(wd, by, selector); err != nil {
		return nil, err
	}

	return wd.FindElement(by, selector)
}

func IsNoSuchElementError(err error) bool {
	if err == nil {
		return false
	}

	err, ok := err.(*selenium.Error)
	if !ok {
		return false
	}

	return err.(*selenium.Error).Err == "no such element"
}

func appendFunc(funcs ...func() error) func() error {
	return func() error {
		for _, f := range funcs {
			if err := f(); err != nil {
				return err
			}
		}

		return nil
	}
}

func defaultChromeCaps(shouldHeadless bool) selenium.Capabilities {
	args := []string{
		"window-size=1920x1080",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"disable-gpu",
		"user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.87 Safari/537.36",
	}

	if shouldHeadless {
		args = append(args, "--headless")
	}

	caps := selenium.Capabilities{}
	caps.AddChrome(chrome.Capabilities{Args: args})
	return caps
}
