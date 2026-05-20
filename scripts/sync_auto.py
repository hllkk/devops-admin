#!/usr/bin/env python3
"""前端自动化升级脚本 - 非交互式版本

使用方式:
    python3 sync_auto.py                    # 同步到 main 并合并到 dev
    python3 sync_auto.py --main-only        # 仅同步到 main
    python3 sync_auto.py --preview          # 预览变更
    python3 sync_auto.py --dry-run          # 模拟运行（不实际执行）
    python3 sync_auto.py --restore-local    # 合并后恢复本地文件
"""

import argparse
import subprocess
import sys
import os
import json
from pathlib import Path
from datetime import datetime

# 配置
DEFAULT_CONFIG = {
    "frontend_dir": "/home/devops-admin/frontend",
    "upstream_remote": "upstream",
    "upstream_branch": "main",
    "main_branch": "main",
    "dev_branch": "dev",
    "auto_accept_patterns": [
        "pnpm-lock.yaml",
        "package-lock.json",
        "yarn.lock",
        "*.md",
        "package.json"
    ],
    "keep_local_patterns": [
        "src/router/elegant/*.ts",
        "src/typings/*.d.ts",
        ".env*",
        "vite.config.ts"
    ],
    "commit_message": "chore: 同步 upstream"
}

CONFIG_FILE = Path(__file__).parent / "sync_config.json"


def load_config():
    """加载配置"""
    if CONFIG_FILE.exists():
        with open(CONFIG_FILE, 'r', encoding='utf-8') as f:
            return {**DEFAULT_CONFIG, **json.load(f)}
    return DEFAULT_CONFIG


def run_cmd(cmd, cwd=None, check=True, capture=True):
    """执行命令"""
    result = subprocess.run(
        cmd,
        cwd=cwd,
        capture_output=capture,
        text=True,
        check=check
    )
    if capture:
        return result.stdout.strip(), result.stderr.strip()
    return result.returncode == 0


def log(level, message):
    """日志输出"""
    colors = {
        "info": "\033[36m",
        "success": "\033[32m",
        "warning": "\033[33m",
        "error": "\033[31m",
        "reset": "\033[0m"
    }
    prefix = {"info": "[INFO]", "success": "[OK]", "warning": "[WARN]", "error": "[ERR]"}
    print(f"{colors.get(level, '')}{prefix.get(level, '')} {message}{colors['reset']}")


def get_current_branch(cwd):
    """获取当前分支"""
    stdout, _ = run_cmd(["git", "branch", "--show-current"], cwd=cwd)
    return stdout


def get_branch_hash(branch, cwd):
    """获取分支 hash"""
    stdout, _ = run_cmd(["git", "rev-parse", branch], cwd=cwd)
    return stdout[:12] if stdout else None


def has_uncommitted_changes(cwd):
    """检查未提交更改"""
    result = subprocess.run(
        ["git", "diff-index", "--quiet", "HEAD", "--"],
        cwd=cwd,
        capture_output=True
    )
    return result.returncode != 0


def matches_pattern(filename, patterns):
    """检查文件匹配模式"""
    import fnmatch
    basename = os.path.basename(filename)
    for pattern in patterns:
        if fnmatch.fnmatch(basename, pattern) or fnmatch.fnmatch(filename, pattern):
            return True
    return False


def get_conflict_files(cwd):
    """获取冲突文件列表"""
    stdout, _ = run_cmd(["git", "diff", "--name-only", "--diff-filter=U"], cwd=cwd, check=False)
    return stdout.split('\n') if stdout else []


def resolve_conflict_auto(file, cwd, config):
    """自动解决冲突"""
    # 检查是否应该保留本地版本
    keep_local = config.get("keep_local_patterns", [])
    auto_accept = config.get("auto_accept_patterns", [])

    if matches_pattern(file, keep_local):
        log("info", f"  保留本地: {file}")
        run_cmd(["git", "checkout", "--ours", file], cwd=cwd)
        return "ours"

    if matches_pattern(file, auto_accept):
        log("info", f"  接受上游: {file}")
        run_cmd(["git", "checkout", "--theirs", file], cwd=cwd)
        return "theirs"

    # 默认接受上游版本（安全策略：框架升级通常需要上游版本）
    log("warning", f"  默认接受上游: {file}")
    run_cmd(["git", "checkout", "--theirs", file], cwd=cwd)
    return "theirs"


def restore_local_files(cwd, config):
    """恢复本地生成的文件"""
    keep_local = config.get("keep_local_patterns", [])
    restored = []

    # 查找匹配的文件并恢复
    for pattern in keep_local:
        # 使用 git ls-files 查找匹配文件
        stdout, _ = run_cmd(["git", "ls-files", "--modified"], cwd=cwd, check=False)
        for file in stdout.split('\n') if stdout else []:
            if matches_pattern(file, [pattern]):
                run_cmd(["git", "restore", file], cwd=cwd, check=False)
                restored.append(file)
                log("info", f"  已恢复本地文件: {file}")

    return restored


