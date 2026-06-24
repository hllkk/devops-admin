"""通用冒烟：登录 → 主页加载 → 全局无报错。每次验证的基线，也用于工具自测。

运行（账号命令行注入）：
  cd scripts/verify && DEVOPS_TEST_USER=User DEVOPS_TEST_PASS='admin@123' python3 smoke.py
退出码：0 = 通过，1 = 有错误/失败。
"""
from __future__ import annotations

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from playwright.sync_api import sync_playwright

import config
import auth
import probe
import report


def run() -> int:
    config.check_services_reachable()
    account = config.Account.from_env()
    shot = report.screenshot_path("smoke-home")
    os.makedirs(os.path.dirname(shot), exist_ok=True)

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        ctx = browser.new_context()
        page = ctx.new_page()
        session = probe.attach(page)

        login_res = auth.login(ctx, page, account)
        if not login_res.success:
            page.screenshot(path=shot)
            print(report.render(
                "通用冒烟", smoke_passed=False, smoke_detail=login_res.detail,
                checks=[], probe_obj=session, screenshot_path=shot))
            browser.close()
            return 1

        page.wait_for_load_state("networkidle", timeout=config.DEFAULT_TIMEOUT_MS)
        page.screenshot(path=shot)
        browser.close()

    smoke_detail = f"{login_res.detail} | networkidle✓ | 主页={login_res.final_url}"
    out = report.render(
        "通用冒烟", smoke_passed=not session.has_errors, smoke_detail=smoke_detail,
        checks=[], probe_obj=session, screenshot_path=shot)
    print(out)
    return 0 if not session.has_errors else 1


if __name__ == "__main__":
    sys.exit(run())
