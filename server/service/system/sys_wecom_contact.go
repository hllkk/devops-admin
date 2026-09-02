package system

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/utils"
	"github.com/hllkk/devops-admin/server/utils/logger"
)

// WecomContactService 企业微信通讯录同步服务(部门/用户/岗位,单向拉取:企微 → 本地,不回写企微)。
//
// 蓝本为 SoyDisk 的 sys_wecom_contact.go(其本身是对旧 devops-admin 实现的修复版),本次移植并做四项修复:
//   - 复职恢复:离职差集停用时置 sys_social.disabled_by_sync,复职(同一 userid 回到企微返回集)仅恢复
//     "同步停用"的用户并清标——管理员手动停用的在职员工不被同步反复启用(对齐 AiKey.DisabledByCascade 模式);
//   - 同步状态快照落 OPS_CACHE:多实例部署下任一实例的 syncStatus 都能读到最近状态(内存状态机仅本实例精确);
//   - fail-fast 错误带定位:建号/更新失败的用户与部门错误均 wrap 上下文标识;
//   - 绑定孤儿清理:本地用户已硬删但 sys_social 残留的绑定,同步时识别并删除,随后按需重建。
//
// 沿用 SoyDisk 的核心设计(规避旧 devops-admin 的性能/安全问题):
//   - 批量 load 内存 map 比对,替代逐行 FirstOrCreate 的 N+1;
//   - 每用户/每部门小事务或单 SQL,替代"全量塞一个事务"的大事务持锁;
//   - 离职停用仅作用于"同步在册"绑定(in_sync_scope=true),扫码登录建号不受影响;
//   - 建号用随机不可用口令,非 123456。
//
// 复用现有:utils.WecomClient(access_token 缓存+singleflight+失效重试)、sys_social(wecom 绑定)、
// sys_user_departments(多部门)、SysGeneralConfig.DefaultRoleId(默认角色)。
type WecomContactService struct{}

const (
	wecomSyncLockKey   = "wecom:sync_lock"   // 分布式锁 key(防同步重入)
	wecomSyncLockTTL   = 5 * time.Minute     // 锁 TTL,略大于千级成员同步预期耗时
	wecomFetchLimit    = 5                   // 并发拉成员限速(规避企微频率限制)
	wecomSyncStateKey  = "wecom:sync:status" // OPS_CACHE 同步状态快照 key(多实例可见)
	wecomSyncStateTTL  = time.Hour           // 快照 TTL:完成后保留 1h 供轮询/面板查看
	wecomStatusEnable  = "0"                 // sys_users.status 启用(对齐项目字面量惯例)
	wecomStatusDisable = "1"                 // sys_users.status 停用
)

// wecomSyncMu Redis 不可用时的进程内降级锁(单实例语义)
var wecomSyncMu sync.Mutex

// wecomSyncState 异步同步状态(本实例精确;跨实例经 OPS_CACHE 快照补充,见 snapshot)。
// 供 POST 启动返回 + GET 状态查询。
type wecomSyncState struct {
	mu         sync.RWMutex
	running    bool
	result     system.WecomSyncResult
	err        string
	finishedAt time.Time
	hasResult  bool
}

var wecomSync = &wecomSyncState{}

// startIfIdle 若当前空闲则置 running=true 并返回 true(本次启动);已在同步则返回 false。
func (st *wecomSyncState) startIfIdle() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.running {
		return false
	}
	st.running = true
	return true
}

// cancelStart 撤销 startIfIdle 的占用(试锁失败反悔):仅复位 running,不覆盖已有结果。
func (st *wecomSyncState) cancelStart() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.running = false
}

// finish 同步结束:置 running=false,记录结果/错误/完成时间,并写 OPS_CACHE 快照(跨实例可见)。
func (st *wecomSyncState) finish(r system.WecomSyncResult, e error) {
	st.mu.Lock()
	st.running = false
	st.result = r
	if e != nil {
		st.err = e.Error()
	} else {
		st.err = ""
	}
	st.finishedAt = time.Now()
	st.hasResult = true
	snap := st.snapshotLocked()
	st.mu.Unlock()
	persistWecomSyncStatus(snap)
}

// snapshotLocked 读锁已持有时组装快照(不取锁,finish 内复用)。
func (st *wecomSyncState) snapshotLocked() system.WecomSyncStatus {
	status := system.WecomSyncStatus{InProgress: st.running, Error: st.err}
	if st.hasResult {
		r := st.result
		status.Result = &r
		t := st.finishedAt
		status.FinishedAt = &t
	}
	return status
}

