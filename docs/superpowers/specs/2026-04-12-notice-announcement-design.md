# 通知公告功能完善设计文档

## 概述

完善运维管理系统的通知公告功能，实现前后端完整对接，包括新增、删除、编辑功能，支持发布范围控制、有效期管理、置顶功能以及详细的阅读记录追踪。

## 当前状态

### 前端（已实现）
- 列表页面 `frontend/src/views/manage/notice/index.vue`
- 搜索组件 `frontend/src/views/manage/notice/modules/notice-search.vue`
- 操作抽屉 `frontend/src/views/manage/notice/modules/notice-operate-drawer.vue`
- API 定义 `frontend/src/service/api/system/notice.ts`
- TypeScript 类型定义 `frontend/src/typings/api/system.api.d.ts`
- 使用 WangEditor 富文本编辑器
- 字典类型：`sys_notice_type`（通知/公告）、`sys_normal_disable`（正常/停用）

### 后端（缺失）
- 无任何通知公告相关代码
- 需新建：Model、API 控制器、Service、Router

## 核心需求

### 功能需求
1. **公告管理**：新增、编辑、删除、列表查询
2. **发布范围控制**：支持全员可见、按用户、按角色、按部门三种方式
3. **时效控制**：支持生效时间和失效时间
4. **置顶功能**：支持置顶标志和置顶截止时间
5. **阅读记录**：记录用户首次阅读时间、最后阅读时间、阅读次数
6. **权限控制**：停用公告仅管理员可见，普通用户不可见

### 字典定义（已存在）
- `sys_notice_type`：通知(1)、公告(2)
- `sys_normal_disable`：正常(0)、停用(1)

---

## 数据库设计

### 主表 `sys_notice`

| 字段 | 类型 | GORM 标签 | 说明 |
|------|------|----------|------|
| ID | int64 | primarykey | 主键（雪花算法） |
| TenantID | int64 | index | 租户ID |
| NoticeTitle | string | size:100;not null | 公告标题 |
| NoticeType | string | type:char(1);not null | 类型：1=通知, 2=公告 |
| NoticeContent | string | type:text | 公告内容（富文本HTML） |
| Status | string | type:char(1);default:0 | 状态：0=正常, 1=停用 |
| IsAll | string | type:char(1);default:0 | 全员可见：0=否, 1=是 |
| TopFlag | string | type:char(1);default:0 | 置顶标志：0=否, 1=是 |
| TopEndTime | *time.Time | 置顶截止时间 |
| EffectiveTime | *time.Time | 生效时间 |
| ExpireTime | *time.Time | 失效时间 |
| CreateBy | int64 | index | 创建者ID |
| CreateByName | string | size:100 | 创建者名称 |
| CreatedAt | time.Time | 创建时间 |
| UpdatedAt | time.Time | 更新时间 |
| DeletedAt | gorm.DeletedAt | index | 软删除时间 |

### 关联表 `sys_notice_user`

公告与用户关联

| 字段 | 类型 | GORM 标签 | 说明 |
|------|------|----------|------|
| NoticeID | int64 | index;not null | 公告ID |
| UserID | int64 | index;not null | 用户ID |

### 关联表 `sys_notice_role`

公告与角色关联

| 字段 | 类型 | GORM 标签 | 说明 |
|------|------|----------|------|
| NoticeID | int64 | index;not null | 公告ID |
| RoleID | int64 | index;not null | 角色ID |

### 关联表 `sys_notice_dept`

公告与部门关联

| 字段 | 类型 | GORM 标签 | 说明 |
|------|------|----------|------|
| NoticeID | int64 | index;not null | 公告ID |
| DeptID | int64 | index;not null | 部门ID |

### 阅读记录表 `sys_notice_read`

| 字段 | 类型 | GORM 标签 | 说明 |
|------|------|----------|------|
| ID | int64 | primarykey | 主键（雪花算法） |
| NoticeID | int64 | uniqueIndex:idx_notice_user | 公告ID |
| UserID | int64 | uniqueIndex:idx_notice_user | 用户ID |
| FirstReadTime | time.Time | 首次阅读时间 |
| LastReadTime | time.Time | 最后阅读时间 |
| ReadCount | int | default:1 | 阅读次数 |

---

## API 接口设计

### 管理端 API（管理员使用）

基础路径：`/system/notice`

| 接口 | 方法 | 说明 | 权限标识 |
|------|------|------|---------|
| `/list` | GET | 分页查询公告列表 | system:notice:list |
| `/:id` | GET | 获取公告详情（编辑回显） | system:notice:query |
| `/` | POST | 新增公告 | system:notice:add |
| `/` | PUT | 更新公告 | system:notice:edit |
| `/:ids` | DELETE | 批量删除公告 | system:notice:remove |
| `/read-stats/:id` | GET | 获取阅读统计 | system:notice:query |

### 用户端 API（普通用户使用）

基础路径：`/system/notice/my`

