package system

import (
	"bytes"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/hllkk/devops-admin/server/utils/excel"
)

// 各模块导出/导入列定义(中文表头 → struct 字段名)。
// Field 走 reflect.FieldByName,可命中嵌入基座 OPS_AUDIT_MODEL 的提升字段(如 CreatedAt)。
// 列对齐前端各列表的展示列;字典值字段(sex/status/businessType 等)导出原始码值,与 gva 一致。
var (
	userHeaders = []excel.Header{
		{Field: "UserName", Title: "用户名"},
		{Field: "NickName", Title: "昵称"},
		{Field: "DeptName", Title: "部门"},
		{Field: "Email", Title: "邮箱"},
		{Field: "Phonenumber", Title: "手机号"},
		{Field: "Sex", Title: "性别"},
		{Field: "Status", Title: "状态"},
		{Field: "CreatedAt", Title: "创建时间"},
	}
	// userImportHeaders 用户导入模板列;Field 与 UserService.ImportUsers 读取的 map key 对齐。
	userImportHeaders = []excel.Header{
		{Field: "UserName", Title: "用户名"},
		{Field: "NickName", Title: "昵称"},
		{Field: "DeptId", Title: "部门ID"},
		{Field: "Phonenumber", Title: "手机号"},
		{Field: "Email", Title: "邮箱"},
		{Field: "Sex", Title: "性别(0男1女2未知)"},
		{Field: "Status", Title: "状态(0正常1停用)"},
	}
	roleHeaders = []excel.Header{
		{Field: "RoleName", Title: "角色名称"},
		{Field: "RoleKey", Title: "权限字符"},
		{Field: "RoleSort", Title: "显示顺序"},
		{Field: "Status", Title: "状态"},
		{Field: "CreatedAt", Title: "创建时间"},
	}
	postHeaders = []excel.Header{
		{Field: "PostCode", Title: "岗位编码"},
		{Field: "PostCategory", Title: "岗位类别"},
		{Field: "PostName", Title: "岗位名称"},
		{Field: "PostSort", Title: "显示顺序"},
		{Field: "Status", Title: "状态"},
		{Field: "CreatedAt", Title: "创建时间"},
	}
	dictTypeHeaders = []excel.Header{
		{Field: "DictName", Title: "字典名称"},
		{Field: "DictType", Title: "字典类型"},
		{Field: "Remark", Title: "备注"},
		{Field: "CreatedAt", Title: "创建时间"},
	}
	dictDataHeaders = []excel.Header{
		{Field: "DictLabel", Title: "字典标签"},
		{Field: "DictValue", Title: "字典键值"},
		{Field: "DictSort", Title: "字典排序"},
		{Field: "DictType", Title: "字典类型"},
		{Field: "IsDefault", Title: "是否默认"},
		{Field: "Remark", Title: "备注"},
		{Field: "CreatedAt", Title: "创建时间"},
	}
	loginLogHeaders = []excel.Header{
		{Field: "UserName", Title: "用户账号"},
		{Field: "DeviceType", Title: "设备类型"},
		{Field: "Ipaddr", Title: "登录IP"},
		{Field: "LoginLocation", Title: "登录地点"},
		{Field: "Browser", Title: "浏览器"},
		{Field: "Os", Title: "操作系统"},
		{Field: "Status", Title: "状态"},
		{Field: "LoginTime", Title: "登录时间"},
	}
	operLogHeaders = []excel.Header{
		{Field: "Title", Title: "系统模块"},
		{Field: "BusinessType", Title: "操作类型"},
		{Field: "OperName", Title: "操作人员"},
		{Field: "OperIp", Title: "操作IP"},
		{Field: "OperLocation", Title: "操作地点"},
		{Field: "Status", Title: "状态"},
		{Field: "OperTime", Title: "操作时间"},
		{Field: "CostTime", Title: "耗时(毫秒)"},
	}
)

// writeXlsx 统一输出 xlsx 二进制流,适配前端 useDownload(POST 表单 + 读 Download-Filename 头)。
// name 为业务名,最终兜底文件名 = name.xlsx(URL 编码);前端传入的 filename 优先。
// success 头供操作日志中间件识别为成功下载。
func writeXlsx(c *gin.Context, name string, buf *bytes.Buffer) {
	const ct = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	filename := url.QueryEscape(name + ".xlsx")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", ct)
	c.Header("success", "true")
	c.Header("Download-Filename", filename)
	c.Data(http.StatusOK, ct, buf.Bytes())
}
