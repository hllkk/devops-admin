"""冲突处理模块"""

from typing import List, Tuple

from .git_ops import (
    get_conflict_files,
    checkout_file_version,
    add_files,
    commit,
)
from .config import is_auto_accept_file
from .ui import (
    show_warning,
    show_error,
    show_success,
    show_message,
    show_diff_preview,
    ask_conflict_action,
    show_spinner
)


def handle_conflicts(cwd: str) -> Tuple[bool, str]:
    """
    处理合并冲突

    Returns:
        (success, error_message)
    """
    conflict_files = get_conflict_files(cwd)

    if not conflict_files:
        return True, ""

    show_warning(f"发现 {len(conflict_files)} 个冲突文件")

    auto_processed = []
    manual_processed = []
    skipped = []

    for file in conflict_files:
        basename = file.split('/')[-1]

        if is_auto_accept_file(file):
            # 自动处理
            show_spinner(f"自动处理: {basename}", checkout_file_version, file, 'theirs', cwd)
            auto_processed.append(file)
            show_message(f"  自动接受上游版本: {basename}", "info")
        else:
            # 手动处理
            show_message(f"\n需要处理冲突文件: [bold]{file}[/bold]", "warning")

            # 显示差异预览（尝试读取文件内容）
            try:
                with open(f"{cwd}/{file}", 'r') as f:
                    content = f.read()
                    # 显示冲突标记区域
                    show_diff_preview(file, content[:2000])
            except Exception:
                pass

            action = ask_conflict_action(file)

            if action == 'a':
                checkout_file_version(file, 'theirs', cwd)
                manual_processed.append(file)
                show_message(f"  接受上游版本: {basename}", "success")
            elif action == 'b':
                checkout_file_version(file, 'ours', cwd)
                manual_processed.append(file)
                show_message(f"  保留本地版本: {basename}", "success")
            elif action == 'm':
                show_message("  请在其他终端手动编辑文件后继续", "warning")
                manual_processed.append(file)
            elif action == 's':
                skipped.append(file)
                show_message(f"  跳过: {basename}", "warning")

    # 检查是否所有冲突都已处理
    remaining_conflicts = get_conflict_files(cwd)

    if remaining_conflicts:
        show_error(f"仍有 {len(remaining_conflicts)} 个冲突未处理")
        show_message("请在解决所有冲突后手动执行 git commit", "warning")
        return False, "存在未解决的冲突"

    # 提交解决冲突后的更改
    processed_files = auto_processed + manual_processed

    if processed_files and not skipped:
        show_spinner("正在提交...", add_files, processed_files, cwd)
        commit("chore: 解决合并冲突", cwd)
        show_success("冲突已解决并提交")

    return True, ""


def has_conflicts(cwd: str) -> bool:
    """检查是否有冲突"""
    return len(get_conflict_files(cwd)) > 0