// snapshot 返回当前状态快照(供 GET syncStatus):本实例 running 时内存最准;
// idle 时优先读 OPS_CACHE 快照——多实例下它可能比本实例内存新(他实例刚完成一轮)。
func (st *wecomSyncState) snapshot() system.WecomSyncStatus {
	st.mu.RLock()
	running := st.running
	local := st.snapshotLocked()
	st.mu.RUnlock()
	if running {
		return local
	}
	if snap, ok := loadPersistedWecomSyncStatus(); ok {
		return snap
	}
	return local
}

// persistWecomSyncStatus 写状态快照到 OPS_CACHE。
// Redis 后端会做 JSON 往返(类型不保真),故自己 Marshal 成 string 存。
func persistWecomSyncStatus(status system.WecomSyncStatus) {
	b, err := json.Marshal(status)
	if err != nil {
		return
	}
	global.OPS_CACHE.Set(wecomSyncStateKey, string(b), wecomSyncStateTTL)
}

// loadPersistedWecomSyncStatus 读 OPS_CACHE 状态快照(无或损坏返回 false)。
func loadPersistedWecomSyncStatus() (system.WecomSyncStatus, bool) {
	v, ok := global.OPS_CACHE.Get(wecomSyncStateKey)
	if !ok {
		return system.WecomSyncStatus{}, false
	}
	s, ok := v.(string)
	if !ok {
		return system.WecomSyncStatus{}, false
	}
	var status system.WecomSyncStatus
	if err := json.Unmarshal([]byte(s), &status); err != nil {
		return system.WecomSyncStatus{}, false
	}
	return status, true
}

// acquireSyncLock 获取同步锁:Redis SetNX 优先(多实例),不可用降级进程内 TryLock。
// 返回 release 闭包(调用方 defer);未获取返回 ok=false。
func acquireSyncLock(ctx context.Context) (release func(), ok bool) {
	if global.OPS_REDIS != nil {
		got, err := global.OPS_REDIS.SetNX(ctx, wecomSyncLockKey, "1", wecomSyncLockTTL).Result()
		if err == nil {
			if !got {
				return nil, false
			}
			return func() { _ = global.OPS_REDIS.Del(ctx, wecomSyncLockKey).Err() }, true
		}
		// Redis 出错降级进程锁
		logger.Bg().Mod("wecom").Err(err).Warn("同步锁 Redis 异常,降级进程内锁")
	}
	if !wecomSyncMu.TryLock() {
		return nil, false
	}
	return wecomSyncMu.Unlock, true
}

// StartSync 异步启动通讯录同步(手动触发入口):先试同步锁(拿不到=已有同步进行中,直接返回不启动),
// 拿到后置 running、起 goroutine 执行,完成后记结果/错误到 wecomSync 并写 OPS_CACHE 快照。
// 全量首同步含逐用户新建(bcrypt+事务),耗时远超 HTTP 超时(前端默认 10s),故脱离请求生命周期异步执行;
// 结果经 SyncStatus 轮询。
func (s *WecomContactService) StartSync() system.WecomSyncStatus {
	if !wecomSync.startIfIdle() {
		return system.WecomSyncStatus{Started: false, InProgress: true}
	}
	release, ok := acquireSyncLock(context.Background())
	if !ok {
		// 他实例正在同步:撤销本实例占用,不启动 goroutine(也就不会用锁冲突错误覆盖状态)
		wecomSync.cancelStart()
		return system.WecomSyncStatus{Started: false, InProgress: true}
	}
	persistWecomSyncStatus(system.WecomSyncStatus{InProgress: true})
	go func() {
		// goroutine 脱离请求生命周期,全局 GinRecovery 兜不到;此处必须 recover,
		// 否则 panic 既崩溃进程、又使 running 标志永不复位(后续同步全被 startIfIdle 拒)。
		defer func() {
			if r := recover(); r != nil {
				wecomSync.finish(system.WecomSyncResult{}, fmt.Errorf("同步异常 panic: %v", r))
				logger.Bg().Mod("wecom").Error(fmt.Sprintf("通讯录同步 panic: %v", r))
			}
		}()
		defer release()
		result, err := s.syncStructureLocked(context.Background())
		wecomSync.finish(result, err)
		if err != nil {
			logger.Bg().Mod("wecom").Err(err).Error("通讯录同步失败")
		} else {
			logger.Bg().Mod("wecom").Info("通讯录同步完成")
		}
	}()
	return system.WecomSyncStatus{Started: true, InProgress: true}
}

// SyncStatus 查询异步同步状态(进度/最近结果/错误)。
func (s *WecomContactService) SyncStatus() system.WecomSyncStatus {
	return wecomSync.snapshot()
}

