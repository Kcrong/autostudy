import io
import logging
import os
import time
from contextlib import suppress
from dataclasses import dataclass
from datetime import datetime
from typing import List, Optional

import telegram

from selenium import webdriver
from selenium.common import (
    NoSuchElementException,
    StaleElementReferenceException,
    TimeoutException,
)
from selenium.webdriver import ActionChains
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.by import By
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support import expected_conditions
from selenium.webdriver.support.wait import WebDriverWait

username = "hyunwoo1010"
password = os.environ.get("PW")

chat_id = 5538533245

# driver = webdriver.Chrome("./chromedriver")
driver = webdriver.Chrome(service=Service("./chromedriver"))
actions = ActionChains(driver)
bot = telegram.Bot(token=os.environ.get("TELEGRAM_API_TOKEN"))


@dataclass
class Lecture:
    title: str
    hasDone: bool
    button: Optional[WebElement]  # 강의 듣기 버튼


@dataclass
class Subject:  # 과목
    title: str
    progress: float
    lectures: List[Lecture]


def main():
    driver.get("https://www.naver.com")

    assert len(driver.window_handles) == 1
    main_tab = driver.window_handles[0]

    # login
    driver.get("https://ep.knou.ac.kr/")

    if driver.current_url == "https://ep.knou.ac.kr/login.do?epTicket=LOG":
        driver.find_element(By.ID, "username").send_keys(username)
        pw_element = driver.find_element(By.ID, "password")
        pw_element.send_keys(password)
        pw_element.send_keys(Keys.ENTER)

    assert driver.current_url == "https://ep.knou.ac.kr/main.do"

    # campus site
    driver.get(
        "https://ucampus.knou.ac.kr/ekp/user/study/retrieveUMYStudy.sdo"
    )

    lecture_progress = driver.find_element(By.CLASS_NAME, "lecture-progress")
    bot.send_message(chat_id=chat_id, text="현재 수강 상황입니다.")
    bot.send_photo(
        chat_id=chat_id, photo=io.BytesIO(lecture_progress.screenshot_as_png)
    )

    subject_elements = lecture_progress.find_elements(
        By.CLASS_NAME, "lecture-progress-item"
    )

    for element in subject_elements:
        subject = parse_subject(element)

        for le in subject.lectures:
            if le.hasDone:
                continue

            if le.button is None:
                break

            le.button.click()
            driver.implicitly_wait(1)

            # Tab switching
            added = get_added_window_handle(driver.window_handles, main_tab)
            driver.switch_to.window(added)

            driver.implicitly_wait(3)

            # player action

            play_player_with_fastest()

            wait_until_lecture_completion()

            driver.switch_to.default_content()
            try:
                driver.find_element(
                    By.XPATH, """//*[@id="top"]/div[2]/button"""
                ).click()
                WebDriverWait(driver, 30).until(
                    expected_conditions.alert_is_present()
                )
                driver.switch_to.alert.accept()
            except TimeoutException:
                logging.warning("No alert when finishing the video")
                driver.close()

            driver.switch_to.window(main_tab)
            bot.send_message(chat_id=chat_id, text=f"{le.title} 을 수강했습니다.")

        # bot.send_message(
        #     chat_id=chat_id, text=f"{subject.title} 과목을 모두 수강했습니다."
        # )


def wait_until_lecture_completion():
    total_duration = driver.find_element(
        By.XPATH,
        """//*[@id="wp-controls-outer-controlbar"]\
/div[2]/div[2]/div/div/div[3]/span""",
    ).get_attribute("innerHTML")

    while True:
        current_location = driver.find_element(
            By.XPATH,
            """//*[@id="wp-controls-outer-controlbar"]/div[2]/div[\
2]/div/div/div[1]/span""",
        ).get_attribute("innerHTML")

        if total_duration == current_location:
            break

        else:
            delta = datetime.strptime(
                total_duration, "%M:%S"
            ) - datetime.strptime(current_location, "%M:%S")
            period = delta.total_seconds() / 3

            time.sleep(period)


def play_player_with_fastest():
    driver.implicitly_wait(3)
    time.sleep(3)

    # play
    while True:
        try:
            driver.find_element(By.ID, "ifrmVODPlayer_0").click()
        except (StaleElementReferenceException, NoSuchElementException):
            driver.implicitly_wait(1)
        else:
            driver.switch_to.frame(
                driver.find_element(By.ID, "ifrmVODPlayer_0")
            )
            watch_continue()
            if is_playing():
                break
            driver.switch_to.default_content()

    while True:
        try:
            actions.move_to_element(
                driver.find_element(By.ID, "player0")
            ).perform()
            driver.find_element(
                By.XPATH, """//*[@id="currentSpeedTitle"]"""
            ).click()
            driver.find_element(By.ID, "opSpeed_20").click()
        except:
            pass
        else:
            break


def watch_continue():
    with suppress(Exception):
        driver.find_element(
            By.XPATH, """//*[@id="wp_elearning_seek"]"""
        ).click()

    with suppress(Exception):
        driver.find_element(
            By.XPATH, """//*[@id="wp_elearning_play"]"""
        ).click()


def is_playing():
    try:
        if (
            driver.find_element(
                By.XPATH, """//*[@id="comment_player0"]"""
            ).value_of_css_property("display")
            == "none"
        ):
            return True
    except:
        return False


def get_added_window_handle(handles, main_tab):
    for handle in handles:
        if handle != main_tab:
            return handle


def parse_subject(element):
    info = element.find_element(By.CLASS_NAME, "lecture-info")

    title = info.find_element(By.CLASS_NAME, "btn-toggle").text

    progress_text = (
        info.find_element(By.CLASS_NAME, "lecture-per")
        .find_element(By.CLASS_NAME, "value")
        .text
    )
    try:
        progress = float(progress_text)
    except ValueError:
        progress = float(0)
        logging.warning(
            f"invalid progress. progress_text: {progress_text} title: {title}"
        )

    # 강의 목록 파싱 전 확장 버튼 클릭 필요
    info.find_element(By.CLASS_NAME, "btn-toggle").click()
    driver.implicitly_wait(1)

    lecture_elements = (
        element.find_element(By.CLASS_NAME, "lecture-progress-item-body")
        .find_element(By.CLASS_NAME, "lecture-list")
        .find_elements(By.CLASS_NAME, "clearfix")
    )

    return Subject(
        title=title,
        progress=progress,
        lectures=[parse_lecture(le) for le in lecture_elements],
    )


def parse_lecture(element):
    title_element = element.find_element(By.CLASS_NAME, "lecture-title")
    title = title_element.text
    try:
        has_done_classes = (
            element.find_element(By.CLASS_NAME, "lecture-list-in")
            .find_element(By.TAG_NAME, "a")
            .get_attribute("class")
            .split()
        )
    except NoSuchElementException as e:
        if element.find_element(By.CLASS_NAME, "con-waiting"):
            return Lecture(
                title=title,
                hasDone=False,
                button=None,
            )
        else:
            raise e

    if "ch" not in has_done_classes:
        logging.warning(
            f"invalid done text. has_done_classes: {has_done_classes} "
            f"lecture_title: {title}"
        )

    return Lecture(
        title=title,
        hasDone="on" in has_done_classes,
        button=title_element,
    )


if __name__ == "__main__":
    while True:
        try:
            main()
        except Exception as e:
            logging.error(e)
            logging.info("Unexpected error has been raised. Retrying...")
        else:
            bot.send_message(chat_id=chat_id, text="현재 수강 가능한 과목이 없습니다.")
