package univ

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/tebeka/selenium"

	"github.com/Kcrong/autostudy/pkg/driver"
)

type Subject struct {
	Title        string
	Progress     float32
	Lectures     []*Lecture
	ExpandButton selenium.WebElement
}

func (s Subject) IsCompleted() bool {
	for _, lecture := range s.Lectures {
		if !lecture.IsDone() {
			return false
		}
	}

	return true
}

type GetSubjectsOption struct {
	WithLectures bool
	WatchFunc    WatchFuncType
}

func GetSubjects(url string, wd selenium.WebDriver, withLectures bool, watchFunc WatchFuncType) ([]*Subject, error) {
	element, err := getSubjectElement(url, wd)
	if err != nil {
		return nil, err
	}

	return parseSubjects(element, withLectures, watchFunc)
}

func getSubjectElement(url string, wd selenium.WebDriver) (selenium.WebElement, error) {
	if err := driver.AssertUrl(url, wd); err != nil {
		return nil, err
	}

	return wd.ActiveElement()
}

func parseSubjects(element selenium.WebElement, parseLectures bool, watchFunc WatchFuncType) ([]*Subject, error) {
	progressElement, err := element.FindElement(selenium.ByClassName, "lecture-progress")
	if err != nil {
		return nil, errors.Wrap(err, "element.FindElement(lecture-progress)")
	}

	subjectElements, err := progressElement.FindElements(selenium.ByClassName, "lecture-progress-item")
	if err != nil {
		return nil, errors.Wrap(err, "progress.FindElements(lecture-progress-item)")
	}

	subjects := make([]*Subject, len(subjectElements))
	for i, subjectElement := range subjectElements {
		sj, err := parseSubjectElement(subjectElement, parseLectures, watchFunc)
		if err != nil {
			return nil, err
		}
		subjects[i] = sj
	}

	return subjects, nil
}

func parseSubjectElement(subjectElement selenium.WebElement, parseLecture bool, watchFunc WatchFuncType) (*Subject, error) {
	infoElement, err := subjectElement.FindElement(selenium.ByClassName, "lecture-info")
	if err != nil {
		return nil, errors.Wrap(err, "subjectElement.FindElement(lecture-info)")
	}

	progress, err := extractProgress(infoElement)
	if err != nil {
		return nil, err
	}

	buttonElement, err := infoElement.FindElement(selenium.ByClassName, "btn-toggle")
	if err != nil {
		return nil, errors.Wrap(err, "infoElement.FindElement(btn-toggle)")
	}

	titleText, err := buttonElement.Text()
	if err != nil {
		return nil, errors.Wrap(err, "buttonElement.Text()")
	}

	var lectures []*Lecture
	if parseLecture {
		// Click the button to expand the lecture list.
		if err := buttonElement.Click(); err != nil {
			return nil, errors.Wrap(err, "buttonElement.Click()")
		}

		// Extract lecture web elements
		lectureElements, err := extractLectureElements(subjectElement)
		if err != nil {
			return nil, err
		}

		lectures = make([]*Lecture, len(lectureElements))
		for i, element := range lectureElements {
			lectures[i], err = ParseLecture(element, watchFunc)
			if err != nil {
				return nil, err
			}
		}
	}

	return &Subject{
		Title:        titleText,
		Progress:     progress,
		Lectures:     lectures,
		ExpandButton: buttonElement,
	}, nil
}

func extractLectureElements(subjectElement selenium.WebElement) ([]selenium.WebElement, error) {
	body, err := subjectElement.FindElement(selenium.ByClassName, "lecture-progress-item-body")
	if err != nil {
		return nil, errors.Wrap(err, "subjectElement.FindElement(lecture-progress-item-body)")
	}

	list, err := body.FindElement(selenium.ByClassName, "lecture-list")
	if err != nil {
		return nil, errors.Wrap(err, "body.FindElement(lecture-list)")
	}

	lectureElements, err := list.FindElements(selenium.ByClassName, "clearfix")
	if err != nil {
		return nil, errors.Wrap(err, "list.FindElements(clearfix)")
	}

	return lectureElements, nil
}

func extractProgress(infoElement selenium.WebElement) (float32, error) {
	per, err := infoElement.FindElement(selenium.ByClassName, "lecture-per")
	if err != nil {
		return 0, errors.Wrap(err, "infoElement.FindElement(lecture-per)")
	}

	value, err := per.FindElement(selenium.ByClassName, "value")
	if err != nil {
		return 0, errors.Wrap(err, "per.FindElement(value)")
	}

	text, err := value.Text()
	if err != nil {
		return 0, errors.Wrap(err, "value.Text()")
	}

	p, err := strconv.ParseFloat(text, 32)
	if err != nil {
		return 0, errors.Wrap(err, "strconv.ParseFloat")
	}

	return float32(p), nil
}
