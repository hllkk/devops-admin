# 浏览器验证工具 (scripts/verify)

Claude 开发完功能后，用本工具驱动 headless 浏览器验证功能 + 抓取浏览器报错/后端返回。

## 前提
- 前端 dev 运行中 (`localhost:9527`)，后端运行中 (`172.21.10.40:8888`)
- 账号通过环境变量注入，**绝不写入文件**：
  ```bash
  export DEVOPS_TEST_USER=User
  export DEVOPS_TEST_PASS='admin@123'
  ```

## 登录机制说明
dev 模式 `pnpm dev` = `vite --mode test`，前端**直连后端**（跨站），后端用 cookie 传递 JWT。
跨站 + HTTP 下浏览器无法自动管理该 cookie，故 `auth.py` 采用：**API 登录拿 token → route.fetch 注入 cookie → localStorage 注入 `isAuthenticated`**。
对使用者透明，调用 `auth.login(ctx, page, account)` 即得已登录 page。

## 1. 跑通用冒烟（每次基线 + 工具自测）
```bash
cd scripts/verify
DEVOPS_TEST_USER=User DEVOPS_TEST_PASS='admin@123' python3 smoke.py
```

## 2. 验证某个具体功能
复制模板并修改断言：
```bash
cp tasks/_example.py tasks/verify-my-feature.py
# 编辑 verify-my-feature.py 的 FEATURE_NAME / TARGET_URL / assert_feature
DEVOPS_TEST_USER=User DEVOPS_TEST_PASS='admin@123' python3 tasks/verify-my-feature.py
```

`assert_feature(page)` 里用 Playwright 选择器对本次功能做断言，返回 `report.CheckResult` 列表。
常用断言：`page.locator(sel).count()`、`page.locator(sel).is_visible()`、`page.get_by_text(...)`。

## 3. 跑单测
```bash
cd scripts/verify
python3 -m unittest discover -s tests -v
```

## 报告
- 终端打印结构化报告（基线冒烟 / 功能检查 / 错误清单）。
- 截图存 `reports/`（已 gitignore）。
- 退出码：0=通过，1=有错误。

## 报告里会捕获的错误
- `console.error` / `console.warning`
- JS 未捕获异常 (`pageerror`)
- 网络 4xx/5xx
- API 业务码 ≠ `0000`（后端返回异常）

## 模块
| 文件 | 职责 |
|------|------|
| `config.py` | 配置/账号(环境变量)/可达性探测 |
| `auth.py` | 登录（API+route注入方案） |
| `probe.py` | 错误捕获（console/pageerror/network/业务码） |
| `report.py` | 报告渲染 + 截图路径 |
| `smoke.py` | 通用冒烟编排 |
| `tasks/_example.py` | 功能定制模板 |
