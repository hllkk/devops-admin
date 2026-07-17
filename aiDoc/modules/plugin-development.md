# 插件开发约束

## 后端插件结构

后端插件推荐保持以下结构（位于 `server/plugin/<name>/`）：

- `api/`
- `config/`
- `initialize/`
- `model/`
- `model/request/`
- `router/`
- `service/`
- `plugin.go`

## 前端插件结构

前端插件推荐保持以下结构（位于 `web/src/plugins/<name>/`）：

- `api/`
- `components/`
- `view/`
- `form/`
- `config.ts` 或等价入口文件

## 插件入口约束

`plugin.go` 至少要承担以下职责：

- 实现项目要求的插件接口（`interfaces.Plugin`，`utils/plugin/v2`）
- 在 `init()` 中完成插件注册（`interfaces.Register(Plugin)`）
- 通过 `Register` 方法挂载路由（调 `initialize.Router`）
- 通过 `RouterPath` 返回插件根路径

并在 `server/plugin/register.go` 用 `_ "github.com/hllkk/devops-admin/server/plugin/<name>"` 匿名引用激活。

入口写法见 `aiDoc/examples/backend/plugin-go-example.md`。

## 插件设计原则

- 尽量自包含
- 保持可配置
- 预留扩展点
- 与主系统保持一致的风格与约定

## 推荐开发流程

1. 先明确插件边界与数据模型
2. 先完成后端模型、服务、接口与初始化
3. 再完成前端接口封装、页面与表单
4. 最后完成菜单、权限、联调与测试
