#!/bin/bash
# 冲突分析脚本 - 生成建议

FRONTEND_DIR="/home/devops-admin/frontend"
CONFLICT_FLAG="/home/devops-admin/logs/.sync-conflict-pending"

if [ ! -f "$CONFLICT_FLAG" ]; then
    echo "无待处理的冲突"
    exit 0
fi

cd "$FRONTEND_DIR"

echo "========== 冲突分析报告 =========="
echo ""

CONFLICTS=$(git diff --name-only --diff-filter=U 2>/dev/null || true)

for file in $CONFLICTS; do
    echo "📁 文件: $file"
    echo "----------------------------------------"
    
    case $file in
        package.json)
            echo "📝 类型: 依赖配置文件"
            echo "💡 建议: 检查依赖版本差异"
            echo "   - 上游新增依赖: 接受上游版本"
            echo "   - 项目自定义依赖: 保留本地版本"
            echo ""
            # 显示差异
            echo "🔍 依赖差异预览:"
            git diff "$file" 2>/dev/null | head -30
            ;;
        pnpm-lock.yaml)
            echo "📝 类型: 依赖锁定文件"
            echo "💡 建议: 通常使用上游版本，然后运行 pnpm install"
            ;;
        *.vue|*.ts|*.tsx|*.js|*.jsx)
            echo "📝 类型: 源代码文件"
            echo "💡 建议: 需要人工审查代码差异"
            echo ""
            echo "🔍 冲突内容:"
            git diff "$file" 2>/dev/null | head -50
            ;;
        *.scss|*.css)
            echo "📝 类型: 样式文件"
            echo "💡 建议: 检查样式差异，通常保留最新修改"
            ;;
        *.json)
            echo "📝 类型: 配置文件"
            echo "💡 建议: 检查配置差异，合并双方修改"
            ;;
        *)
            echo "📝 类型: 其他文件"
            echo "💡 建议: 检查文件内容后决定"
            ;;
    esac
    echo ""
    echo "========================================"
    echo ""
done

echo "📋 处理命令:"
echo "  接受上游版本: git checkout --theirs <文件>"
echo "  保留本地版本: git checkout --ours <文件>"
echo "  完成后提交:   git add . && git commit -m \"chore: 解决冲突\""
