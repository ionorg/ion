#!/usr/bin/env python

# DEPENDENCIES
####### Selenium - https://pypi.org/project/selenium/ - `pip install selenium`
####### Chromedriver - https://chromedriver.chromium.org/downloads

import sys
from time import sleep

from selenium.webdriver import Chrome, ChromeOptions
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as ec
from threading import Thread


CLIENTS = 5  # How many connections to start
SLEEP = 1  # Time between connections
TIME = 90  # Time for connection to stay active


LOGIN = '#login_displayName'
LOGIN_BUTTON = 'button.login-join-button'
EXPECTED = ec.visibility_of_element_located((By.CSS_SELECTOR, LOGIN))
URL = 'https://conference.pion.ly:8080/?room=MyRoomName' # Change to your domain and test room	


opts = ChromeOptions()
opts.add_argument('headless')  # Comment this line out to show browsers
opts.add_argument('use-fake-ui-for-media-stream')
opts.add_argument('use-fake-device-for-media-stream')


def join_video(url, num):
    driver = Chrome(options=opts)
    driver.get(url)
    login = WebDriverWait(driver, 5, 1).until(EXPECTED)
    login.send_keys('selenium' + str(num))
    btn = driver.find_element(By.CSS_SELECTOR, LOGIN_BUTTON)
    btn.click()
    sleep(TIME)
    driver.quit()


def main(url=None):
    url = url or URL
    threads = []
    for i in range(CLIENTS):
        threads.append(Thread(target=join_video, args=(url, i)))

    for thread in threads:
        thread.start()
        if SLEEP:
            sleep(SLEEP)

    for thread in threads:
        thread.join()


if __name__ == "__main__":
    # Optionally pass in meeting URL from command line
    url = sys.argv[-1] if len(sys.argv) == 2 else None
    main(url)