def sync_to_main(config, dry_run=False, restore_local=False):
    """同步 upstream 到 main"""
    cwd = config["frontend_dir"]
    upstream = config["upstream_remote"]
    upstream_branch = config["upstream_branch"]
    main_branch = config["main_branch"]

    # 记录当前状态
    original_branch = get_current_branch(cwd)
    log("info", f"当前分支: {original_branch}")

    # 检查未提交更改
    if has_uncommitted_changes(cwd):
        log("warning", "存在未提交的更改，将自动暂存...")
        run_cmd(["git", "stash", "push", "-m", "auto-stash-sync"], cwd=cwd)

    if dry_run:
        log("info", "[模拟] 将执行以下操作:")
        log("info", f"  1. 切换到 {main_branch}")
        log("info", f"  2. Fetch {upstream}")
        log("info", f"  3. Merge {upstream}/{upstream_branch}")
        log("info", f"  4. Push {main_branch}")
        log("info", f"  5. 切换回 {original_branch}")
        return True

    # 切换到 main
    log("info", f"切换到 {main_branch}...")
    run_cmd(["git", "checkout", main_branch], cwd=cwd)

    # Fetch upstream
    log("info", f"获取 {upstream} 更新...")
    run_cmd(["git", "fetch", upstream], cwd=cwd)

    # 检查是否有更新
    upstream_hash = get_branch_hash(f"{upstream}/{upstream_branch}", cwd)
    local_hash = get_branch_hash(main_branch, cwd)

    if upstream_hash == local_hash:
        log("success", f"{main_branch} 已是最新版本 ({upstream_hash})")
        run_cmd(["git", "checkout", original_branch], cwd=cwd)
        return True

    log("info", f"上游版本: {upstream_hash}, 本地版本: {local_hash}")

    # Merge upstream
    log("info", f"合并 {upstream}/{upstream_branch}...")
    _, stderr = run_cmd(["git", "merge", f"{upstream}/{upstream_branch}", "--no-edit"], cwd=cwd, check=False)

    # 处理冲突
    conflicts = get_conflict_files(cwd)
    if conflicts:
        log("warning", f"发现 {len(conflicts)} 个冲突文件")
        for file in conflicts:
            if file:
                resolve_conflict_auto(file, cwd, config)

        # 添加已解决的文件
        run_cmd(["git", "add", "-A"], cwd=cwd)

    # 提交
    commit_msg = f"{config['commit_message']} v{detect_upstream_version(cwd, upstream, upstream_branch)}"
    run_cmd(["git", "commit", "-m", commit_msg], cwd=cwd, check=False)

    # 恢复本地生成的文件
    if restore_local:
        log("info", "恢复本地生成的文件...")
        restore_local_files(cwd, config)

    # Push
    log("info", f"推送 {main_branch}...")
    run_cmd(["git", "push", "origin", main_branch], cwd=cwd)

    log("success", f"{main_branch} 已同步到 {upstream_hash}")

    # 切换回原分支
    run_cmd(["git", "checkout", original_branch], cwd=cwd)

    # 恢复 stash
    run_cmd(["git", "stash", "pop"], cwd=cwd, check=False)

    return True


def merge_to_dev(config, dry_run=False, restore_local=False):
    """合并 main 到 dev"""
    cwd = config["frontend_dir"]
    main_branch = config["main_branch"]
    dev_branch = config["dev_branch"]

    original_branch = get_current_branch(cwd)

    if dry_run:
        log("info", "[模拟] 将执行以下操作:")
        log("info", f"  1. 切换到 {dev_branch}")
        log("info", f"  2. Merge {main_branch}")
        log("info", f"  3. Push {dev_branch}")
        return True

    # 切换到 dev
    log("info", f"切换到 {dev_branch}...")
    run_cmd(["git", "checkout", dev_branch], cwd=cwd)

    # Merge main
    log("info", f"合并 {main_branch}...")
    _, stderr = run_cmd(["git", "merge", main_branch, "--no-edit"], cwd=cwd, check=False)

    # 处理冲突
    conflicts = get_conflict_files(cwd)
    if conflicts:
        log("warning", f"发现 {len(conflicts)} 个冲突文件")
        for file in conflicts:
            if file:
                resolve_conflict_auto(file, cwd, config)

        run_cmd(["git", "add", "-A"], cwd=cwd)

    # 恢复本地生成的文件
    if restore_local:
        log("info", "恢复本地生成的文件...")
        restore_local_files(cwd, config)

    # 提交
    run_cmd(["git", "commit", "-m", f"chore: 合并 {main_branch} 到 {dev_branch}"], cwd=cwd, check=False)

    # Push
    log("info", f"推送 {dev_branch}...")
    run_cmd(["git", "push", "origin", dev_branch], cwd=cwd)

    log("success", f"{dev_branch} 已合并")

    # 切换回原分支
    run_cmd(["git", "checkout", original_branch], cwd=cwd)

    return True


