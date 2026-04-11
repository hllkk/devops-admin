"""用户界面组件模块"""

import sys

# 尝试导入 rich，如果不可用则使用 fallback
from typing import Optional

try:
    from rich.console import Console
    from rich.panel import Panel
    from rich.table import Table
    from rich.prompt import Prompt, Confirm
    from rich.progress import Progress, SpinnerColumn, TextColumn
    from rich import print as rprint
    RICH_AVAILABLE = True
    console = Console()
except ImportError:
    RICH_AVAILABLE = False
    console = None
    rprint = print


def _simple_prompt(message: str, choices: list = None, default: str = None) -> str:
    """简单的提示函数（无 rich 时的 fallback）"""
    if choices:
        print(f"{message} [{', '.join(choices)}]")
    else:
        print(message)

    if default:
        print(f"默认: {default}")

    result = input("> ").strip()
    if not result and default:
        return default
    return result


def _simple_confirm(message: str, default: bool = True) -> bool:
    """简单的确认函数（无 rich 时的 fallback）"""
    hint = "[Y/n]" if default else "[y/N]"
    print(f"{message} {hint}")
    result = input("> ").strip().lower()

    if not result:
        return default
    return result in ('y', 'yes', 'true', '1')


def show_header():
    """显示标题"""
    if RICH_AVAILABLE:
        console.clear()
        rprint(Panel(
            "[bold cyan]前端 Upstream 同步工具[/bold cyan] [dim]v1.0[/dim]",
            border_style="blue"
        ))
    else:
        print("\n" + "=" * 50)
        print("      前端 Upstream 同步工具 v1.0")
        print("=" * 50 + "\n")


def show_main_menu() -> str:
    """显示主菜单并获取用户选择"""
    if RICH_AVAILABLE:
        menu_text = """
[1] 📥 同步 upstream 到 main
[2] 🔄 同步并合并到 dev
[3] 👁 预览 upstream 变更
[4] ⏪ 回滚最近同步
[5] 📋 查看同步历史
[6] 🔧 配置设置
[0] 🚪 退出
"""
        rprint(Panel(menu_text, title="[bold]主菜单[/bold]", border_style="green"))
        return Prompt.ask("\n请选择操作", choices=["0", "1", "2", "3", "4", "5", "6"], default="2")
    else:
        print("\n主菜单:")
        print("  [1] 同步 upstream 到 main")
        print("  [2] 同步并合并到 dev")
        print("  [3] 预览 upstream 变更")
        print("  [4] 回滚最近同步")
        print("  [5] 查看同步历史")
        print("  [6] 配置设置")
        print("  [0] 退出")
        return _simple_prompt("\n请选择操作", ["0", "1", "2", "3", "4", "5", "6"], "2")


def show_config_menu() -> str:
    """显示配置菜单"""
    if RICH_AVAILABLE:
        menu_text = """
[1] 自动接受的文件模式
[2] 分支名称设置
[3] 日志路径设置
[4] 保存配置
[0] 返回主菜单
"""
        rprint(Panel(menu_text, title="[bold]🔧 配置设置[/bold]", border_style="yellow"))
        return Prompt.ask("\n请选择", choices=["0", "1", "2", "3", "4"], default="0")
    else:
        print("\n配置设置:")
        print("  [1] 自动接受的文件模式")
        print("  [2] 分支名称设置")
        print("  [3] 日志路径设置")
        print("  [4] 保存配置")
        print("  [0] 返回主菜单")
        return _simple_prompt("\n请选择", ["0", "1", "2", "3", "4"], "0")


def show_message(message: str, style: str = "info"):
    """显示消息"""
    if RICH_AVAILABLE:
        style_colors = {
            "info": "cyan",
            "success": "green",
            "warning": "yellow",
            "error": "red"
        }
        color = style_colors.get(style, "white")
        rprint(f"[{color}]{message}[/{color}]")
    else:
        # 清除 rich 标记
        import re
        clean_message = re.sub(r'\[.*?\]', '', message)
        prefix = {"info": "[INFO]", "success": "[OK]", "warning": "[WARN]", "error": "[ERR]"}[style]
        print(f"{prefix} {clean_message}")


def show_error(message: str):
    """显示错误消息"""
    if RICH_AVAILABLE:
        rprint(f"[bold red]❌ {message}[/bold red]")
    else:
        import re
        clean_message = re.sub(r'\[.*?\]', '', message)
        print(f"[ERROR] {clean_message}")


def show_success(message: str):
    """显示成功消息"""
    if RICH_AVAILABLE:
        rprint(f"[bold green]✅ {message}[/bold green]")
    else:
        import re
        clean_message = re.sub(r'\[.*?\]', '', message)
        print(f"[OK] {clean_message}")


def show_warning(message: str):
    """显示警告消息"""
    if RICH_AVAILABLE:
        rprint(f"[bold yellow]⚠️ {message}[/bold yellow]")
    else:
        import re
        clean_message = re.sub(r'\[.*?\]', '', message)
        print(f"[WARN] {clean_message}")


def show_confirm(message: str, default: bool = True) -> bool:
    """显示确认提示"""
    if RICH_AVAILABLE:
        return Confirm.ask(message, default=default)
    else:
        import re
        clean_message = re.sub(r'\[.*?\]', '', message)
        return _simple_confirm(clean_message, default)


