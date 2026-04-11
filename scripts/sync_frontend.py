#!/usr/bin/env python3
"""前端 Upstream 同步工具 - 主脚本"""

import sys
import os

# 添加模块路径
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from sync_lib.git_ops import (
    get_current_branch,
    get_branch_hash,
    has_uncommitted_changes,
    stash_changes,
    stash_pop,
    checkout_branch,
    fetch_remote,
    merge_branch,
    push_branch,
)
from sync_lib.config import (
    load_config,
    save_config,
    get_config,
    get_frontend_dir,
)
from sync_lib.ui import (
    show_header,
    show_main_menu,
    show_config_menu,
    show_message,
    show_error,
    show_success,
    show_warning,
    show_confirm,
    show_spinner,
)
from sync_lib.preview import preview_command
from sync_lib.conflict import handle_conflicts, has_conflicts
from sync_lib.rollback import (
    create_sync_record,
    add_history_record,
    rollback_command,
    history_command,
)


def sync_to_main() -> bool:
    """
    仅同步 upstream 到 main 分支

    Returns:
        是否成功
    """
    frontend_dir = get_frontend_dir()
    upstream_remote = get_config("upstream_remote", "upstream")
    upstream_branch = get_config("upstream_branch", "main")
    main_branch = get_config("main_branch", "main")

    current_branch = get_current_branch(frontend_dir)
    stashed = False

    # 切换到 main 分支
    show_spinner("切换到 main 分支...", checkout_branch, main_branch, frontend_dir)

    # 检查未提交更改
    if has_uncommitted_changes(frontend_dir):
        show_warning("存在未提交的更改")
        if show_confirm("是否暂存这些更改？"):
            stashed = stash_changes(frontend_dir)

    # 获取上游更新
    show_spinner("获取 upstream 更新...", fetch_remote, upstream_remote, frontend_dir)

    # 检查是否有更新
    upstream_hash = get_branch_hash(f"{upstream_remote}/{upstream_branch}", frontend_dir)
    local_hash = get_branch_hash(main_branch, frontend_dir)

    if upstream_hash == local_hash:
        show_message("main 分支已是最新版本", "success")
        checkout_branch(current_branch, frontend_dir)
        if stashed:
            stash_pop(frontend_dir)
        return True

    # 记录当前状态用于历史
    main_before = local_hash

    # 合并上游
    success, error = merge_branch(f"{upstream_remote}/{upstream_branch}", frontend_dir)

    if not success:
        show_warning("合并过程中发现冲突")
        success, error = handle_conflicts(frontend_dir)

        if not success:
            show_error(f"冲突处理失败: {error}")
            checkout_branch(current_branch, frontend_dir)
            return False

    # 推送
    show_spinner("推送 main 分支...", push_branch, main_branch, frontend_dir, "origin")

    # 记录历史
    main_after = get_branch_hash(main_branch, frontend_dir)
    record = create_sync_record(
        upstream_hash=upstream_hash,
        main_before=main_before,
        main_after=main_after,
        status="success"
    )
    add_history_record(record)

    # 恢复原始分支
    checkout_branch(current_branch, frontend_dir)
    if stashed:
        stash_pop(frontend_dir)

    show_success("main 分支同步完成")
    return True