// SyncStructure 通讯录同步(定时任务入口):取锁 → 同步执行全流程。
// 手动触发走 StartSync(异步);两者共用 acquireSyncLock 互斥。
func (s *WecomContactService) SyncStructure(ctx context.Context) (system.WecomSyncResult, error) {
	release, ok := acquireSyncLock(ctx)
	if !ok {
		return system.WecomSyncResult{}, errors.New("通讯录同步正在进行中,请稍后再试")
	}
	defer release()
	result, err := s.syncStructureLocked(ctx)
	if err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Error("通讯录同步失败")
	} else {
		logger.WithCtx(ctx).Mod("wecom").Info("通讯录同步完成")
	}
	// 定时通道也写快照,让前端轮询/其他实例能看到最近一次结果
	wecomSync.finish(result, err)
	return result, err
}

// syncStructureLocked 主流程(调用方已持同步锁):拉全部门 → 同步部门 → 并发拉全员 →
// 同步用户(含复职恢复/离职停用/孤儿清理)→ 同步岗位。
func (s *WecomContactService) syncStructureLocked(ctx context.Context) (system.WecomSyncResult, error) {
	var result system.WecomSyncResult

	cfg := (&AuthConfigService{}).Current(ctx)
	client := wecomClientFromCfg(cfg)
	if !client.Configured() {
		return result, errors.New("企业微信配置不完整(CorpId/Secret/AgentId/回调地址)")
	}
	// 建号默认角色前置校验(对齐扫码登录 loginOrRegister:已配置+存在+未停用)
	if _, err := wecomDefaultRoleId(ctx); err != nil {
		return result, err
	}

	// 拉全部门
	wecomDepts, err := client.DepartmentList(ctx)
	if err != nil {
		return result, err
	}
	result.DeptTotal = len(wecomDepts)

	// 同步部门,拿到 wecomDeptId→本地 deptId 映射供用户同步
	deptIdByWecom, err := s.syncDepartments(ctx, wecomDepts, &result)
	if err != nil {
		return result, err
	}

	// 并发拉全员(按 userid 去重)
	wecomUsers, err := fetchAllWecomUsers(ctx, client, wecomDepts)
	if err != nil {
		return result, err
	}
	result.UserTotal = len(wecomUsers)

	// 同步用户(含复职恢复/离职停用/孤儿清理)
	if err = s.syncUsers(ctx, wecomUsers, deptIdByWecom, &result); err != nil {
		return result, err
	}
	// 同步岗位(企微 position 派生,归属主部门)+ 用户岗位关联
	if err = s.syncPosts(ctx, wecomUsers, deptIdByWecom, &result); err != nil {
		return result, err
	}
	return result, nil
}

// wecomDefaultRoleId 取并校验建号默认角色(常规配置 SysGeneralConfig.DefaultRoleId,
// 供企微扫码登录/通讯录同步复用):已配置、角色存在且未停用,否则返回错误。
func wecomDefaultRoleId(ctx context.Context) (int64, error) {
	defaultRoleId := (&GeneralConfigService{}).Current(ctx).DefaultRoleId
	if defaultRoleId == 0 {
		return 0, errors.New("默认角色未配置,请在「系统设置 → 常规配置」中配置默认角色")
	}
	var role system.SysRole
	global.OPS_DB.WithContext(ctx).Where("role_id = ?", defaultRoleId).Limit(1).Find(&role)
	if role.RoleId == 0 {
		return 0, errors.New("企业微信默认角色不存在,请联系管理员")
	}
	if role.Status != wecomStatusEnable {
		return 0, errors.New("企业微信默认角色已停用,请联系管理员")
	}
	return defaultRoleId, nil
}

