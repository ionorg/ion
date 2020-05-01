#!/usr/bin/env python

# DEPENDENCIES
####### Selenium - https://pypi.org/project/selenium/ - `pip install selenium`
####### Chromedriver - https://chromedriver.chromium.org/downloads

import sys
from time import sleep

from selenium.webdriver import Chrome, ChromeOptions
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as ec
from threading import Thread


SLEEP = 1  # Time between connections
TIME = 90  # Time for connection to stay active


LOGIN = '#login_displayName'
LOGIN_BUTTON = 'button.login-join-button'
EXPECTED = ec.visibility_of_element_located((By.CSS_SELECTOR, LOGIN))
# Change to your domain and test room
URL = 'https://conference.pion.ly:8080/?room=MyRoomName'


opts = ChromeOptions()
opts.add_argument('headless')  # Comment this line out to show browsers
opts.add_argument('use-fake-ui-for-media-stream')
opts.add_argument('use-fake-device-for-media-stream')


def join_room(url, num):
    driver = Chrome(options=opts)
    driver.get(url)
    login = WebDriverWait(driver, 5, 1).until(EXPECTED)
    login.send_keys('selenium' + str(num))
    btn = driver.find_element(By.CSS_SELECTOR, LOGIN_BUTTON)
    btn.click()
    sleep(TIME)
    leave_room(driver)


def leave_room(driver):
    leave_button = driver.find_element_by_css_selector('.app-header-tool .ant-btn-circle')
    leave_button.click()

    ok_button = '.ant-modal-confirm-btns .ant-btn-primary'
    confirm = WebDriverWait(driver, 5, 1).until(ec.element_to_be_clickable((By.CSS_SELECTOR, ok_button)))
    confirm.send_keys(Keys.ENTER)

    driver.quit()


def main(runType='single', clients=2, rooms=1, url=None):
    url = url or URL

    if runType == 'multiple':
        multiple(url, clients, rooms)
    else:
        single(url, clients)


def single(url, clients):
    threads = []

    for i in range(clients):
        threads.append(Thread(target=join_room, args=(url, i)))

    for thread in threads:
        thread.start()
        if SLEEP:
            sleep(SLEEP)

    for thread in threads:
        thread.join()


def multiple(url, clients, rooms):
    threads = []

    for x in range(rooms):
        room = url + str(x)
        for i in range(clients):
            threads.append(Thread(target=join_room, args=(room, i)))

    for thread in threads:
        thread.start()
        if SLEEP:
            sleep(SLEEP)

    for thread in threads:
        thread.join()


if __name__ == "__main__":
    runType = sys.argv[1] if len(sys.argv) >= 2 else None
    clients = int(sys.argv[2]) if len(sys.argv) >= 3 else 2
    rooms = int(sys.argv[3]) if len(sys.argv) >= 4 else 1
    url = sys.argv[4] if len(sys.argv) >= 5 else None

    main(runType, clients, rooms, url)
