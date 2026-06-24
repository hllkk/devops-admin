"""验证工具配置：集中管理 URL、业务码、超时、账号（仅从环境变量读取）。"""
from __future__ import annotations

import os
import urllib.request
from dataclasses import dataclass

# 后端业务成功码（来自前端 .env VITE_SERVICE_SUCCESS_CODE）
SUCCESS_CODE = "0000"
FRONTEND_URL = "http://localhost:9527"
LOGIN_PATH = "/login"
DEFAULT_TIMEOUT_MS = 15000


def _require_env(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise RuntimeError(
            f"环境变量 {name} 未设置。请用 `export {name}=...` 或命令行前缀注入，"
            f"不要写入源码或配置文件。"
        )
    return value


@dataclass(frozen=True)
class Account:
    username: str
    password: str

    @classmethod
    def from_env(cls) -> "Account":
        return cls(
            username=_require_env("DEVOPS_TEST_USER"),
            password=_require_env("DEVOPS_TEST_PASS"),
        )


def is_success_code(code) -> bool:
    """后端业务码是否表示成功（字符串/整数/None 统一转字符串比较）。"""
    return str(code) == SUCCESS_CODE


def check_services_reachable() -> None:
    """启动前探测前端可达（后端由 Vite 代理转发，无需直连）。不可达则快速失败给出明确错误。"""
    try:
        urllib.request.urlopen(FRONTEND_URL, timeout=5).close()
    except Exception as e:
        raise RuntimeError(
            f"前端 {FRONTEND_URL} 不可达，请确认 dev 服务已启动 (pnpm dev)。原始错误: {e}"
        )