// syncDepartments 增量同步部门:一次性 load 本地(wecom_dept_id>0)到 map,
// 阶段1 为缺失部门建本地记录(ancestors 占位"0"),阶段2 在所有 localId 就绪后
// 递归重算 ancestors 并比对更新。部门只增/改,不删(企微侧删除的部门本地保留,避免级联破坏用户/数据权限)。
func (s *WecomContactService) syncDepartments(ctx context.Context, wecomDepts []utils.WecomDepartment, result *system.WecomSyncResult) (map[int64]int64, error) {
	byId := make(map[int64]utils.WecomDepartment, len(wecomDepts))
	for _, d := range wecomDepts {
		byId[d.Id] = d
	}

	var localDepts []system.SysDepartment
	if err := global.OPS_DB.WithContext(ctx).Where("wecom_dept_id > 0").Find(&localDepts).Error; err != nil {
		return nil, err
	}
	localByWecom := make(map[int64]*system.SysDepartment, len(localDepts))
	deptIdByWecom := make(map[int64]int64, len(wecomDepts))
	for i := range localDepts {
		localByWecom[localDepts[i].WecomDeptId] = &localDepts[i]
		deptIdByWecom[localDepts[i].WecomDeptId] = localDepts[i].DeptId
	}

	// 阶段1:缺失部门先建本地记录(雪花回调自动生成 dept_id,ancestors 占位)
	for _, wd := range wecomDepts {
		if _, exists := localByWecom[wd.Id]; exists {
			continue
		}
		d := system.SysDepartment{
			DeptName:    wd.Name,
			OrderNum:    int(wd.Order),
			Status:      wecomStatusEnable,
			WecomDeptId: wd.Id,
		}
		if err := global.OPS_DB.WithContext(ctx).Create(&d).Error; err != nil {
			return nil, fmt.Errorf("创建部门[%s]失败: %w", wd.Name, err)
		}
		deptIdByWecom[wd.Id] = d.DeptId
		result.DeptCreated++
	}

	// 阶段2:所有 localId 就绪,递归算 ancestors,比对更新(新建部门修正占位)。
	// 递归带 memo,不依赖企微返回顺序,规避"按 parentid 排序非拓扑序"的树结构错乱陷阱。
	ancestorsMemo := make(map[int64]string)
	var calcAncestors func(wecomId int64) string
	calcAncestors = func(wecomId int64) string {
		if a, ok := ancestorsMemo[wecomId]; ok {
			return a
		}
		wd := byId[wecomId]
		if wd.Id == 0 || wd.ParentId <= 0 { // 根部门
			ancestorsMemo[wecomId] = "0"
			return "0"
		}
		parentLocal, hasParent := deptIdByWecom[wd.ParentId]
		if !hasParent { // 父不在同步范围(如被过滤),归顶层
			ancestorsMemo[wecomId] = "0"
			return "0"
		}
		a := calcAncestors(wd.ParentId) + "," + strconv.FormatInt(parentLocal, 10)
		ancestorsMemo[wecomId] = a
		return a
	}
	for _, wd := range wecomDepts {
		newParent := int64(0)
		if wd.ParentId > 0 {
			newParent = deptIdByWecom[wd.ParentId] // 父已就绪则取本地 id,否则 0
		}
		newAncestors := calcAncestors(wd.Id)
		exist := localByWecom[wd.Id]
		if exist != nil {
			if exist.DeptName == wd.Name && exist.ParentId == newParent &&
				exist.OrderNum == int(wd.Order) && exist.Ancestors == newAncestors {
				result.DeptSkipped++
				continue
			}
			if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{}).Where("dept_id = ?", exist.DeptId).
				Updates(map[string]interface{}{
					"dept_name": wd.Name,
					"parent_id": newParent,
					"order_num": int(wd.Order),
					"ancestors": newAncestors,
				}).Error; err != nil {
				return nil, fmt.Errorf("更新部门[%s]失败: %w", wd.Name, err)
			}
			result.DeptUpdated++
		} else {
			// 本轮新建:修正阶段1 的占位 parent/ancestors(根部门 newAncestors=="0"&&newParent==0 跳过)
			if newAncestors != "0" || newParent != 0 {
				if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDepartment{}).Where("dept_id = ?", deptIdByWecom[wd.Id]).
					Updates(map[string]interface{}{"parent_id": newParent, "ancestors": newAncestors}).Error; err != nil {
					return nil, fmt.Errorf("修正部门[%s]层级失败: %w", wd.Name, err)
				}
			}
		}
	}
	return deptIdByWecom, nil
}

// fetchAllWecomUsers 并发拉取全员(每部门一 goroutine + 令牌限速),按 userid 去重聚合。
// 任一部门拉取失败则整体失败(fail-fast,错误带部门 id 定位,让调用方重试)。
func fetchAllWecomUsers(ctx context.Context, client *utils.WecomClient, wecomDepts []utils.WecomDepartment) (map[string]utils.WecomContactUser, error) {
	var mu sync.Mutex
	result := make(map[string]utils.WecomContactUser)
	sem := make(chan struct{}, wecomFetchLimit)
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	for _, d := range wecomDepts {
		wg.Add(1)
		go func(deptId int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			users, err := client.DepartmentUsers(ctx, deptId, false)
			if err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("拉取部门[%d]成员失败: %w", deptId, err) })
				return
			}
			mu.Lock()
			for _, u := range users {
				if u.UserID != "" {
					if _, exists := result[u.UserID]; !exists {
						result[u.UserID] = u
					}
				}
			}
			mu.Unlock()
		}(d.Id)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return result, nil
}

