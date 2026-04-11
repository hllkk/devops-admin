"""预览功能模块"""

from typing import Dict, List, Tuple
import os

from .git_ops import (
    fetch_remote,
    get_diff_files,
    get_file_diff,
    get_branch_hash
)
from .config import get_frontend_dir, get_config
from .ui import (
    show_spinner,
    show_preview_summary,
    show_file_table,
    show_diff_preview,
    ask_file_preview,
    show_message,
)


def categorize_files(files: List[str]) -> Dict[str, List[Tuple[str, str]]]:
    """
    将变更文件按类型分类

    Returns:
        {'deps': [...], 'src': [...], 'styles': [...], 'config': [...]}
    """
    categories = {
        'deps': [],      # 依赖相关
        'src': [],       # 源代码
        'styles': [],    # 样式文件
        'config': [],    # 配置/文档
    }

    for file_line in files:
        if not file_line:
            continue

        # 解析 name-status 格式: "A\tfilename" 或 "M\tfilename"
        parts = file_line.split('\t')
        if len(parts) < 2:
            continue

        status, file = parts[0], parts[1]
        basename = os.path.basename(file)

        # 分类
        if basename in ['package.json', 'pnpm-lock.yaml', 'package-lock.json', 'yarn.lock']:
            categories['deps'].append((status, file))
        elif any(file.endswith(ext) for ext in ['.vue', '.ts', '.tsx', '.js', '.jsx']):
            categories['src'].append((status, file))
        elif any(file.endswith(ext) for ext in ['.scss', '.css', '.less']):
            categories['styles'].append((status, file))
        else:
            categories['config'].append((status, file))

    return categories


def parse_diff_stat(file_lines: List[str]) -> Dict[str, int]:
    """
    解析 diff --name-status 输出

    Returns:
        {'total': n, 'added': n, 'modified': n, 'deleted': n}
    """
    stats = {'total': 0, 'added': 0, 'modified': 0, 'deleted': 0}

    for line in file_lines:
        if not line.strip():
            continue

        status = line.split('\t')[0] if '\t' in line else ''

        if status == 'A':
            stats['added'] += 1
            stats['total'] += 1
        elif status == 'M':
            stats['modified'] += 1
            stats['total'] += 1
        elif status == 'D':
            stats['deleted'] += 1
            stats['total'] += 1

    return stats


def show_upstream_preview() -> Tuple[bool, List[str]]:
    """
    预览 upstream 变更

    Returns:
        (has_updates, changed_files)
    """
    frontend_dir = get_frontend_dir()
    upstream_remote = get_config("upstream_remote", "upstream")
    upstream_branch = get_config("upstream_branch", "main")
    main_branch = get_config("main_branch", "main")

    # 获取上游更新
    show_spinner("正在获取 upstream 更新...", fetch_remote, upstream_remote, frontend_dir)

    # 检查是否有更新
    upstream_hash = get_branch_hash(f"{upstream_remote}/{upstream_branch}", frontend_dir)
    local_hash = get_branch_hash(main_branch, frontend_dir)

    if upstream_hash == local_hash:
        show_message("main 分支已是最新版本，无需更新", "info")
        return False, []

    # 获取变更文件
    file_lines = get_diff_files(main_branch, f"{upstream_remote}/{upstream_branch}", frontend_dir)
    files = [line.split('\t')[1] if '\t' in line else line for line in file_lines if line]

    # 解析统计
    stats = parse_diff_stat(file_lines)

    # 显示摘要
    show_preview_summary(stats)

    # 分类显示
    categories = categorize_files(file_lines)

    from rich import print as rprint
    from rich.panel import Panel

    category_display = []
    if categories['deps']:
        category_display.append("[bold]📦 依赖变更:[/bold]")
        for status, f in categories['deps']:
            status_icon = {'A': '+', 'M': '~', 'D': '-'}.get(status, '?')
            category_display.append(f"  {status_icon} {f}")

    if categories['src']:
        category_display.append("[bold]📝 源代码变更:[/bold]")
        for status, f in categories['src']:
            status_icon = {'A': '+', 'M': '~', 'D': '-'}.get(status, '?')
            category_display.append(f"  {status_icon} {f}")

    if categories['styles']:
        category_display.append("[bold]🎨 样式文件变更:[/bold]")
        for status, f in categories['styles']:
            status_icon = {'A': '+', 'M': '~', 'D': '-'}.get(status, '?')
            category_display.append(f"  {status_icon} {f}")

    if categories['config']:
        category_display.append("[bold]📄 配置/文档变更:[/bold]")
        for status, f in categories['config']:
            status_icon = {'A': '+', 'M': '~', 'D': '-'}.get(status, '?')
            category_display.append(f"  {status_icon} {f}")

    if category_display:
        rprint(Panel('\n'.join(category_display), title="[bold]变更详情[/bold]", border_style="blue"))

    # 允许用户查看特定文件差异
    while True:
        choice = ask_file_preview(files)
        if choice is None:
            break

        file = files[choice]
        diff = get_file_diff(file, main_branch, f"{upstream_remote}/{upstream_branch}", frontend_dir)
        show_diff_preview(file, diff)

    return True, files


def preview_command():
    """预览命令入口"""
    show_upstream_preview()