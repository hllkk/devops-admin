"""报告渲染：把 ProbeSession + 功能检查渲染成终端文本，并管理截图路径。"""
from __future__ import annotations

import os
from dataclasses import dataclass

import probe

REPORTS_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "reports")


@dataclass(frozen=True)
class CheckResult:
    name: str
    passed: bool
    detail: str = ""


def screenshot_path(feature_name: str) -> str:
    """生成 reports/ 下唯一的截图路径（调用方负责创建目录）。"""
    safe = "".join(c if c.isalnum() or c in "-_" else "-" for c in feature_name)
    return os.path.join(REPORTS_DIR, f"{safe}.png")


def _mark(ok: bool) -> str:
    return "✅" if ok else "❌"


def render(feature_name: str, smoke_passed: bool, smoke_detail: str,
           checks: list, probe_obj: "probe.ProbeSession", screenshot_path: str) -> str:
    func_ok = probe_obj.has_errors is False and all(c.passed for c in checks)

    lines = []
    lines.append(f"========== 验证报告: {feature_name} ==========")
    lines.append(f"基线冒烟:    {_mark(smoke_passed)} {'PASS' if smoke_passed else 'FAIL'}  ({smoke_detail})")
    lines.append(f"功能检查:    {_mark(func_ok)} {'PASS' if func_ok else 'FAIL'}")
    lines.append(f"截图:        {screenshot_path}")
    lines.append("")

    if probe_obj.has_errors:
        lines.append("【错误清单】")
        summary = probe_obj.summary_by_kind()
        for err in probe_obj.errors:
            loc = f"  ({err.location})" if err.location else ""
            lines.append(f"[{err.kind}] {err.message}{loc}")
        lines.append(f"(按类型统计: {summary})")
        lines.append("")

    lines.append("【功能实现检查】")
    for c in checks:
        tail = f" {c.detail}" if c.detail else ""
        lines.append(f"- {c.name}: {_mark(c.passed)}{tail}")
    lines.append("=" * 38)
    return "\n".join(lines)