// syncUsers 同步用户:批量 load 本地 sys_social(wecom)+对应 SysUser+多部门归属 到 map,
// 先清绑定孤儿,再逐个比对 upsert(含复职恢复),末尾按差集停用离职(仅同步在册绑定,见 in_sync_scope)。
func (s *WecomContactService) syncUsers(ctx context.Context, wecomUsers map[string]utils.WecomContactUser, deptIdByWecom map[int64]int64, result *system.WecomSyncResult) error {
	defaultRoleId, err := wecomDefaultRoleId(ctx)
	if err != nil {
		return err
	}

	// load 本地 wecom 绑定 + 对应用户 + 多部门归属
	var socials []system.SysSocial
	if err := global.OPS_DB.WithContext(ctx).Where("source = ?", "wecom").Find(&socials).Error; err != nil {
		return err
	}
	socialByOpenId := make(map[string]*system.SysSocial, len(socials))
	userIdByOpenId := make(map[string]int64, len(socials))
	localUserIds := make([]int64, 0, len(socials))
	for i := range socials {
		socialByOpenId[socials[i].OpenId] = &socials[i]
		userIdByOpenId[socials[i].OpenId] = socials[i].UserId
		localUserIds = append(localUserIds, socials[i].UserId)
	}
	userById := make(map[int64]*system.SysUser, len(localUserIds))
	userDepts := make(map[int64]map[int64]bool, len(localUserIds)) // userId -> set(localDeptId)
	if len(localUserIds) > 0 {
		var users []system.SysUser
		if err := global.OPS_DB.WithContext(ctx).Where("id IN ?", localUserIds).Find(&users).Error; err != nil {
			return err
		}
		for i := range users {
			userById[users[i].UserId] = &users[i]
		}
		var links []system.SysUserDepartment
		if err := global.OPS_DB.WithContext(ctx).Where("sys_user_id IN ?", localUserIds).Find(&links).Error; err != nil {
			return err
		}
		for _, l := range links {
			if userDepts[l.SysUserId] == nil {
				userDepts[l.SysUserId] = make(map[int64]bool)
			}
			userDepts[l.SysUserId][l.SysDepartmentId] = true
		}
	}

	// 绑定孤儿清理:本地用户已硬删但绑定残留(user_id 悬空)→ 删除绑定。
	// 纯技术映射数据无保留价值,硬删;随后若该企微成员仍在返回集,走下方 createWecomUser 重建。
	orphanIds := make([]int64, 0)
	for openId, rec := range socialByOpenId {
		if _, alive := userById[rec.UserId]; alive {
			continue
		}
		orphanIds = append(orphanIds, rec.ID)
		delete(socialByOpenId, openId)
		delete(userIdByOpenId, openId)
	}
	if len(orphanIds) > 0 {
		if err := global.OPS_DB.WithContext(ctx).Unscoped().Where("id IN ?", orphanIds).Delete(&system.SysSocial{}).Error; err != nil {
			logger.WithCtx(ctx).Mod("wecom").Err(err).Warn(fmt.Sprintf("清理绑定孤儿失败(%d 条,非阻断)", len(orphanIds)))
		} else {
			logger.WithCtx(ctx).Mod("wecom").Warn(fmt.Sprintf("清理绑定孤儿 %d 条(本地用户已删除)", len(orphanIds)))
		}
	}

	// upsert 每个企微成员
	for _, wu := range wecomUsers {
		if rec, exists := socialByOpenId[wu.UserID]; exists {
			u := userById[rec.UserId]
			if u == nil {
				// 兜底:孤儿清理后理论不可达(user_id 已过滤)
				result.UserSkipped++
				continue
			}
			// 在册标记:命中返回集即纳入同步管理(离职差集停用的作用域)。
			// 置位失败仅告警不阻断——保持 false 只会让差集不停用(偏保守方向)
			if !rec.InSyncScope {
				if err := global.OPS_DB.WithContext(ctx).Model(&system.SysSocial{}).
					Where("id = ?", rec.ID).Update("in_sync_scope", true).Error; err != nil {
					logger.WithCtx(ctx).Mod("wecom").Err(err).Warn("标记同步在册失败(非阻断)")
				} else {
					rec.InSyncScope = true
				}
			}
			// 复职恢复:曾因离职被同步停用(disabled_by_sync)且本期回到企微返回集 → 恢复启用并清标。
			// 仅恢复"同步停用"的:管理员手动停用的在职员工(无标记)不被同步反复启用
			if rec.DisabledBySync && u.Status == wecomStatusDisable {
				if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).
					Where("id = ?", u.UserId).Update("status", wecomStatusEnable).Error; err != nil {
					logger.WithCtx(ctx).Mod("wecom").Err(err).Warn(fmt.Sprintf("复职恢复用户[%s]失败(非阻断)", wu.UserID))
				} else {
					_ = global.OPS_DB.WithContext(ctx).Model(&system.SysSocial{}).
						Where("id = ?", rec.ID).Update("disabled_by_sync", false).Error
					u.Status = wecomStatusEnable
					result.UserRestored++
				}
			}
			if s.applyUserUpdates(ctx, u, &wu, deptIdByWecom, userDepts[u.UserId]) {
				result.UserUpdated++
			} else {
				result.UserSkipped++
			}
			s.applySocialUpdates(ctx, rec, &wu)
		} else {
			if err := s.createWecomUser(ctx, &wu, deptIdByWecom, defaultRoleId); err != nil {
				return fmt.Errorf("创建用户[%s/%s]失败: %w", wu.UserID, wu.Name, err)
			}
			result.UserCreated++
		}
	}

	// 离职停用:仅作用于"同步在册"绑定(in_sync_scope=true,曾出现在同步返回集)——
	// 此前在册且本期不在企微返回集 = 从在册变不在册(离职/移出通讯录),置停用并打同步停用标记
	// (供复职时识别恢复;管理员手动停用不在此列);扫码登录建号的绑定从不在册,不受此停用影响
	disabledUserIds := make([]int64, 0)
	disabledSocialIds := make([]int64, 0)
	for openId, uid := range userIdByOpenId {
		if _, stillIn := wecomUsers[openId]; stillIn {
			continue
		}
		rec := socialByOpenId[openId]
		if rec == nil || !rec.InSyncScope {
			continue
		}
		if u := userById[uid]; u != nil && u.Status == wecomStatusEnable {
			disabledUserIds = append(disabledUserIds, uid)
			disabledSocialIds = append(disabledSocialIds, rec.ID)
		}
	}
	if len(disabledUserIds) > 0 {
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).
			Where("id IN ?", disabledUserIds).Update("status", wecomStatusDisable).Error; err != nil {
			return fmt.Errorf("离职停用失败: %w", err)
		}
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysSocial{}).
			Where("id IN ?", disabledSocialIds).Update("disabled_by_sync", true).Error; err != nil {
			logger.WithCtx(ctx).Mod("wecom").Err(err).Warn("离职停用打同步停用标记失败(非阻断,影响复职自动恢复)")
		}
		result.UserDisabled = len(disabledUserIds)
	}
	return nil
}

