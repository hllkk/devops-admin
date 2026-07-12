package request

import (
	"encoding/json"
	"testing"
)

// 验证线上 ID 全字符串：标量 *string、数组 []string 都能从 JSON 正确绑定。
func TestSysUserReqBindsStringIDs(t *testing.T) {
	raw := `{"userId":"123","deptId":"42","userName":"alice","status":"0","roleIds":["1","2"],"postIds":[]}`
	var req SysUserReq
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.UserId == nil || *req.UserId != "123" {
		t.Errorf("userId 绑定错误：%v", req.UserId)
	}
	if req.DeptId == nil || *req.DeptId != "42" {
		t.Errorf("deptId 绑定错误：%v", req.DeptId)
	}
	if len(req.RoleIds) != 2 || req.RoleIds[0] != "1" || req.RoleIds[1] != "2" {
		t.Errorf("roleIds 绑定错误：%v", req.RoleIds)
	}
	if len(req.PostIds) != 0 {
		t.Errorf("postIds 应为空切片，实际 %v", req.PostIds)
	}
}
