// VibeCoding — Root: 目录检测 + 提示
'use strict';
const cwd = process.cwd();
if (cwd.includes('/frontend')) {
  console.log('[VibeCoding] 当前在 frontend/ 目录，使用前端配置');
} else if (cwd.includes('/backend')) {
  console.log('[VibeCoding] 当前在 backend/ 目录，使用后端配置');
} else {
  console.log('[VibeCoding] 根目录模式 — 前端开发请进入 frontend/, 后端开发请进入 backend/');
}
process.exit(0);