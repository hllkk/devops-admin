package system

import "time"

// WecomSyncResult 企业微信通讯录同步结果统计(对齐前端 Api.System.WecomSyncResult)。
// 由 WecomContactService.SyncStructure 填充,手动触发与定时任务共用。
type WecomSyncResult struct {
	DeptTotal    int `json:"deptTotal"`    // 企微拉取的部门总数
	DeptCreated  int `json:"deptCreated"`  // 新建部门数
	DeptUpdated  int `json:"deptUpdated"`  // 更新部门数
	DeptSkipped  int `json:"deptSkipped"`  // 跳过(无变化)部门数
	UserTotal    int `json:"userTotal"`    // 企微拉取的成员总数(去重后)
	UserCreated  int `json:"userCreated"`  // 新建用户数
	UserUpdated  int `json:"userUpdated"`  // 更新用户数
	UserRestored int `json:"userRestored"` // 复职恢复用户数(曾因离职被同步停用,本期回到企微返回集)
	UserDisabled int `json:"userDisabled"` // 离职停用用户数(本地有、企微无)
	UserSkipped  int `json:"userSkipped"`  // 跳过(无变化)用户数
	PostTotal    int `json:"postTotal"`    // 同步派生的岗位总数(去重后)
	PostCreated  int `json:"postCreated"`  // 新建岗位数
}

// WecomSyncStatus 异步同步状态(POST syncStructure 启动返回 + GET syncStatus 查询返回)。
// 手动同步走异步:全量首同步含逐用户新建(bcrypt+事务),耗时远超 HTTP 超时(前端默认 10s),
// goroutine 内执行,结果与进度经本结构查询。Started 仅 POST 有意义(true=本次成功启动新同步,
// false=已有同步进行中)。状态快照同时写 OPS_CACHE(Redis 优先),多实例部署下任一实例可查。
type WecomSyncStatus struct {
	Started    bool             `json:"started"`              // 本次请求是否成功启动新同步(POST)
	InProgress bool             `json:"inProgress"`           // 是否正在同步
	Result     *WecomSyncResult `json:"result,omitempty"`     // 最近一次同步结果(进行中=上次结果)
	Error      string           `json:"error,omitempty"`      // 最近一次同步错误(空=成功/未发生)
	FinishedAt *time.Time       `json:"finishedAt,omitempty"` // 最近一次完成时间
}
