# 企微扫码登录失败排查（60020 可信IP）+ sys_error 详情丢失修复 v1.0.5

日期：2026-09-04

## 现象与根因

生产企微扫码授权成功后提示「企业微信授权失败，请重试」。错误链路：
`WecomCallbackView → handleWecomLogin(sys_wecom_auth.go:269) → ProfileByCode → UserIDByCode → auth/getuserinfo`。

**根因（配置侧，非代码）**：`errcode=60020 not allow to access from your ip, from ip: 183.242.12.86`。
企微 2022-06 后新建自建应用强制「企业可信IP」：gettoken 不校验 IP（能换 token）、
getuserinfo 校验（被拒）→ 能授权（域名维度）拿不到 userid（IP 维度），症状即此。
生产出口 IP 已验证 `curl ip.sb` = 183.242.12.86。

**修复动作**：企微管理后台 → 应用 → 「企业可信IP」加 `183.242.12.86`。无需改代码/重启。
应用消息推送 message/send 受同一限制，加完一并打通。

## 顺带发现并修复：sys_error 详情恒缺失错误内容（v1.0.5）

排查时错误日志页面只看到调用栈、无错误详情，根因是提取脱钩：
`core/internal/zap_core.go` 提取 errStr 只匹配 `f.Key=="error"/"err"` + ErrorType，
而 `logger.Builder.Err()` 实际写 `zap.String("error_msg")`（fields.go:21 常量 FieldErrorMsg，
注释声明与 zap_core 共用常量但 zap_core 用了字面量）→ info 恒缺「| 错误: xxx」段。
修复：抽纯函数 `extractErrorDetail`，键名用 `logger.FieldErrorMsg` 常量对齐 + 7 场景表驱动单测。
**err 明文一直都在容器日志文件里**（zap JSON 的 error_msg 字段），sys_error 只是展示丢。

## 发版记录

v1.0.5-e8e0281 已发布生效（本机 172.21.10.41:8100 /opt/devops-admin-publish）：
build-release.sh 三件套 → 拷入发布目录 → manifest 填 changeLog 后 mv 原子生效。
生产走「关于 → 检查更新 → 在线升级」拉 136MB 增量包（仅自研三镜像，数据面不动）。
注意：本机 ssh 自己 host key 不通，publish.sh 需改用直拷；root 的 mv 是 -i 别名，
脚本内覆盖必须 `mv -f`（否则卡交互）。
