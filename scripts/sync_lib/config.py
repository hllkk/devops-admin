"""配置管理模块"""

import json
import os
from typing import Dict, Any, List
from pathlib import Path

# 默认配置
DEFAULT_CONFIG = {
    "frontend_dir": "/home/devops-admin/frontend",
    "log_dir": "/home/devops-admin/logs",
    "auto_accept_patterns": [
        "pnpm-lock.yaml",
        "package-lock.json",
        "yarn.lock",
        "*.md"
    ],
    "upstream_remote": "upstream",
    "upstream_branch": "main",
    "dev_branch": "dev",
    "main_branch": "main",
    "max_history_records": 50
}

# 配置文件路径
CONFIG_FILE = Path(__file__).parent.parent / "sync_config.json"


def load_config() -> Dict[str, Any]:
    """加载配置，不存在则创建默认配置"""
    if not CONFIG_FILE.exists():
        save_config(DEFAULT_CONFIG)
        return DEFAULT_CONFIG.copy()

    with open(CONFIG_FILE, 'r', encoding='utf-8') as f:
        return json.load(f)


def save_config(config: Dict[str, Any]) -> bool:
    """保存配置到文件"""
    try:
        with open(CONFIG_FILE, 'w', encoding='utf-8') as f:
            json.dump(config, f, indent=2, ensure_ascii=False)
        return True
    except Exception:
        return False


def get_config(key: str, default: Any = None) -> Any:
    """获取单个配置项"""
    config = load_config()
    return config.get(key, default)


def set_config(key: str, value: Any) -> bool:
    """设置单个配置项"""
    config = load_config()
    config[key] = value
    return save_config(config)


def matches_pattern(filename: str, patterns: List[str]) -> bool:
    """
    检查文件名是否匹配任一模式

    Args:
        filename: 文件名
        patterns: 模式列表（支持 * 通配符）
    """
    import fnmatch
    for pattern in patterns:
        if fnmatch.fnmatch(filename, pattern):
            return True
    return False


def is_auto_accept_file(filename: str) -> bool:
    """判断文件是否应该自动接受上游版本"""
    patterns = get_config("auto_accept_patterns", DEFAULT_CONFIG["auto_accept_patterns"])
    basename = os.path.basename(filename)
    return matches_pattern(basename, patterns)


def get_frontend_dir() -> str:
    """获取前端目录路径"""
    return get_config("frontend_dir", DEFAULT_CONFIG["frontend_dir"])


def get_log_dir() -> str:
    """获取日志目录路径"""
    return get_config("log_dir", DEFAULT_CONFIG["log_dir"])