// createWecomUser 新建本地用户(随机不可用口令 + 默认角色)+ wecom 绑定(在册) + 多部门归属,单事务。
func (s *WecomContactService) createWecomUser(ctx context.Context, wu *utils.WecomContactUser, deptIdByWecom map[int64]int64, defaultRoleId int64) error {
	localDepts := resolveLocalDeptIds(wu.Department, deptIdByWecom)
	mainDept := int64(0)
	if len(localDepts) > 0 {
		mainDept = localDepts[0]
	}
	pwd, err := utils.RandomWecomStateToken() // 32 位 hex,与扫码登录建号同口径(不可用于密码登录)
	if err != nil {
		return err
	}
	nick := wu.Name
	if nick == "" {
		nick = wu.Mobile
	}
	if nick == "" {
		nick = "wecom_" + wu.UserID
	}
	now := time.Now()
	return global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := system.SysUser{
			UUID:              uuid.New(),
			UserName:          "wecom_" + wu.UserID, // 前缀避免与手动建号登录名冲突
			NickName:          nick,
			Password:          utils.BcryptHash(pwd),
			Email:             wu.Email,
			Phonenumber:       wu.Mobile,
			Avatar:            wu.Avatar,
			DeptId:            mainDept,
			Sex:               WecomGenderToSex(wu.Gender),
			RoleId:            defaultRoleId,
			Status:            wecomStatusEnable,
			PasswordUpdatedAt: &now, // 标记刚设置,避免密码过期判定(对齐扫码登录 loginOrRegister)
		}
		if err := tx.Create(&user).Error; err != nil {
			return fmt.Errorf("创建用户失败: %w", err)
		}
		if err := tx.Create(&system.SysUserRole{SysUserId: user.UserId, SysRoleId: defaultRoleId}).Error; err != nil {
			return fmt.Errorf("分配角色失败: %w", err)
		}
		if err := tx.Create(&system.SysSocial{
			UserId:      user.UserId,
			Source:      "wecom",
			OpenId:      wu.UserID,
			AuthId:      "wecom_" + wu.UserID,
			NickName:    nick,
			Avatar:      wu.Avatar,
			Email:       wu.Email,
			Mobile:      wu.Mobile,
			InSyncScope: true, // 同步建号即在册,纳入离职差集停用管理
		}).Error; err != nil {
			return fmt.Errorf("建立社交绑定失败: %w", err)
		}
		if len(localDepts) > 0 {
			links := make([]system.SysUserDepartment, 0, len(localDepts))
			for _, did := range localDepts {
				links = append(links, system.SysUserDepartment{SysUserId: user.UserId, SysDepartmentId: did})
			}
			if err := tx.Create(&links).Error; err != nil {
				return fmt.Errorf("建立多部门归属失败: %w", err)
			}
		}
		return nil
	})
}

