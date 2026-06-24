"""功能定制验证模板 —— 每次开发完一个功能，复制本文件改名后修改。

示例：验证「网盘页列表能正常加载」。
运行（账号命令行注入）：
  cd scripts/verify && DEVOPS_TEST_USER=User DEVOPS_TEST_PASS='admin@123' \
      python3 tasks/_example.py
"""
from __future__ import annotations

import os
import sys

# tasks/ 在子目录，需把 scripts/verify 加入 sys.path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from playwright.sync_api import sync_playwright

import config
import auth
import probe
import report

# ===== 本次验证配置（每次按功能修改这里）=====
FEATURE_NAME = "网盘页列表加载"
TARGET_URL = f"{config.FRONTEND_URL}/disk"   # 本次功能的目标页面


def assert_feature(page) -> list:
    """本次功能的关键断言。返回 CheckResult 列表。"""
    checks = []
    try:
        page.wait_for_selector("body", timeout=5000)
        # 按功能改: 用真实选择器断言，例如：
        # items = page.locator(".disk-item").count()
        # checks.append(report.CheckResult("列表渲染", items >= 1, f"实际 {items} 项"))
        checks.append(report.CheckResult("页面可访问", True, TARGET_URL))
    except Exception as e:
        checks.append(report.CheckResult("页面可访问", False, str(e)))
    return checks


def run() -> int:
    config.check_services_reachable()
    account = config.Account.from_env()
    shot = report.screenshot_path(FEATURE_NAME)
    os.makedirs(os.path.dirname(shot), exist_ok=True)

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        ctx = browser.new_context()
        page = ctx.new_page()
        session = probe.attach(page)

        login_res = auth.login(ctx, page, account)
        if not login_res.success:
            page.screenshot(path=shot)
            print(report.render(FEATURE_NAME, smoke_passed=False, smoke_detail=login_res.detail,
                                checks=[], probe_obj=session, screenshot_path=shot))
            browser.close()
            return 1

        page.goto(TARGET_URL, wait_until="networkidle", timeout=config.DEFAULT_TIMEOUT_MS)
        checks = assert_feature(page)
        page.screenshot(path=shot)
        browser.close()

    out = report.render(FEATURE_NAME, smoke_passed=not session.has_errors,
                        smoke_detail=f"{login_res.detail} | 目标页加载✓",
                        checks=checks, probe_obj=session, screenshot_path=shot)
    print(out)
    has_fail = session.has_errors or any(not c.passed for c in checks)
    return 1 if has_fail else 0


if __name__ == "__main__":
    sys.exit(run())
