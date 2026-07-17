# 后端分层约束

> 适用范围：`server/`。写任何后端代码前先读本文件。技术基座 = gin-vue-admin 同款分层范式，module = `github.com/hllkk/devops-admin/server`（简称 `devops-admin`）。

## 总原则

- 严格遵守 `Router -> API -> Service -> Model` 依赖方向
- 禁止跨层直接调用（API 不直接操作数据库；Service 不依赖 HTTP 语义）
- `enter.go` 作为组装与暴露入口（`ServiceGroupApp` / `ApiGroupApp` / `RouterGroupApp`），避免循环引用

## Model 层

- 全局基座位于 `server/global/model.go`，分三个（最底层 `OPS_BASE` 不含主键）：
  - `global.OPS_BASE`：生命周期基座（`CreateTime/UpdateTime/DeletedAt`），所有表通用，**不含主键**
  - `global.OPS_MODEL`：过渡基座 = `OPS_BASE` + `ID`（`uint`，`json:"id,string"`，自带主键），供尚未改造为业务命名主键的内部系统表（日志/黑名单/单行配置等）直接复用
  - `global.OPS_AUDIT_MODEL`：审计基座 = `OPS_BASE` + `CreateBy/UpdateBy`（`json:"createBy,string"` / `"updateBy,string"`），用于对齐 RuoYi/前端 `CommonRecord` 的对外业务实体（`SysUser`…）
- 内部系统记录（日志、黑名单等 append-only 表）用 `OPS_MODEL`；对外业务实体用 `OPS_AUDIT_MODEL`，并自定义业务命名主键
- 主键策略：对外业务实体自定义业务命名主键（`userId`/`roleId`，`json:"userId,string"`），内部表用 `id`。**目标**统一雪花 `int64` + `autoIncrement:false` + 回调 `ops:snowflake_id`（按 `PrioritizedPrimaryField` 自动填充，仅在主键为 0 时不覆盖显式值）；该回调与 `utils/snowflake` **当前待落地**，现阶段主键走 DB 自增，新建模型须显式声明主键列
- 字段应补全清晰的 `json` 与 `gorm` 标签
- 请求模型放在 `model/request/`
- 列表查询模型应定义 `XxxSearch`，并内嵌通用的 `request.PageInfo`

### 关联建模：多对多用显式关联表，不用 GORM `many2many`

多对多关系一律建**独立的关联表 struct**（复合主键 + 显式 `TableName()`），并在 `RegisterTables()` 注册；**不挂** `gorm:"many2many:..."` 自动关联。参考 `SysUserRole` / `SysRoleDepartment` / `SysUserDepartment`。

理由：

- **主键策略冲突**：项目用雪花 `int64` 主键 + "回调只在主键为 0 时填充、不覆盖显式值"的写入约定。`many2many` 的 `Append/Replace` 由 GORM 接管 join 行写入，会和"插入必须显式传两个 ID"打架。
- **契约精确对齐**：RuoYi/前端要求固定表名、列名、复合主键。GORM 自动 join 表按其内部约定命名，纠正它要堆 `JoinTable`/`JoinForeignKey` tag，不如直接写 struct + `column:` 干净。
- **授权场景是批量替换**：给角色分配菜单 = `DELETE FROM sys_role_menu WHERE role_id=?` + 批量 INSERT。显式表让"删后插"直白、好对 SQL 日志、好调优；`many2many` 的 `Replace` 是黑盒。
- **鉴权要定向查询**：权限校验只想 `SELECT role_key ... JOIN sys_user_role`，不需要 `Preload("Roles").First(&user)` 预加载整棵对象图（易 N+1、拉无用字段）。显式表鼓励精确 SQL。
- **关联行是一等公民**：关联表无软删除/无审计字段，但同样进 `AutoMigrate`、同样写迁移测试，可独立验证，而非 GORM 隐式生成的二等表。

读写约定：

- **写入侧**：批量删插走显式 struct（如 `SysUserRole`）。
- **读取侧**：默认也走显式 join 查询。若某场景确需对象图导航，可在模型上挂只读 `gorm:"many2many:..."` 字段（复用同一张物理表），但写入侧不变。

## 类型一致性

- 同一字段在模型、请求结构、响应结构、前端使用处必须保持一致
- 状态字段、ID 字段、枚举字段、时间字段是高风险字段，必须重点检查
- 若涉及指针类型与非指针类型互转，必须在 Service 层显式处理 `nil`
- 详见前后端契约 `aiDoc/frontend-backend/boundary.md`

## Service 层

- 只承载业务逻辑，不处理 HTTP 语义
- 不要依赖 `gin.Context`
- 函数应返回业务结果和 `error`
- 每个模块在 `service/` 下建立独立文件，并在 `service/enter.go` 注册

## API 层

- 负责参数提取、参数校验、调用 Service 和统一响应
- **参数从哪里取，取决于前端怎么传、协议怎么设计、当前逻辑需要什么，以及哪个位置更合理**
- **不要把绑定方式写死成某一种固定模板**

### 常见参数来源
JSON body / Query string / Path params / `multipart/form-data` / Header / Cookie

### 常见取法
- JSON body: `ShouldBindJSON`
- Query: `ShouldBindQuery`、`c.Query(...)`、`c.DefaultQuery(...)`
- Path: `c.Param(...)`
- form-data / file upload: `c.FormFile(...)`、`c.DefaultPostForm(...)`、`c.Request.FormValue(...)`
- Header: `c.GetHeader(...)`、`c.Request.Header.Get(...)`
- Cookie: `c.Cookie(...)`

### 使用原则
- 绑定方式要与真实参数来源一致
- 不要为了套模板，把 Header / Cookie / Query / form-data 中的数据强行改成 body
- 认证、追踪、网关透传等信息，很多时候本来就应该从 Header 或 Cookie 获取
- 上传文件时，应按上传协议从 `multipart/form-data` 中取文件和附带字段（网盘模块分片上传尤其注意）
- 必须通过 `service.ServiceGroupApp` 访问服务层
- 必须使用项目统一的 `response` 包输出结果
- 每个对外 API 都必须写完整且准确的 Swagger 注释

## Router 层

- 负责路由分组、中间件挂载和处理函数绑定
- 必须通过 `api.ApiGroupApp` 引用 API 层
- 每个模块在 `router/` 下建立独立文件，并在 `router/enter.go` 注册

## Initialize 层

插件或模块若需要初始化入口，至少关注以下职责：

- `gorm.go`: 表结构迁移（`AutoMigrate`）
- `router.go`: 路由注册
- `menu.go`: 菜单与权限初始化
- `viper.go`: 配置加载
- `api.go`: API 注册

## Swagger 约束

对外 API 的 Swagger 注释至少要准确说明：

- 功能说明
- 请求参数
- 响应结构
- 路由路径
- 鉴权要求

## 参考
- 目录与结构关系：`aiDoc/relations/system-map.md`
- 开发流程：`aiDoc/relations/development-workflow.md`
- 业务模块：`aiDoc/modules/business-modules.md`
- 前后端契约：`aiDoc/frontend-backend/boundary.md`