// applyUserUpdates 增量更新已绑定用户:比对昵称/头像/手机/邮箱/性别/主部门/多部门集合,有变化才写。返回是否变化。
func (s *WecomContactService) applyUserUpdates(ctx context.Context, u *system.SysUser, wu *utils.WecomContactUser, deptIdByWecom map[int64]int64, existingDepts map[int64]bool) bool {
	localDepts := resolveLocalDeptIds(wu.Department, deptIdByWecom)
	mainDept := int64(0)
	if len(localDepts) > 0 {
		mainDept = localDepts[0]
	}
	updates := map[string]interface{}{}
	// 空值不覆盖:通讯录接口对无权限字段可能返回空,防止清掉登录链路已写入的真实资料
	if wu.Name != "" && wu.Name != u.NickName {
		updates["nick_name"] = wu.Name
	}
	if wu.Avatar != "" && wu.Avatar != u.Avatar {
		updates["avatar"] = wu.Avatar
	}
	if wu.Mobile != "" && wu.Mobile != u.Phonenumber {
		updates["phonenumber"] = wu.Mobile
	}
	if wu.Email != "" && wu.Email != u.Email {
		updates["email"] = wu.Email
	}
	// 仅当企微返回真实性别("1"男/"2"女)才更新;通讯录 user/list 可能仅返回"0"/缺失(未 snsapi 授权),
	// 此时不应覆盖登录链路已写入的真实性别,故跳过。
	if wu.Gender == "1" || wu.Gender == "2" {
		if newSex := WecomGenderToSex(wu.Gender); newSex != u.Sex {
			updates["sex"] = newSex
		}
	}
	if mainDept != u.DeptId {
		updates["dept_id"] = mainDept
	}
	newSet := make(map[int64]bool, len(localDepts))
	for _, d := range localDepts {
		newSet[d] = true
	}
	deptsChanged := !deptSetEqual(existingDepts, newSet)

	if len(updates) > 0 {
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysUser{}).Where("id = ?", u.UserId).Updates(updates).Error; err != nil {
			logger.WithCtx(ctx).Mod("wecom").Err(err).Warn("更新用户资料失败(非阻断)")
			return false
		}
	}
	if deptsChanged {
		s.rebuildUserDepartments(ctx, u.UserId, localDepts)
	}
	return len(updates) > 0 || deptsChanged
}

// rebuildUserDepartments 按企微部门列表重建用户多部门归属(删后建)。
func (s *WecomContactService) rebuildUserDepartments(ctx context.Context, userId int64, deptIds []int64) {
	global.OPS_DB.WithContext(ctx).Where("sys_user_id = ?", userId).Delete(&system.SysUserDepartment{})
	if len(deptIds) == 0 {
		return
	}
	links := make([]system.SysUserDepartment, 0, len(deptIds))
	for _, did := range deptIds {
		links = append(links, system.SysUserDepartment{SysUserId: userId, SysDepartmentId: did})
	}
	if err := global.OPS_DB.WithContext(ctx).Create(&links).Error; err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Warn("重建用户多部门归属失败(非阻断)")
	}
}

// applySocialUpdates 增量刷新 sys_social 快照(非阻断,失败仅告警)。
func (s *WecomContactService) applySocialUpdates(ctx context.Context, rec *system.SysSocial, wu *utils.WecomContactUser) {
	updates := map[string]interface{}{}
	if wu.Name != "" && wu.Name != rec.NickName {
		updates["nick_name"] = wu.Name
	}
	if wu.Avatar != "" && wu.Avatar != rec.Avatar {
		updates["avatar"] = wu.Avatar
	}
	if wu.Mobile != "" && wu.Mobile != rec.Mobile {
		updates["mobile"] = wu.Mobile
	}
	if wu.Email != "" && wu.Email != rec.Email {
		updates["email"] = wu.Email
	}
	if len(updates) == 0 {
		return
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysSocial{}).Where("id = ?", rec.ID).Updates(updates).Error; err != nil {
		logger.WithCtx(ctx).Mod("wecom").Err(err).Warn("刷新企微绑定快照失败(非阻断)")
	}
}

// resolveLocalDeptIds 把企微部门 id 列表映射为本地部门 id 列表(过滤未同步的)。
func resolveLocalDeptIds(wecomDeptIds []int64, deptIdByWecom map[int64]int64) []int64 {
	out := make([]int64, 0, len(wecomDeptIds))
	for _, wid := range wecomDeptIds {
		if lid, ok := deptIdByWecom[wid]; ok && lid != 0 {
			out = append(out, lid)
		}
	}
	return out
}

