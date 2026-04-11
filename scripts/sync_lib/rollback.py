"""回滚功能模块"""

import json
from datetime import datetime
from typing import Dict, List, Optional
from pathlib import Path

from .git_ops import (
    get_current_branch,
    checkout_branch,
    reset_hard,
    push_branch,
)
from .config import get_frontend_dir, get_log_dir, get_config
from .ui import (
    show_error,
    show_success,
    show_warning,
    show_message,
    show_spinner,
    ask_rollback_record,
    show_confirm
)


def get_history_file() -> Path:
    """获取历史记录文件路径"""
    return Path(get_log_dir()) / "sync-history.json"


def load_history() -> List[Dict]:
    """加载同步历史"""
    history_file = get_history_file()

    if not history_file.exists():
        return []

    with open(history_file, 'r', encoding='utf-8') as f:
        data = json.load(f)
        return data.get('records', [])


def save_history(records: List[Dict]) -> bool:
    """保存同步历史"""
    history_file = get_history_file()

    try:
        history_file.parent.mkdir(parents=True, exist_ok=True)
        with open(history_file, 'w', encoding='utf-8') as f:
            json.dump({'records': records}, f, indent=2, ensure_ascii=False)
        return True
    except Exception:
        return False


def add_history_record(record: Dict) -> bool:
    """添加同步历史记录"""
    records = load_history()
    records.append(record)

    # 限制记录数量
    max_records = get_config('max_history_records', 50)
    if len(records) > max_records:
        records = records[-max_records:]

    return save_history(records)


def create_sync_record(
    upstream_hash: str,
    main_before: str,
    main_after: str,
    dev_before: Optional[str] = None,
    dev_after: Optional[str] = None,
    status: str = "success"
) -> Dict:
    """创建同步记录"""
    timestamp = datetime.now().isoformat()
    record_id = f"sync-{timestamp[:10]}-{len(load_history()) + 1:03d}"

    return {
        "id": record_id,
        "timestamp": timestamp,
        "upstream_hash": upstream_hash,
        "main_before": main_before,
        "main_after": main_after,
        "dev_before": dev_before or "",
        "dev_after": dev_after or "",
        "status": status
    }


def show_sync_history() -> List[Dict]:
    """显示同步历史"""
    records = load_history()

    if not records:
        show_message("暂无同步历史记录", "info")
        return []

    from rich import print as rprint
    from rich.table import Table

    table = Table(title="同步历史记录", show_header=True)
    table.add_column("ID", style="dim", width=20)
    table.add_column("时间", style="white", width=20)
    table.add_column("状态", style="white", width=12)
    table.add_column("Upstream", style="cyan", width=10)
    table.add_column("Main", style="green", width=10)

    for record in records[-10:]:  # 最近 10 条
        status = record.get("status", "unknown")
        status_display = {
            "success": "[green]成功[/green]",
            "rolled_back": "[yellow]已回滚[/yellow]",
            "conflict": "[red]冲突[/red]"
        }.get(status, status)

        table.add_row(
            record.get("id", "")[:20],
            record.get("timestamp", "")[:19],
            status_display,
            record.get("upstream_hash", "")[:10],
            record.get("main_after", "")[:10]
        )

    rprint(table)
    return records


def rollback_sync(record_index: int) -> bool:
    """
    回滚到指定的同步记录

    Args:
        record_index: 记录索引（从最近记录列表中的位置）
    """
    records = load_history()

    if record_index >= len(records):
        show_error("无效的记录索引")
        return False

    record = records[record_index]

    if record.get("status") == "rolled_back":
        show_warning("该记录已被回滚")
        return False

    frontend_dir = get_frontend_dir()
    main_branch = get_config("main_branch", "main")
    dev_branch = get_config("dev_branch", "dev")

    # 确认回滚
    show_warning(f"即将回滚到: {record.get('timestamp', '')[:19]}")
    if not show_confirm("确定要回滚吗？此操作不可撤销"):
        show_message("已取消回滚", "info")
        return False

    original_branch = get_current_branch(frontend_dir)

    # 回滚 main 分支
    show_spinner("回滚 main 分支...", checkout_branch, main_branch, frontend_dir)
    main_before = record.get("main_before", "")

    if main_before:
        show_spinner(f"重置到 {main_before[:10]}...", reset_hard, main_before, frontend_dir)

    # 回滚 dev 分支（如果有）
    dev_before = record.get("dev_before", "")

    if dev_before:
        show_spinner("回滚 dev 分支...", checkout_branch, dev_branch, frontend_dir)
        show_spinner(f"重置到 {dev_before[:10]}...", reset_hard, dev_before, frontend_dir)

    # 恢复原始分支
    checkout_branch(original_branch, frontend_dir)

    # 更新记录状态
    records[record_index]["status"] = "rolled_back"
    save_history(records)

    show_success("回滚完成")

    # 提示是否需要 force push
    if show_confirm("是否需要 force push 到远程？（谨慎操作）", default=False):
        show_spinner("Force push main...", push_branch, main_branch, frontend_dir, "origin")
        if dev_before:
            checkout_branch(dev_branch, frontend_dir)
            show_spinner("Force push dev...", push_branch, dev_branch, frontend_dir, "origin")
            checkout_branch(original_branch, frontend_dir)
        show_warning("已 force push，请确保团队知晓此操作")

    return True


def rollback_command():
    """回滚命令入口"""
    records = show_sync_history()

    if not records:
        return

    # 只显示最近 10 条供选择
    recent_records = records[-10:]
    index = ask_rollback_record(recent_records)

    if index is not None:
        # 计算实际索引位置
        actual_index = len(records) - 10 + index
        rollback_sync(actual_index)


def history_command():
    """历史查看命令入口"""
    show_sync_history()