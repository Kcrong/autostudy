package univ

import (
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tebeka/selenium"

	"github.com/Kcrong/autostudy/pkg/driver"
)

type Lecture struct {
	Title string

	IsReadied bool

	HasPlayed        bool
	HasExam          bool
	HasExamCompleted bool

	PlaybackLocation time.Duration
	PlaybackDuration time.Duration

	ButtonElement selenium.WebElement
}

func (l Lecture) ShouldBePlayed() bool {
	if !l.IsReadied {
		return false
	}

	// Not played yet or has exam and not completed
	return !l.HasPlayed || (l.HasExam && !l.HasExamCompleted)
}

func (l Lecture) ShouldBeExamined() bool {
	return l.HasExam && !l.HasExamCompleted
}

func (l Lecture) IsDone() bool {
	if !l.IsReadied {
		return false
	}

	return !l.ShouldBePlayed() && !l.ShouldBeExamined()
}

func ParseLecture(lectureElement selenium.WebElement, watchFunc WatchFuncType) (*Lecture, error) {
	titleElement, err := lectureElement.FindElement(selenium.ByClassName, "lecture-title")
	if err != nil {
		return nil, errors.Wrap(err, "lectureElement.FindElement(lecture-title)")
	}

	title, err := titleElement.Text()
	if err != nil {
		return nil, errors.Wrap(err, "titleElement.Text")
	}

	if !isLectureReady(lectureElement) {
		return &Lecture{
			Title:     title,
			IsReadied: false,
		}, nil
	}

	lectureStatusElement, err := lectureElement.FindElement(selenium.ByClassName, "lecture-list-in")
	if err != nil {
		return nil, errors.Wrap(err, "lectureElement.FindElement(lecture-list-in)")
	}

	hasPlayed, hasExam, hasExamCompleted, playbackLocationMin, playbackDurationMin, err := extractLectureStatus(lectureStatusElement)
	if err != nil {
		return nil, err
	}

	lecture := &Lecture{
		Title:            title,
		IsReadied:        true,
		HasPlayed:        hasPlayed,
		HasExam:          hasExam,
		HasExamCompleted: hasExamCompleted,
		PlaybackLocation: playbackLocationMin,
		PlaybackDuration: playbackDurationMin,
		ButtonElement:    titleElement,
	}

	if watchFunc != nil && !lecture.IsDone() {
		if err := watchFunc(lecture); err != nil {
			return nil, err
		}
	}

	return lecture, nil
}

func isLectureReady(lectureElement selenium.WebElement) bool {
	_, err := lectureElement.FindElement(selenium.ByClassName, "con-waiting")
	return driver.IsNoSuchElementError(err)
}

func extractLectureStatus(lectureStatusElement selenium.WebElement) (bool, bool, bool, time.Duration, time.Duration, error) {
	liElements, err := lectureStatusElement.FindElements(selenium.ByTagName, "li")
	if err != nil {
		return false, false, false, 0, 0, errors.Wrap(err, "lectureStatusElement.FindElements")
	}

	if len(liElements) != 3 {
		return false, false, false, 0, 0, errors.Errorf("len(liElements) != 3: %d", len(liElements))
	}

	playbackElement, examElement, playbackMinElement := liElements[0], liElements[1], liElements[2]

	hasPlayed, err := extractHasPlayed(playbackElement)
	if err != nil {
		return false, false, false, 0, 0, err
	}

	hasExam, hasExamCompleted, err := extractExamInfo(examElement)
	if err != nil {
		return hasPlayed, false, false, 0, 0, err
	}

	locationMin, durationMin, err := extractPlaybackMin(playbackMinElement)
	if err != nil {
		return hasPlayed, hasExam, hasExamCompleted, 0, 0, err
	}

	return hasPlayed, hasExam, hasExamCompleted, locationMin, durationMin, nil
}

func extractHasPlayed(playbackElement selenium.WebElement) (bool, error) {
	a, err := playbackElement.FindElement(selenium.ByTagName, "a")
	if err != nil {
		return false, errors.Wrap(err, "playbackElement.FindElement")
	}

	return isChecked(a)
}

func extractExamInfo(examElement selenium.WebElement) (bool, bool, error) {
	a, err := examElement.FindElement(selenium.ByTagName, "a")
	if err != nil {
		if driver.IsNoSuchElementError(err) {
			return false, false, nil
		}

		return false, false, errors.Wrap(err, "examElement.FindElement")
	}

	completed, err := isChecked(a)

	return true, completed, err
}

func extractPlaybackMin(playbackSecElement selenium.WebElement) (time.Duration, time.Duration, error) {
	spans, err := playbackSecElement.FindElements(selenium.ByTagName, "span")
	if err != nil {
		return 0, 0, errors.Wrap(err, "a.FindElements")
	}

	locationElement, durationElement := spans[0], spans[1]

	locationMin, err := parseTimeElement(locationElement)
	if err != nil {
		return 0, 0, err
	}

	durationMin, err := parseTimeElement(durationElement)
	if err != nil {
		return locationMin, 0, err
	}

	return locationMin, durationMin, nil
}

func parseTimeElement(timeElement selenium.WebElement) (time.Duration, error) {
	minuteStr, err := timeElement.Text()
	if err != nil {
		return 0, errors.Wrap(err, "timeElement.Text")
	}

	minute, err := strconv.ParseInt(minuteStr, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "strconv.ParseInt")
	}

	return time.Duration(minute) * time.Minute, nil
}

func isChecked(aElement selenium.WebElement) (bool, error) {
	class, err := aElement.GetAttribute("class")
	if err != nil {
		return false, errors.Wrap(err, "a.GetAttribute")
	}

	return strings.Contains(class, "on"), nil
}
