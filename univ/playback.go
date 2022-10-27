package univ

import (
	"math/rand"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tebeka/selenium"

	"github.com/Kcrong/autostudy/driver"
)

type WatchFuncType func(selenium.WebElement, bool, bool) error

func NewWatchFunc(wd selenium.WebDriver, url string) WatchFuncType {
	return func(button selenium.WebElement, hasPlayed, hasQuiz bool) error {
		if err := driver.AssertUrl(url, wd); err != nil {
			return err
		}

		if err := button.Click(); err != nil {
			return errors.Wrap(err, "button.Click()")
		}

		_ = wd.Wait(func(wd selenium.WebDriver) (bool, error) {
			handles, err := wd.WindowHandles()
			if err != nil {
				return false, errors.Wrap(err, "wd.CurrentWindowHandle()")
			}
			return len(handles) == 2, nil
		})

		handles, err := wd.WindowHandles()
		if err != nil {
			return errors.Wrap(err, "wd.CurrentWindowHandle()")
		}
		mainWindowHandle, lectureWindowHandle := handles[0], handles[1]

		if err := wd.SwitchWindow(lectureWindowHandle); err != nil {
			return errors.Wrap(err, "wd.SwitchWindow(lectureWindowHandle)")
		}

		if !hasPlayed {
			if err := play(wd); err != nil {
				return err
			}
		}
		if hasQuiz {
			if err := solveQuiz(wd); err != nil {
				return err
			}
		}

		return closeLectureWindow(wd, mainWindowHandle)
	}
}

func solveQuiz(wd selenium.WebDriver) error {
	examElement, err := driver.WaitAndFindElement(wd, selenium.ByClassName, "exam")
	if err != nil {
		return err
	}

	examListElement, err := examElement.FindElement(selenium.ByClassName, "exam-number")
	if err != nil {
		return errors.Wrap(err, "examElement.FindElement(selenium.ByClassName, \"exam-number\")")
	}
	examNumberItems, err := examListElement.FindElements(selenium.ByTagName, "li")
	if err != nil {
		return errors.Wrap(err, "examListElement.FindElements(selenium.ByTagName, \"li\")")
	}

	formElements, err := examElement.FindElements(selenium.ByTagName, "form")
	if err != nil {
		return errors.Wrap(err, "examListElement.FindElements(selenium.ByTagName, \"form\")")
	}

	if len(examNumberItems) != len(formElements) {
		return errors.Errorf(
			"solveQuiz: len(examNumberItems):%d != len(formElements):%d",
			len(examNumberItems), len(formElements),
		)
	}

	for idx := range examNumberItems {
		item := examNumberItems[idx]
		form := formElements[idx]

		if err := item.Click(); err != nil {
			return errors.Wrap(err, "item.Click()")
		}

		answers, err := parseAnswers(form)
		if err != nil {
			return err
		}

		if len(answers) == 0 {
			// 주관식..
			continue
		}

		// do twice
		for i := 0; i < 2; i++ {
			pickedAnswer := answers[rand.Intn(len(answers))]
			if err := pickedAnswer.Click(); err != nil {
				return errors.Wrap(err, "pickedAnswer.Click()")
			}

			_ = wd.Wait(func(wd selenium.WebDriver) (bool, error) {
				_, err := form.FindElement(selenium.ByClassName, "confirmAnswer")
				return err == nil, nil
			})

			submitButton, err := form.FindElement(selenium.ByClassName, "confirmAnswer")
			if err != nil {
				return errors.Wrap(err, "form.FindElement(selenium.ByClassName, \"confirmAnswer\")")
			}

			shouldSubmit, err := submitButton.IsDisplayed()
			if err != nil {
				return errors.Wrap(err, "submitButton.IsDisplayed()")
			}

			if shouldSubmit {
				if err := submitButton.Click(); err != nil {
					return errors.Wrap(err, "submitButton.Click()")
				}
			}

			_ = wd.AcceptAlert()
		}
	}

	return nil
}

func parseAnswers(formElement selenium.WebElement) ([]selenium.WebElement, error) {
	answerElement, err := formElement.FindElement(selenium.ByClassName, "exam-answer")
	if err != nil {
		return nil, errors.Wrap(err, "formElement.FindElement(selenium.ByClassName, \"exam-answer\")")
	}

	return answerElement.FindElements(selenium.ByClassName, "lists")
}

func play(wd selenium.WebDriver) error {
	iframeElement, err := driver.WaitAndFindElement(wd, selenium.ByTagName, "iframe")
	if err != nil {
		return errors.Wrap(err, "wd.FindElement(selenium.ByID, \"ifrmVODPlayer_0\")")
	}

	if err := wd.SwitchFrame(iframeElement); err != nil {
		return errors.Wrap(err, "wd.SwitchFrame(playerElement)")
	}

	if err := startPlayer(wd); err != nil {
		return err
	}

	if err := setFastest(wd); err != nil {
		return err
	}

	totalDuration, err := parsePlayerDuration(wd, `//*[@id="wp-controls-outer-controlbar"]/div[2]/div[2]/div/div/div[3]/span`)
	if err != nil {
		return err
	}

	if err := wd.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error) {
		currentLocation, err := parsePlayerDuration(wd, `//*[@id="wp-controls-outer-controlbar"]/div[2]/div[2]/div/div/div[1]/span`)
		if err != nil {
			return false, err
		}

		if currentLocation == nil {
			return true, nil
		}

		return totalDuration.Sub(*currentLocation) <= time.Minute, nil
	}, 24*time.Hour, time.Minute); err != nil {
		return err
	}

	return wd.SwitchFrame(nil)
}