def detect_upstream_version(cwd, upstream, branch):
    """检测上游版本号"""
    # 尝试从 tag 获取版本
    stdout, _ = run_cmd(["git", "describe", "--tags", f"{upstream}/{branch}"], cwd=cwd, check=False)
    if stdout and stdout.startswith('v'):
        return stdout.lstrip('v').split('-')[0]

    # 从 package.json 获取版本
    stdout, _ = run_cmd(["git", "show", f"{upstream}/{branch}:package.json"], cwd=cwd, check=False)
    if stdout:
        try:
            data = json.loads(stdout)
            return data.get("version", "unknown")
        except:
            pass

    return "unknown"


def preview_changes(config):
    """预览上游变更"""
    cwd = config["frontend_dir"]
    upstream = config["upstream_remote"]
    upstream_branch = config["upstream_branch"]
    main_branch = config["main_branch"]

    log("info", "获取上游更新...")
    run_cmd(["git", "fetch", upstream], cwd=cwd)

    upstream_hash = get_branch_hash(f"{upstream}/{upstream_branch}", cwd)
    local_hash = get_branch_hash(main_branch, cwd)

    version = detect_upstream_version(cwd, upstream, upstream_branch)

    print(f"\n{'='*60}")
    print(f"上游版本: {version}")
    print(f"上游 Hash: {upstream_hash}")
    print(f"本地 Hash: {local_hash}")
    print(f"{'='*60}\n")

    if upstream_hash == local_hash:
        log("success", "本地已是最新版本，无需更新")
        return

    # 统计变更
    stdout, _ = run_cmd(["git", "diff", "--stat", f"{main_branch}..{upstream}/{upstream_branch}"], cwd=cwd, check=False)
    if stdout:
        print("变更统计:")
        print(stdout)

    # 列出变更文件
    stdout, _ = run_cmd(["git", "diff", "--name-status", f"{main_branch}..{upstream}/{upstream_branch}"], cwd=cwd, check=False)
    if stdout:
        files = stdout.split('\n')
        added = [f for f in files if f.startswith('A')]
        modified = [f for f in files if f.startswith('M')]
        deleted = [f for f in files if f.startswith('D')]

        print(f"\n变更文件: {len(files)} 个")
        print(f"  新增: {len(added)}")
        print(f"  修改: {len(modified)}")
        print(f"  删除: {len(deleted)}")

    # 提交列表
    stdout, _ = run_cmd(["git", "log", "--oneline", f"{main_branch}..{upstream}/{upstream_branch}"], cwd=cwd, check=False)
    if stdout:
        commits = stdout.split('\n')
        print(f"\n新提交: {len(commits)} 个")
        for commit in commits[:10]:  # 只显示前10个
            print(f"  {commit}")
        if len(commits) > 10:
            print(f"  ... 还有 {len(commits) - 10} 个提交")


def main():
    parser = argparse.ArgumentParser(
        description="前端自动化升级脚本",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
    %(prog)s                    # 同步 upstream 到 main 并合并到 dev
    %(prog)s --main-only        # 仅同步到 main
    %(prog)s --preview          # 预览变更（不执行）
    %(prog)s --dry-run          # 模拟运行
    %(prog)s --restore-local    # 合并后恢复本地生成的文件
        """
    )

    parser.add_argument("--main-only", action="store_true", help="仅同步到 main 分支")
    parser.add_argument("--preview", action="store_true", help="预览上游变更")
    parser.add_argument("--dry-run", action="store_true", help="模拟运行（不实际执行）")
    parser.add_argument("--restore-local", action="store_true", help="合并后恢复本地生成的文件")
    parser.add_argument("--dir", type=str, help="指定前端目录路径")

    args = parser.parse_args()

    # 加载配置
    config = load_config()
    if args.dir:
        config["frontend_dir"] = args.dir

    # 检查目录是否存在
    if not os.path.exists(config["frontend_dir"]):
        log("error", f"前端目录不存在: {config['frontend_dir']}")
        sys.exit(1)

    # 检查是否在 git 仓库
    stdout, _ = run_cmd(["git", "remote", "-v"], cwd=config["frontend_dir"], check=False)
    if not stdout:
        log("error", "当前目录不是 git 仓库")
        sys.exit(1)

    try:
        if args.preview:
            preview_changes(config)
        elif args.main_only:
            sync_to_main(config, args.dry_run, args.restore_local)
        else:
            # 默认: 同步到 main 并合并到 dev
            sync_to_main(config, args.dry_run, args.restore_local)
            if not args.dry_run:
                merge_to_dev(config, args.dry_run, args.restore_local)

        log("success", "操作完成")

    except subprocess.CalledProcessError as e:
        log("error", f"命令执行失败: {e}")
        sys.exit(1)
    except Exception as e:
        log("error", f"发生错误: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()