def sync_and_merge_to_dev() -> bool:
    """
    同步 upstream 到 main 并合并到 dev

    Returns:
        是否成功
    """
    frontend_dir = get_frontend_dir()
    upstream_remote = get_config("upstream_remote", "upstream")
    upstream_branch = get_config("upstream_branch", "main")
    main_branch = get_config("main_branch", "main")
    dev_branch = get_config("dev_branch", "dev")

    current_branch = get_current_branch(frontend_dir)
    stashed = False

    # 检查未提交更改
    if has_uncommitted_changes(frontend_dir):
        show_warning("存在未提交的更改")
        if show_confirm("是否暂存这些更改？"):
            stashed = stash_changes(frontend_dir)

    # 切换到 main 分支
    show_spinner("切换到 main 分支...", checkout_branch, main_branch, frontend_dir)

    # 获取上游更新
    show_spinner("获取 upstream 更新...", fetch_remote, upstream_remote, frontend_dir)

    # 检查是否有更新
    upstream_hash = get_branch_hash(f"{upstream_remote}/{upstream_branch}", frontend_dir)
    local_hash = get_branch_hash(main_branch, frontend_dir)

    if upstream_hash == local_hash:
        show_message("main 分支已是最新版本", "success")
        checkout_branch(current_branch, frontend_dir)
        if stashed:
            stash_pop(frontend_dir)
        return True

    # 记录当前状态
    main_before = local_hash
    dev_before = get_branch_hash(dev_branch, frontend_dir)

    # 合并上游到 main
    success, error = merge_branch(f"{upstream_remote}/{upstream_branch}", frontend_dir)

    if not success:
        show_warning("main 分支合并发现冲突")
        success, error = handle_conflicts(frontend_dir)

        if not success:
            show_error(f"冲突处理失败: {error}")
            checkout_branch(current_branch, frontend_dir)
            return False

    # 推送 main
    show_spinner("推送 main 分支...", push_branch, main_branch, frontend_dir, "origin")
    show_success("main 分支已更新")

    main_after = get_branch_hash(main_branch, frontend_dir)

    # 切换到 dev 分支
    show_spinner("切换到 dev 分支...", checkout_branch, dev_branch, frontend_dir)

    # 合并 main 到 dev
    success, error = merge_branch(main_branch, frontend_dir)

    if not success:
        show_warning("dev 分支合并发现冲突")
        success, error = handle_conflicts(frontend_dir)

        if not success:
            show_error(f"冲突处理失败: {error}")
            # 记录部分成功的历史
            record = create_sync_record(
                upstream_hash=upstream_hash,
                main_before=main_before,
                main_after=main_after,
                dev_before=dev_before,
                status="conflict"
            )
            add_history_record(record)
            return False

    # 推送 dev
    show_spinner("推送 dev 分支...", push_branch, dev_branch, frontend_dir, "origin")

    dev_after = get_branch_hash(dev_branch, frontend_dir)

    # 记录历史
    record = create_sync_record(
        upstream_hash=upstream_hash,
        main_before=main_before,
        main_after=main_after,
        dev_before=dev_before,
        dev_after=dev_after,
        status="success"
    )
    add_history_record(record)

    # 恢复原始分支
    checkout_branch(current_branch, frontend_dir)
    if stashed:
        stash_pop(frontend_dir)

    show_success("同步完成")
    return True


def show_config():
    """显示和修改配置"""
    config = load_config()

    from rich import print as rprint
    from rich.panel import Panel
    from rich.prompt import Prompt

    while True:
        choice = show_config_menu()

        if choice == "0":
            break
        elif choice == "1":
            # 自动接受模式
            patterns = config.get("auto_accept_patterns", [])
            rprint(Panel(
                '\n'.join(f"  {p}" for p in patterns),
                title="[bold]当前自动接受模式[/bold]"
            ))
            new_patterns = Prompt.ask(
                "输入新模式（逗号分隔，或留空保持不变）",
                default=""
            )
            if new_patterns:
                config["auto_accept_patterns"] = [p.strip() for p in new_patterns.split(',')]
                rprint("[green]模式已更新[/green]")

        elif choice == "2":
            # 分支名称
            rprint(f"[dim]当前 main 分支: {config.get('main_branch', 'main')}[/dim]")
            rprint(f"[dim]当前 dev 分支: {config.get('dev_branch', 'dev')}[/dim]")

            new_main = Prompt.ask("main 分支名称", default=config.get('main_branch', 'main'))
            new_dev = Prompt.ask("dev 分支名称", default=config.get('dev_branch', 'dev'))

            config["main_branch"] = new_main
            config["dev_branch"] = new_dev

        elif choice == "3":
            # 日志路径
            rprint(f"[dim]当前日志目录: {config.get('log_dir', '/home/devops-admin/logs')}[/dim]")
            new_log_dir = Prompt.ask("日志目录路径", default=config.get('log_dir', '/home/devops-admin/logs'))
            config["log_dir"] = new_log_dir

        elif choice == "4":
            # 保存
            if save_config(config):
                show_success("配置已保存")
            else:
                show_error("保存失败")


def main():
    """主入口"""
    try:
        while True:
            show_header()
            choice = show_main_menu()

            if choice == "0":
                show_message("再见！", "info")
                break
            elif choice == "1":
                sync_to_main()
            elif choice == "2":
                sync_and_merge_to_dev()
            elif choice == "3":
                preview_command()
            elif choice == "4":
                rollback_command()
            elif choice == "5":
                history_command()
            elif choice == "6":
                show_config()

            # 等待用户确认继续
            if choice != "0":
                show_confirm("\n按 Enter 继续...", default=True)

    except KeyboardInterrupt:
        show_message("\n已取消操作", "warning")
    except Exception as e:
        show_error(f"发生错误: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    main()