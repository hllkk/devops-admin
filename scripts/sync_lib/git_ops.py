"""Git 操作封装模块"""

import subprocess
import os
from typing import Tuple, Optional, List


def run_git(args: List[str], cwd: Optional[str] = None, check: bool = True) -> Tuple[str, str]:
    """
    执行 git 命令并返回输出

    Args:
        args: git 命令参数列表
        cwd: 工作目录
        check: 是否检查命令成功

    Returns:
        (stdout, stderr) 元组
    """
    result = subprocess.run(
        ['git'] + args,
        cwd=cwd,
        capture_output=True,
        text=True,
        check=check
    )
    return result.stdout.strip(), result.stderr.strip()


def get_current_branch(cwd: str) -> str:
    """获取当前分支名称"""
    stdout, _ = run_git(['branch', '--show-current'], cwd=cwd)
    return stdout


def get_branch_hash(branch: str, cwd: str) -> str:
    """获取分支的 commit hash"""
    stdout, _ = run_git(['rev-parse', branch], cwd=cwd)
    return stdout


def has_uncommitted_changes(cwd: str) -> bool:
    """检查是否有未提交的更改"""
    result = subprocess.run(
        ['git', 'diff-index', '--quiet', 'HEAD', '--'],
        cwd=cwd,
        capture_output=True
    )
    return result.returncode != 0


def stash_changes(cwd: str, message: str = "") -> bool:
    """暂存未提交的更改"""
    msg = message or f"auto-stash-{os.popen('date +%Y%m%d%H%M%S').read().strip()}"
    try:
        run_git(['stash', 'push', '-m', msg], cwd=cwd)
        return True
    except subprocess.CalledProcessError:
        return False


def stash_pop(cwd: str) -> bool:
    """恢复暂存的更改"""
    try:
        run_git(['stash', 'pop'], cwd=cwd, check=False)
        return True
    except subprocess.CalledProcessError:
        return False


def checkout_branch(branch: str, cwd: str) -> bool:
    """切换到指定分支"""
    try:
        run_git(['checkout', branch], cwd=cwd)
        return True
    except subprocess.CalledProcessError:
        return False


def fetch_remote(remote: str, cwd: str) -> bool:
    """获取远程仓库更新"""
    try:
        run_git(['fetch', remote], cwd=cwd)
        return True
    except subprocess.CalledProcessError:
        return False


def merge_branch(branch: str, cwd: str, no_edit: bool = True) -> Tuple[bool, str]:
    """
    合并分支

    Returns:
        (success, error_message)
    """
    args = ['merge', branch]
    if no_edit:
        args.append('--no-edit')

    try:
        run_git(args, cwd=cwd)
        return True, ""
    except subprocess.CalledProcessError as e:
        return False, e.stderr or "合并失败"


def push_branch(branch: str, cwd: str, remote: str = "origin") -> bool:
    """推送分支到远程"""
    try:
        run_git(['push', remote, branch], cwd=cwd)
        return True
    except subprocess.CalledProcessError:
        return False


def get_conflict_files(cwd: str) -> List[str]:
    """获取冲突文件列表"""
    stdout, _ = run_git(['diff', '--name-only', '--diff-filter=U'], cwd=cwd, check=False)
    if stdout:
        return stdout.split('\n')
    return []


def checkout_file_version(file: str, version: str, cwd: str) -> bool:
    """
    检出文件的特定版本

    Args:
        version: 'ours' (本地) 或 'theirs' (上游)
    """
    try:
        run_git(['checkout', f'--{version}', file], cwd=cwd)
        return True
    except subprocess.CalledProcessError:
        return False


def add_files(files: List[str], cwd: str) -> bool:
    """添加文件到暂存区"""
    try:
        run_git(['add'] + files, cwd=cwd)
        return True
    except subprocess.CalledProcessError:
        return False


def commit(message: str, cwd: str) -> bool:
    """提交更改"""
    try:
        run_git(['commit', '-m', message], cwd=cwd)
        return True
    except subprocess.CalledProcessError:
        return False


def reset_hard(hash: str, cwd: str) -> bool:
    """硬重置到指定 commit"""
    try:
        run_git(['reset', '--hard', hash], cwd=cwd)
        return True
    except subprocess.CalledProcessError:
        return False


def get_diff_stat(branch1: str, branch2: str, cwd: str) -> str:
    """获取两个分支的差异统计"""
    stdout, _ = run_git(['diff', '--stat', branch1, branch2], cwd=cwd, check=False)
    return stdout


def get_diff_files(branch1: str, branch2: str, cwd: str) -> List[str]:
    """获取两个分支差异的文件列表"""
    stdout, _ = run_git(['diff', '--name-status', branch1, branch2], cwd=cwd, check=False)
    if stdout:
        return stdout.split('\n')
    return []


def get_file_diff(file: str, branch1: str, branch2: str, cwd: str) -> str:
    """获取特定文件的差异"""
    stdout, _ = run_git(['diff', branch1, branch2, '--', file], cwd=cwd, check=False)
    return stdout