def show_spinner(message: str, action, *args, **kwargs):
    """显示加载动画并执行操作"""
    if RICH_AVAILABLE:
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
            transient=True
        ) as progress:
            task = progress.add_task(message, total=None)
            result = action(*args, **kwargs)
            progress.remove_task(task)
        return result
    else:
        print(f"... {message}")
        result = action(*args, **kwargs)
        print("  完成")
        return result


def show_file_table(files: list, title: str = "文件列表"):
    """显示文件表格"""
    if RICH_AVAILABLE:
        table = Table(title=title, show_header=True)
        table.add_column("序号", style="cyan", width=6)
        table.add_column("文件路径", style="white")

        for i, file in enumerate(files, 1):
            table.add_row(str(i), file)

        rprint(table)
    else:
        print(f"\n{title}:")
        for i, file in enumerate(files, 1):
            print(f"  {i:3d}  {file}")


def show_preview_summary(stats: dict):
    """显示预览摘要"""
    if RICH_AVAILABLE:
        content = f"""
共 {stats.get('total', 0)} 个文件变更
• 新增: {stats.get('added', 0)} 个
• 修改: {stats.get('modified', 0)} 个
• 删除: {stats.get('deleted', 0)} 个
"""
        rprint(Panel(content, title="[bold]📊 Upstream 变更预览[/bold]", border_style="blue"))
    else:
        print("\nUpstream 变更预览:")
        print(f"  共 {stats.get('total', 0)} 个文件变更")
        print(f"  • 新增: {stats.get('added', 0)} 个")
        print(f"  • 修改: {stats.get('modified', 0)} 个")
        print(f"  • 删除: {stats.get('deleted', 0)} 个")


def show_diff_preview(file: str, diff: str, max_lines: int = 50):
    """显示文件差异预览"""
    lines = diff.split('\n')[:max_lines]
    diff_text = '\n'.join(lines)
    if len(diff.split('\n')) > max_lines:
        diff_text += '\n... (省略更多内容)'

    if RICH_AVAILABLE:
        rprint(Panel(
            diff_text,
            title=f"[bold]📝 {file}[/bold]",
            border_style="yellow"
        ))
    else:
        print(f"\n--- {file} ---")
        print(diff_text)
        print("---")


def ask_conflict_action(file: str) -> str:
    """询问冲突文件的处理方式"""
    if RICH_AVAILABLE:
        rprint(f"\n[bold yellow]冲突文件: {file}[/bold yellow]")
        rprint("[dim]a: 接受上游版本 | b: 保留本地版本 | m: 手动编辑 | s: 跳过[/dim]")
        return Prompt.ask("选择操作", choices=["a", "b", "m", "s"], default="a")
    else:
        print(f"\n冲突文件: {file}")
        print("  a: 接受上游版本")
        print("  b: 保留本地版本")
        print("  m: 手动编辑")
        print("  s: 跳过")
        return _simple_prompt("选择操作", ["a", "b", "m", "s"], "a")


def ask_file_preview(files: list) -> Optional[int]:
    """询问要查看哪个文件的差异"""
    show_file_table(files, "变更文件列表")

    if RICH_AVAILABLE:
        rprint("\n[dim]输入序号查看差异，输入 0 返回[/dim]")
        choice = Prompt.ask("选择文件", default="0")
    else:
        print("\n输入序号查看差异，输入 0 返回")
        choice = _simple_prompt("选择文件", default="0")

    if choice == "0":
        return None

    try:
        index = int(choice)
        if 1 <= index <= len(files):
            return index - 1
    except ValueError:
        pass

    return None


def ask_rollback_record(records: list) -> Optional[int]:
    """询问要回滚哪条记录"""
    if RICH_AVAILABLE:
        table = Table(title="同步历史记录", show_header=True)
        table.add_column("序号", style="cyan", width=6)
        table.add_column("时间", style="white", width=20)
        table.add_column("状态", style="white", width=12)
        table.add_column("Upstream Hash", style="dim", width=12)

        for i, record in enumerate(records, 1):
            status = record.get("status", "unknown")
            status_display = {
                "success": "[green]成功[/green]",
                "rolled_back": "[yellow]已回滚[/yellow]",
                "conflict": "[red]冲突[/red]"
            }.get(status, status)

            table.add_row(
                str(i),
                record.get("timestamp", "")[:19],
                status_display,
                record.get("upstream_hash", "")[:12]
            )

        rprint(table)
        rprint("\n[dim]输入序号选择要回滚的记录，输入 0 返回[/dim]")
        choice = Prompt.ask("选择记录", default="0")
    else:
        print("\n同步历史记录:")
        for i, record in enumerate(records, 1):
            status = record.get("status", "unknown")
            print(f"  {i:3d}  {record.get('timestamp', '')[:19]}  {status}  {record.get('upstream_hash', '')[:12]}")

        print("\n输入序号选择要回滚的记录，输入 0 返回")
        choice = _simple_prompt("选择记录", default="0")

    if choice == "0":
        return None

    try:
        index = int(choice)
        if 1 <= index <= len(records):
            return index - 1
    except ValueError:
        pass

    return None