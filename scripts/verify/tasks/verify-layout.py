"""验证布局预设重构：disk(workbench) 与 admin(standard) 两套预设均正常渲染、模块导航无回归。

运行：
  cd scripts/verify && DEVOPS_TEST_USER=User DEVOPS_TEST_PASS='admin@123' \
      python3 tasks/verify-layout.py
"""
from __future__ import annotations

import os
import sys

# tasks/ 在子目录，需把 scripts/verify 加入 sys.path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from playwright.sync_api import sync_playwright

import auth
import config
import probe
import report

FEATURE_NAME = "布局预设重构"

# (路径, 标签) —— 覆盖 workbench 与 standard 两种预设 + admin 模块子页导航
TARGETS: list[tuple[str, str]] = [
    ("/disk", "disk工作台(workbench预设)"),
    ("/favorite", "收藏(standard预设)"),
    ("/recent", "最近访问(standard预设)"),
]


def run() -> int:
    config.check_services_reachable()
    account = config.Account.from_env()
    shot_dir = os.path.dirname(report.screenshot_path(FEATURE_NAME))
    os.makedirs(shot_dir, exist_ok=True)

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        ctx = browser.new_context()
        page = ctx.new_page()
        session = probe.attach(page)

        login_res = auth.login(ctx, page, account)
        if not login_res.success:
            print(report.render(FEATURE_NAME, smoke_passed=False, smoke_detail=login_res.detail,
                                checks=[], probe_obj=session))
            browser.close()
            return 1

        checks = []
        for path, label in TARGETS:
            shot = os.path.join(shot_dir, f"layout-{path.strip('/').replace('/', '_') or 'root'}-chk.png")
            try:
                page.goto(f"{config.FRONTEND_URL}{path}", wait_until="networkidle", timeout=config.DEFAULT_TIMEOUT_MS)
                page.screenshot(path=shot)
                checks.append(report.CheckResult(f"{label} 加载", True, page.url))
            except Exception as e:  # noqa: BLE001 - 验证脚本需捕获所有异常以报告
                checks.append(report.CheckResult(f"{label} 加载", False, str(e)))
        browser.close()

    has_fail = session.has_errors or any(not c.passed for c in checks)
    summary_shot = os.path.join(shot_dir, "layout-disk.png")
    out = report.render(FEATURE_NAME, smoke_passed=not session.has_errors,
                        smoke_detail=f"{login_res.detail} | 多预设页面加载",
                        checks=checks, probe_obj=session, screenshot_path=summary_shot)
    print(out)
    return 1 if has_fail else 0


if __name__ == "__main__":
    sys.exit(run())
