"""登录模块（API + route 注入方案）。

背景：dev 模式 `pnpm dev` = `vite --mode test`，前端直连后端
(http://172.21.10.40:8888)，属跨站请求。后端用 cookie 传递 JWT token，
但跨站 + HTTP 下浏览器无法自动存储/携带该 cookie，导致真实登录页流程在
headless 自动化中失败（getUserInfo 返回 code 7 未登录 → 自动登出）。

解法（已验证可行）：
  1. 直接调登录 API 拿 token（从 Set-Cookie 解析）
  2. 注册 page.route：对 /api/ 请求用 route.fetch 注入 cookie token 后 fulfill
     （route.continue_ 的 Cookie 头会被浏览器忽略，必须用 route.fetch）
  3. add_init_script 注入 localStorage SOY_isAuthenticated=true
     （前端 isLogin 依据，见 store/modules/auth/shared.ts getToken）
  4. goto 前端，前端以登录态加载，API 带 token 正常工作
"""
from __future__ import annotations

import os
import re
from dataclasses import dataclass

import config

# 后端直连地址（.env.test VITE_SERVICE_BASE_URL），可用环境变量覆盖
BACKEND_URL = os.environ.get("DEVOPS_BACKEND_URL", "http://172.21.10.40:8888")
LOGIN_API = f"{BACKEND_URL}/api/v1/auth/loginWithInfo"


@dataclass(frozen=True)
class LoginResult:
    success: bool
    detail: str
    final_url: str = ""


def _obtain_token(context, account: "config.Account") -> str:
    """调登录 API，从 Set-Cookie 解析 token。"""
    resp = context.request.post(LOGIN_API, data={
        "userName": account.username,
        "password": account.password,
        "captchaToken": "",
    })
    set_cookie = resp.headers.get("set-cookie", "")
    m = re.search(r"token=([^;]+)", set_cookie)
    if not m:
        raise RuntimeError(f"登录 API 未返回 token (HTTP {resp.status})")
    return m.group(1)


def _install_auth_route(page, token: str) -> None:
    """注册 route：给后端 /api/ 请求注入 cookie token。

    SSE 长连接 route.fetch 不支持，验证场景不需要，返回空响应让其安静关闭。
    """
    def handler(route, request):
        accept = request.headers.get("accept", "")
        if "text/event-stream" in accept or "/sse/" in request.url:
            route.fulfill(status=200, headers={"content-type": "text/event-stream"}, body="")
            return
        headers = {k: v for k, v in request.headers.items()}
        headers["cookie"] = f"token={token}"
        try:
            response = route.fetch(headers=headers)
            route.fulfill(response=response)
        except Exception:
            route.continue_()

    page.route("**/api/**", handler)


def login(context, page, account: "config.Account") -> LoginResult:
    """API 登录 + 注入登录态。调用方需先 new_context + new_page。"""
    try:
        token = _obtain_token(context, account)
    except Exception as e:
        return LoginResult(success=False, detail=f"登录失败: {e}")

    _install_auth_route(page, token)
    # 前端 isLogin 依据（VITE_STORAGE_PREFIX=SOY_）
    page.add_init_script("localStorage.setItem('SOY_isAuthenticated','true')")

    page.goto(config.FRONTEND_URL, wait_until="domcontentloaded", timeout=config.DEFAULT_TIMEOUT_MS)
    try:
        page.wait_for_url(lambda url: config.LOGIN_PATH not in url, timeout=config.DEFAULT_TIMEOUT_MS)
    except Exception:
        return LoginResult(success=False, detail="登录后仍停留在登录页", final_url=page.url)

    page.wait_for_load_state("networkidle", timeout=config.DEFAULT_TIMEOUT_MS)
    return LoginResult(success=True, detail="登录成功", final_url=page.url)