func closeLectureWindow(wd selenium.WebDriver, mainWindowHandle string) error {
	if err := wd.Close(); err != nil {
		return errors.Wrap(err, "wd.Close()")
	}

	return wd.SwitchWindow(mainWindowHandle)
}

func parsePlayerDuration(wd selenium.WebDriver, xpath string) (*time.Time, error) {
	de, err := wd.FindElement(selenium.ByXPATH, xpath)
	if err != nil {
		return nil, errors.Wrap(err, "wd.FindElement(xpath)")
	}

	text, err := de.Text()
	if err != nil {
		return nil, errors.Wrap(err, "de.Text()")
	}

	// NOTE: When video has been completed, text is empty.
	if text == "" {
		return nil, nil
	}

	return parseDuration(text)
}

func parseDuration(duration string) (*time.Time, error) {
	withHour, err := time.Parse("15:04:05", duration)
	if err == nil {
		return &withHour, nil
	}

	withoutHour, err := time.Parse("04:05", duration)
	if err == nil {
		return &withoutHour, nil
	}

	return nil, errors.Errorf("parseDuration: invalid duration: %s", duration)
}

func setFastest(wd selenium.WebDriver) error {
	playerElement, err := wd.FindElement(selenium.ByID, "player0")
	if err != nil {
		return errors.Wrap(err, "wd.FindElement(selenium.ByID, \"player0\")")
	}
	if err := playerElement.MoveTo(0, 0); err != nil {
		return errors.Wrap(err, "playerElement.MoveTo(0,0)")
	}

	speedTitleElement, err := wd.FindElement(selenium.ByID, "currentSpeedTitle")
	if err != nil {
		return errors.Wrap(err, "wd.FindElement(selenium.ByID, \"currentSpeedTitle\")")
	}
	if err := speedTitleElement.Click(); err != nil {
		return errors.Wrap(err, "speedTitleElement.Click()")
	}

	fastestSpeedElement, err := wd.FindElement(selenium.ByID, "opSpeed_20")
	if err != nil {
		return errors.Wrap(err, "wd.FindElement(selenium.ByID, \"opSpeed_20\")")
	}

	if err := wd.Wait(func(wd selenium.WebDriver) (bool, error) {
		return fastestSpeedElement.IsDisplayed()
	}); err != nil {
		return err
	}

	return fastestSpeedElement.Click()
}

func startPlayer(wd selenium.WebDriver) error {
	playButton, err := driver.WaitAndFindElement(wd, selenium.ByXPATH, `//*[@id="player0"]/div[6]/div[1]/div`)
	if err != nil {
		return errors.Wrap(err, "wd.FindElement(selenium.ByXPATH, `//*[@id=\"player0\"]/div[6]/div[1]/div`)")
	}
	if err := playButton.Click(); err != nil {
		return errors.Wrap(err, "playButton.Click()")
	}

	return wd.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error) {
		playerElement, err := wd.FindElement(selenium.ByID, "player0")
		if err != nil {
			return false, errors.Wrap(err, "wd.FindElement(selenium.ByID, \"player0\")")
		}
		if err := clickContinue(wd); err != nil {
			return false, errors.Wrap(err, "clickContinue()")
		}

		return getStatus(playerElement) == PlayerStatusPlaying, nil
	}, time.Minute, 2*time.Second)
}

type PlayerStatus int

const (
	PlayerStatusUnknown PlayerStatus = iota
	PlayerStatusIdle
	PlayerStatusPending
	PlayerStatusPlaying
)

func getStatus(playerElement selenium.WebElement) PlayerStatus {
	playerElementClass, err := playerElement.GetAttribute("class")
	if err != nil {
		return 0
	}

	if strings.Contains(playerElementClass, "jw-state-idle") {
		return PlayerStatusIdle
	}

	if strings.Contains(playerElementClass, "jw-state-paused") {
		return PlayerStatusPending
	}

	if strings.Contains(playerElementClass, "jw-state-playing") {
		return PlayerStatusPlaying
	}

	return PlayerStatusUnknown
}

func clickContinue(wd selenium.WebDriver) error {
	e, err := wd.FindElement(selenium.ByXPATH, `//*[@id="wp_elearning_seek"]`)
	if err == nil {
		return errors.Wrap(e.Click(), "e.Click()")
	}

	e, err = wd.FindElement(selenium.ByXPATH, `//*[@id="wp_elearning_play"]`)
	if err == nil {
		return errors.Wrap(e.Click(), "e.Click()")
	}

	return nil
}
