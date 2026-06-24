"""错误探测器：在 page 上挂载 console/pageerror/response 监听，收集结构化错误。"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Optional

import config

# 控制台消息类型 → 是否记录及归类
_CONSOLE_KIND = {
    "error": "console.error",
    "warning": "console.warning",
}


def classify_console(msg_type: str) -> Optional[str]:
    """控制台消息类型 → 错误归类；非错误返回 None。"""
    return _CONSOLE_KIND.get(msg_type)


def parse_business_error(body) -> Optional[str]:
    """解析后端响应体，若业务码非成功则返回描述字符串，否则 None。"""
    if not isinstance(body, dict):
        return None
    code = body.get("code")
    if config.is_success_code(code):
        return None
    msg = body.get("msg") or body.get("message") or ""
    return f'业务码 {code}: "{msg}"'


@dataclass(frozen=True)
class VerifError:
    kind: str          # console.error | console.warning | pageerror | network | business
    message: str
    location: str = ""  # 文件:行 或 请求 URL
    detail: str = ""    # 状态码、业务码等补充


@dataclass
class ProbeSession:
    """累积一次验证过程中的所有错误。"""
    errors: list = field(default_factory=list)
    _seen: set = field(default_factory=set)

    def add(self, err: VerifError) -> None:
        key = (err.kind, err.message, err.location)
        if key in self._seen:
            return
        self._seen.add(key)
        self.errors.append(err)

    @property
    def has_errors(self) -> bool:
        return len(self.errors) > 0

    def summary_by_kind(self) -> dict:
        summary: dict = {}
        for e in self.errors:
            summary[e.kind] = summary.get(e.kind, 0) + 1
        return summary


def attach(page) -> ProbeSession:
    """在 page 上挂载三类监听器，返回累积错误的 ProbeSession。"""
    session = ProbeSession()

    def _on_console(msg):
        kind = classify_console(msg.type)
        if not kind:
            return
        loc = msg.location
        location = f"{loc.get('url', '')}:{loc.get('lineNumber', '')}".strip(":")
        session.add(VerifError(kind=kind, message=msg.text, location=location))

    def _on_pageerror(err):
        session.add(VerifError(kind="pageerror", message=str(err)))

    def _on_response(response):
        # 网络 4xx/5xx
        status = response.status
        if status >= 400:
            session.add(VerifError(
                kind="network",
                message=f"{response.request.method} {response.url} → HTTP {status}",
                location=response.url,
                detail=f"HTTP {status}",
            ))
            return
        # API 业务码非成功（仅尝试 JSON 响应）
        url = response.url
        if "/api/" not in url:
            return
        try:
            body = response.json()
        except Exception:
            return
        biz = parse_business_error(body)
        if biz:
            session.add(VerifError(
                kind="business",
                message=f"{response.request.method} {url} → {biz}",
                location=url,
                detail=biz,
            ))

    page.on("console", _on_console)
    page.on("pageerror", _on_pageerror)
    page.on("response", _on_response)
    return session