// deptSetEqual 判断两个部门 id 集合是否相同。
func deptSetEqual(a, b map[int64]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// syncPosts 同步岗位:企微成员 position(职务)派生为岗位(归属用户主部门),并维护用户-岗位关联。
// 岗位来源以 post_code 前缀 'wecom_' 标识,同步仅管理该来源的岗位与关联,不触碰手动分配的岗位。
// 关联采用"全量重建"(删所有 wecom 岗位的 sys_user_post→按当前 position 重建),使调岗/职务变更正确收敛。
func (s *WecomContactService) syncPosts(ctx context.Context, wecomUsers map[string]utils.WecomContactUser, deptIdByWecom map[int64]int64, result *system.WecomSyncResult) error {
	type postKey struct {
		deptId int64
		name   string
	}

	// 1. 收集目标岗位键(主部门,职务名);position 为空或部门未同步的成员不产生目标(关联重建阶段清旧)
	targetKey := make(map[string]postKey, len(wecomUsers))
	keys := make([]postKey, 0, len(wecomUsers))
	seen := make(map[postKey]bool, len(wecomUsers))
	for _, wu := range wecomUsers {
		pos := strings.TrimSpace(wu.Position)
		localDepts := resolveLocalDeptIds(wu.Department, deptIdByWecom)
		if pos == "" || len(localDepts) == 0 {
			continue
		}
		k := postKey{localDepts[0], pos}
		targetKey[wu.UserID] = k
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	result.PostTotal = len(keys)

	// 2. load 本地 wecom 岗位(post_code 前缀 'wecom_')→ (deptId,postName)→postId
	var localPosts []system.SysPost
	if err := global.OPS_DB.WithContext(ctx).Where("post_code LIKE ?", "wecom\\_%").Find(&localPosts).Error; err != nil {
		return err
	}
	postIdByKey := make(map[postKey]int64, len(localPosts))
	wecomPostIds := make([]int64, 0, len(localPosts))
	for i := range localPosts {
		postIdByKey[postKey{localPosts[i].DeptId, localPosts[i].PostName}] = localPosts[i].PostId
		wecomPostIds = append(wecomPostIds, localPosts[i].PostId)
	}

	// 3. 为缺失键创建岗位(雪花回调自动生成 post_id)
	for _, k := range keys {
		if _, ok := postIdByKey[k]; ok {
			continue
		}
		p := system.SysPost{
			DeptId:   k.deptId,
			PostCode: fmt.Sprintf("wecom_%d_%s", k.deptId, postCodeHex(k.name)),
			PostName: k.name,
			Status:   wecomStatusEnable,
			Remark:   "企微通讯录同步",
		}
		if err := global.OPS_DB.WithContext(ctx).Create(&p).Error; err != nil {
			return fmt.Errorf("创建岗位[%s]失败: %w", k.name, err)
		}
		postIdByKey[k] = p.PostId
		wecomPostIds = append(wecomPostIds, p.PostId)
		result.PostCreated++
	}

	// 4. load sys_social(wecom)→userId,供关联重建
	var socials []system.SysSocial
	if err := global.OPS_DB.WithContext(ctx).Where("source = ?", "wecom").Find(&socials).Error; err != nil {
		return err
	}
	userIdByOpenId := make(map[string]int64, len(socials))
	for i := range socials {
		userIdByOpenId[socials[i].OpenId] = socials[i].UserId
	}

	// 5. 全量重建用户-岗位关联:删所有 wecom 岗位的 sys_user_post(只 wecom 来源,不碰手动分配)→ 批量插目标
	if len(wecomPostIds) > 0 {
		if err := global.OPS_DB.WithContext(ctx).Where("sys_post_id IN ?", wecomPostIds).Delete(&system.SysUserPost{}).Error; err != nil {
			return fmt.Errorf("清理用户岗位关联失败: %w", err)
		}
	}
	links := make([]system.SysUserPost, 0, len(targetKey))
	for openId, k := range targetKey {
		uid, ok := userIdByOpenId[openId]
		if !ok {
			continue // 用户未同步(syncUsers 兜底,理论上已建)
		}
		links = append(links, system.SysUserPost{SysUserId: uid, SysPostId: postIdByKey[k]})
	}
	if len(links) > 0 {
		if err := global.OPS_DB.WithContext(ctx).Create(&links).Error; err != nil {
			return fmt.Errorf("建立用户岗位关联失败: %w", err)
		}
	}
	return nil
}

// WecomGenderToSex 企微性别(string,"0"未定义/"1"男/"2"女)→ 项目 Sex 字典值(string,0男1女2未知)。
// 两套值体系不同,必须转换;Gender 为 string(企微官方定义为 string,用数值类型接会反序列化失败)。
func WecomGenderToSex(g string) string {
	switch g {
	case "1":
		return "0" // 企微男 → 项目男
	case "2":
		return "1" // 企微女 → 项目女
	default:
		return "2" // 企微"0"未定义/空/其他 → 项目未知
	}
}

// postCodeHex 把岗位名映射为 8 位 hex(sha1 前4字节),保证 post_code 稳定。
// post_code 全局唯一由 (deptId,position) 键唯一性保证:同 deptId+同 position 生成同 code,
// 不同 deptId 则前缀 wecom_<deptId>_ 不同。
func postCodeHex(name string) string {
	h := sha1.Sum([]byte(name))
	return hex.EncodeToString(h[:4])
}
