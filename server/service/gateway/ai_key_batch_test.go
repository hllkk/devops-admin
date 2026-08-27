package gateway

import (
	"encoding/json"
	"testing"

	"github.com/hllkk/devops-admin/server/model/system"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
)

// TestAiKeyBatchCreateParamsBind 请求体绑定单测：userIds 兼容前端雪花 IdType 字符串数组
// (Int64StringSlice.UnmarshalJSON)，deptId 走 ",string" 指针 tag 解析字符串。
func TestAiKeyBatchCreateParamsBind(t *testing.T) {
	// 前端实际形态：userIds 字符串数组 + deptId 字符串
	var p gatewayReq.AiKeyBatchCreateParams
	if err := json.Unmarshal([]byte(`{"userIds":["123456789012345","9223372036854775807"],"deptId":"42"}`), &p); err != nil {
		t.Fatalf("绑定失败: %v", err)
	}
	if len(p.UserIds) != 2 || p.UserIds[0] != 123456789012345 || p.UserIds[1] != 9223372036854775807 {
		t.Errorf("userIds 解析错误: %v", p.UserIds)
	}
	if p.DeptId == nil || *p.DeptId != 42 {
		t.Errorf("deptId 解析错误: %v", p.DeptId)
	}
	// 仅 userIds(指定用户模式)，deptId 缺省为 nil
	var p2 gatewayReq.AiKeyBatchCreateParams
	if err := json.Unmarshal([]byte(`{"userIds":["1","2"]}`), &p2); err != nil {
		t.Fatalf("仅 userIds 绑定失败: %v", err)
	}
	if p2.DeptId != nil {
		t.Errorf("deptId 缺省应为 nil: %v", p2.DeptId)
	}
}

// classifyBatchTargets 纯函数单测：批量开通主 Key 的目标分类(停用/已存在/待创建)。

func TestClassifyBatchTargets_Mixed(t *testing.T) {
	users := []system.SysUser{
		{UserId: 101, NickName: "正常用户", Status: "0"},
		{UserId: 102, NickName: "停用用户", Status: "1"},
		{UserId: 103, NickName: "已有主Key", Status: "0"},
		{UserId: 104, NickName: "又一个正常用户", Status: "0"},
	}
	existing := map[int64]bool{103: true}

	toCreate, skipped, failed := classifyBatchTargets(users, existing)

	if len(toCreate) != 2 || toCreate[0].UserId != 101 || toCreate[1].UserId != 104 {
		t.Errorf("待创建分类错误: %v", userIds(toCreate))
	}
	if skipped != 1 {
		t.Errorf("已有主 Key 应跳过 1 个,实际 %d", skipped)
	}
	if len(failed) != 1 || failed[0].UserId != 102 || failed[0].Reason != "用户已停用" {
		t.Errorf("停用用户应进失败列表: %v", failed)
	}
}

func TestClassifyBatchTargets_DisabledTakesPrecedenceOverExisting(t *testing.T) {
	// 停用且已有主 Key：判定顺序停用优先(状态异常先报,避免"看似跳过实则账号不可用"误判)
	users := []system.SysUser{{UserId: 201, NickName: "停用且有Key", Status: "1"}}
	existing := map[int64]bool{201: true}

	toCreate, skipped, failed := classifyBatchTargets(users, existing)

	if len(toCreate) != 0 || skipped != 0 {
		t.Errorf("停用用户不应进入创建/跳过: create=%d skipped=%d", len(toCreate), skipped)
	}
	if len(failed) != 1 {
		t.Errorf("应报失败 1 条,实际 %d", len(failed))
	}
}

func TestClassifyBatchTargets_DuplicateIdsInBatch(t *testing.T) {
	// 同批重复ID(按部门+手选并集的防御场景)：仅创建一次
	users := []system.SysUser{
		{UserId: 301, NickName: "重复A", Status: "0"},
		{UserId: 301, NickName: "重复A", Status: "0"},
	}
	toCreate, skipped, _ := classifyBatchTargets(users, map[int64]bool{})

	if len(toCreate) != 1 {
		t.Errorf("重复ID应去重只创建一次,实际 %d", len(toCreate))
	}
	if skipped != 1 {
		t.Errorf("第二个重复ID应算跳过,实际 %d", skipped)
	}
}

// TestClassifyBatchTargets_AllSuccessFailedNotNull 零失败时 failed 必须是空切片而非 nil：
// Go nil 切片 JSON 序列化为 null，违反「空数组=全部成功」契约；前端直接 data.failed.length
// 消费，null 会让渲染链 TypeError 崩溃(批量弹窗冻结、modal 关不掉)。
func TestClassifyBatchTargets_AllSuccessFailedNotNull(t *testing.T) {
	_, _, failed := classifyBatchTargets(
		[]system.SysUser{{UserId: 401, NickName: "正常用户", Status: "0"}},
		map[int64]bool{},
	)
	if failed == nil {
		t.Fatal("零失败时 failed 应为空切片而非 nil(JSON 会序列化成 null,违反空数组契约)")
	}
	raw, err := json.Marshal(failed)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	if string(raw) != "[]" {
		t.Errorf("零失败时 failed 应序列化为 [],实际 %s", raw)
	}
}

func userIds(users []system.SysUser) []int64 {
	ids := make([]int64, 0, len(users))
	for i := range users {
		ids = append(ids, users[i].UserId)
	}
	return ids
}
