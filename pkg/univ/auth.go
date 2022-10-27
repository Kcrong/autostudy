package univ

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/tebeka/selenium"
)

func Login(d selenium.WebDriver, url, id, pw string, afterUrl *string) error {
	if err := d.Get(url); err != nil {
		return errors.Wrap(err, fmt.Sprintf("d.Get(%s)", url))
	}

	idElement, err := d.FindElement(selenium.ByID, "username")
	if err != nil {
		return errors.Wrap(err, "d.FindElement(selenium.ByID, username)")
	}
	if err := idElement.SendKeys(id); err != nil {
		return errors.Wrap(err, "idElement.SendKeys(id)")
	}

	pwElement, err := d.FindElement(selenium.ByID, "password")
	if err != nil {
		return errors.Wrap(err, "d.FindElement(selenium.ByID, password)")
	}
	if err := pwElement.SendKeys(pw); err != nil {
		return errors.Wrap(err, "pwElement.SendKeys(id)")
	}
	if err := pwElement.SendKeys(selenium.EnterKey); err != nil {
		return errors.Wrap(err, "pwElement.SendKeys(selenium.EnterKey)")
	}

	if afterUrl != nil {
		cu, err := d.CurrentURL()
		if err != nil {
			return errors.Wrap(err, "d.CurrentURL()")
		}

		if cu != *afterUrl {
			return errors.Errorf("Login failed. Expected url: %s, Actual url: %s", *afterUrl, cu)
		}
	}

	return nil
}
