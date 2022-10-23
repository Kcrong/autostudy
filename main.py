import io
import os
import time
from contextlib import suppress
from dataclasses import dataclass
from datetime import datetime, timedelta
from typing import List, Optional

import humanize
import pytz
import telegram
from selenium import webdriver
from selenium.common import (
    NoSuchElementException,
    StaleElementReferenceException,
    TimeoutException,
)
from selenium.webdriver import ActionChains
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support import expected_conditions
from selenium.webdriver.support.wait import WebDriverWait

DAILY_SCHEDULED_HOUR = 9  # Runs on every 9 AM

username = os.environ.get("KNOU_ID")
password = os.environ.get("KNOU_PW")

telegram_chat_id = os.environ.get("TELEGRAM_CHAT_ID")
telegram_token = os.environ.get("TELEGRAM_API_TOKEN")
driver_command_url = os.environ.get("DRIVER_COMMAND_URL")


def init_driver():
    options = webdriver.ChromeOptions()
    options.add_argument("headless")
    options.add_argument("window-size=1920x1080")
    options.add_argument("--no-sandbox")
    options.add_argument("--disable-dev-shm-usage")
    options.add_argument("disable-gpu")
    options.add_argument(
        f"user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) "
        f"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.87 "
        f"Safari/537.36"
    )

    return webdriver.Remote(
        command_executor=driver_command_url,
        options=options,
    )


driver = init_driver()
actions = ActionChains(driver)
bot = telegram.Bot(token=telegram_token)


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


def report_via_telegram(
    text: str = "",
    elem: Optional[WebElement] = None,
    should_capture_driver: bool = False,
):
    bot.send_message(chat_id=telegram_chat_id, text=text)
    if elem is not None:
        bot.send_photo(
            chat_id=telegram_chat_id,
            photo=io.BytesIO(elem.screenshot_as_png),
        )
    if should_capture_driver:
        bot.send_photo(
            chat_id=telegram_chat_id,
            photo=io.BytesIO(driver.get_screenshot_as_png()),
        )


def main():
    # login
    driver.get("https://ep.knou.ac.kr/")

    assert len(driver.window_handles) == 1
    main_tab = driver.window_handles[0]

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
    report_via_telegram(text="현재 수강 상황입니다.", elem=lecture_progress)

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

            report_via_telegram(
                text=f"{subject.title} 의 {le.title} 시작합니다. \
현재 수강률은 {subject.progress} 입니다."
            )

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
                report_via_telegram(
                    "학습 종료 시 확인 창이 뜨지 않았습니다.", should_capture_driver=True
                )
                driver.close()

            driver.switch_to.window(main_tab)
            report_via_telegram(text=f"{le.title} 을 수강했습니다.")


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
        report_via_telegram(
            text=f"Progress가 잘못되었습니다. text: {progress_text} title: "
            f"{title}",
            elem=info,
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
        report_via_telegram(
            text=f"invalid done text. has_done_classes: {has_done_classes} "
            f"lecture_title: {title}",
            elem=element,
        )

    return Lecture(
        title=title,
        hasDone="on" in has_done_classes,
        button=title_element,
    )


def td_format(td_object):
    seconds = int(td_object.total_seconds())
    periods = [
        ("년", 60 * 60 * 24 * 365),
        ("월", 60 * 60 * 24 * 30),
        ("일", 60 * 60 * 24),
        ("시간", 60 * 60),
        ("분", 60),
        ("초", 1),
    ]

    strings = []
    for period_name, period_seconds in periods:
        if seconds > period_seconds:
            period_value, seconds = divmod(seconds, period_seconds)
            has_s = "s" if period_value > 1 else ""
            strings.append("%s %s%s" % (period_value, period_name, has_s))

    return ", ".join(strings)


if __name__ == "__main__":
    kst = pytz.timezone("Asia/Seoul")
    humanize.i18n.activate("ko_KR")

    # Wait until DAILY_SCHEDULED_HOUR
    t = datetime.now(kst)
    future = datetime(
        t.year, t.month, t.day, DAILY_SCHEDULED_HOUR, 0, tzinfo=kst
    )
    if t.hour >= DAILY_SCHEDULED_HOUR:
        future += timedelta(days=1)
    delta = future - t
    report_via_telegram(
        text=f"재시작되었습니다. " f"{humanize.naturaltime(delta)} 시작합니다."
    )
    time.sleep(delta.total_seconds())

    try:
        main()
    except Exception as e:
        report_via_telegram(
            text=f"에러가 발생했습니다: {e}", should_capture_driver=True
        )

    driver.quit()