| 接口 | 方法 | 说明 |
|------|------|------|
| `/list` | GET | 当前用户可见公告列表 |
| `/:id` | GET | 公告详情（自动记录阅读） |
| `/unread-count` | GET | 未读公告数量 |

### 请求参数结构

**查询列表参数**：
```go
type NoticeSearchParams struct {
    PageNum     int    `form:"pageNum"`
    PageSize    int    `form:"pageSize"`
    NoticeTitle string `form:"noticeTitle"`
    NoticeType  string `form:"noticeType"`
    Status      string `form:"status"`
}
```

**新增/编辑请求参数**：
```go
type NoticeOperateRequest struct {
    NoticeId      *int64     `json:"noticeId"`      // 编辑时必填
    NoticeTitle   string     `json:"noticeTitle" binding:"required"`
    NoticeType    string     `json:"noticeType" binding:"required"`
    NoticeContent string     `json:"noticeContent" binding:"required"`
    Status        string     `json:"status" binding:"required"`
    IsAll         string     `json:"isAll"`         // 默认 "0"
    UserIds       []int64    `json:"userIds"`
    RoleIds       []int64    `json:"roleIds"`
    DeptIds       []int64    `json:"deptIds"`
    TopFlag       string     `json:"topFlag"`       // 默认 "0"
    TopEndTime    *time.Time `json:"topEndTime"`
    EffectiveTime *time.Time `json:"effectiveTime"`
    ExpireTime    *time.Time `json:"expireTime"`
}
```

### 响应结构

**公告详情响应**：
```go
type NoticeDetailResponse struct {
    NoticeId      int64     `json:"noticeId"`
    NoticeTitle   string    `json:"noticeTitle"`
    NoticeType    string    `json:"noticeType"`
    NoticeContent string    `json:"noticeContent"`
    Status        string    `json:"status"`
    IsAll         string    `json:"isAll"`
    TopFlag       string    `json:"topFlag"`
    TopEndTime    *string   `json:"topEndTime"`
    EffectiveTime *string   `json:"effectiveTime"`
    ExpireTime    *string   `json:"expireTime"`
    CreateByName  string    `json:"createByName"`
    CreatedAt     string    `json:"createTime"`
    UserIds       []int64   `json:"userIds"`
    RoleIds       []int64   `json:"roleIds"`
    DeptIds       []int64   `json:"deptIds"`
}
```

**阅读统计响应**：
```go
type NoticeReadStatsResponse struct {
    TotalReaders   int               `json:"totalReaders"`
    TotalUnreaders int               `json:"totalUnreaders"`
    ReadRecords    []ReadRecordItem  `json:"readRecords"`
}

type ReadRecordItem struct {
    UserId        int64  `json:"userId"`
    UserName      string `json:"userName"`
    NickName      string `json:"nickName"`
    FirstReadTime string `json:"firstReadTime"`
    LastReadTime  string `json:"lastReadTime"`
    ReadCount     int    `json:"readCount"`
}
```

---

## 前端改动设计

### 修改文件清单

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| `src/typings/api/system.api.d.ts` | 扩展 | 添加新字段和类型定义 |
| `src/service/api/system/notice.ts` | 扩展 | 添加新接口函数 |
| `src/views/manage/notice/modules/notice-operate-drawer.vue` | 重写 | 添加发布范围、时效、置顶表单项 |
| `src/views/manage/notice/index.vue` | 扩展 | 添加阅读统计查看入口 |

### TypeScript 类型扩展

```typescript
// 扩展 Notice 类型
type Notice = Common.CommonRecord<{
  noticeId: CommonType.IdType;
  tenantId: CommonType.IdType;
  noticeTitle: string;
  noticeType: System.NoticeType;
  noticeContent: string;
  status: Common.EnableStatus;
  isAll: Common.YesOrNoStatus;           // 全员可见
  topFlag: Common.YesOrNoStatus;         // 置顶标志
  topEndTime?: string;                   // 置顶截止时间
  effectiveTime?: string;                // 生效时间
  expireTime?: string;                   // 失效时间
  createByName: string;
  remark?: string;
  userIds?: CommonType.IdType[];
  roleIds?: CommonType.IdType[];
  deptIds?: CommonType.IdType[];
}>;

// 阅读记录类型
type NoticeReadRecord = {
  userId: CommonType.IdType;
  userName: string;
  nickName: string;
  firstReadTime: string;
  lastReadTime: string;
  readCount: number;
};

// 阅读统计响应
type NoticeReadStatsResponse = {
  totalReaders: number;
  totalUnreaders: number;
  readRecords: NoticeReadRecord[];
};

// 扩展操作参数
type NoticeOperateParams = CommonType.RecordNullable<
  Pick<Api.System.Notice,
    'noticeId' | 'noticeTitle' | 'noticeType' | 'noticeContent' | 'status'
    | 'isAll' | 'userIds' | 'roleIds' | 'deptIds'
    | 'topFlag' | 'topEndTime' | 'effectiveTime' | 'expireTime'
  >
>;
```

### API 函数扩展

```typescript
// 新增接口
export function fetchGetNoticeDetail(id: CommonType.IdType);
export function fetchGetNoticeReadStats(id: CommonType.IdType);
export function fetchGetMyNoticeList(params?: Api.System.NoticeSearchParams);
export function fetchGetMyNoticeDetail(id: CommonType.IdType);
export function fetchGetUnreadCount();
```

### 操作抽屉新增表单项

- **全员可见开关**：`NSwitch` 组件
- **发布范围**：当全员可见关闭时显示
  - 用户选择器：`UserSelect` 组件（多选）
  - 角色选择器：`RoleSelect` 组件（多选）
  - 部门选择器：`DeptTreeSelect` 组件（多选）
- **置顶开关**：`NSwitch` 组件
- **置顶截止时间**：`NDatePicker` 组件（置顶开启时显示）
- **生效时间**：`NDatePicker` 组件
- **失效时间**：`NDatePicker` 组件

---

## 后端代码架构

### 目录结构

```
backend/
├── model/system/
│   ├── sys_notice.go              # 公告主表模型
│   ├── sys_notice_scope.go       # 关联表模型（user/role/dept）
│   ├── sys_notice_read.go        # 阅读记录模型
│   ├── request/
│   │   └── sys_notice.go         # 请求参数结构体
│   └── response/
│       └── sys_notice.go         # 响应结构体
├── api/v1/system/
│   ├── sys_notice.go             # 公告API控制器
│   └── enter.go                  # 注册 NoticeApi
├── service/system/
│   ├── sys_notice.go             # 公告Service业务逻辑
│   └── enter.go                  # 注册 NoticeService
├── router/system/
│   ├── sys_notice.go             # 公告路由注册
│   └── enter.go                  # 注册 NoticeRouter
├── source/system/
│   └── notice.go                 # 数据库迁移（AutoMigrate）
```

### 权限标识

| 权限标识 | 说明 |
|---------|------|
| `system:notice:list` | 查询公告列表 |
| `system:notice:add` | 新增公告 |
| `system:notice:edit` | 编辑公告 |
| `system:notice:remove` | 删除公告 |
| `system:notice:query` | 查询公告详情和阅读统计 |

---

## Service 层核心逻辑

### 权限过滤逻辑（用户端获取公告列表）

```go
// 用户可见公告查询条件
func GetUserNoticeQueryConditions(userId int64, userRoleIds []int64, userDeptIds []int64) {
    // 1. 状态必须为"正常"(status='0')
    // 2. 当前时间在有效期范围内
    //    - effective_time IS NULL OR effective_time <= now
    //    - expire_time IS NULL OR expire_time >= now
    // 3. 权限范围过滤：
    //    - isAll='1' -> 全员可见
    //    - isAll='0' -> 检查关联表：
    //      - sys_notice_user 中有 userId 记录
    //      - sys_notice_role 中有 roleId 在 userRoleIds 中的记录
    //      - sys_notice_dept 中有 deptId 在 userDeptIds（含祖先部门）中的记录
    // 4. 排序：置顶优先（topFlag='1' AND topEndTime >= now），然后按创建时间降序
}
```

### 部门层级查询逻辑

```go
// 获取用户部门及其所有祖先部门ID
func GetUserDeptWithAncestors(deptId int64) []int64 {
    // 通过 ancestors 字段解析祖先部门ID链
    // 例如 ancestors = "0,100,101" -> [0, 100, 101, 当前deptId]
}
```

### 阅读记录逻辑

```go
// 记录用户阅读
func RecordUserRead(noticeId int64, userId int64) {
    // 1. 查询是否存在阅读记录
    // 2. 不存在：
    //    - 创建新记录
    //    - firstReadTime = now
    //    - readCount = 1
    // 3. 存在：
    //    - 更新 lastReadTime = now
    //    - readCount++
}
```

### 未读数量计算

```go
// 计算用户未读公告数量
func GetUnreadCount(userId int64, visibleNoticeIds []int64) int {
    // 1. 获取用户可见公告ID列表
    // 2. 查询用户已读公告ID列表
    // 3. 未读数量 = 可见数量 - 已读数量
}
```

---

## 实现要点

### 性能优化

1. **批量查询优化**：使用 `IN` 查询一次性获取关联数据
2. **索引设计**：所有关联表和阅读记录表的关键字段建立索引
3. **缓存策略**：字典数据缓存（已实现），用户权限范围可考虑缓存

### 代码规范遵循

- 后端遵循 `backend/.claude/CLAUDE.md` 规范
- 前端遵循 `frontend/.claude/CLAUDE.md` 规范
- API 响应使用统一格式 `response.OkWithData()` / `response.FailWithMessage()`
- 日志使用 `global.OPS_LOG` + `zap`

### 测试要点

1. 前端修改后执行 `pnpm typecheck` 和 `pnpm lint`
2. 后端编译验证 `go build`
3. 功能测试：
   - 新增公告并设置不同发布范围
   - 编辑公告修改范围和时效
   - 删除公告
   - 不同用户查看公告列表权限验证
   - 阅读记录追踪验证
   - 置顶和时效自动失效验证