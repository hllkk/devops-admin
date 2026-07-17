package system

import (
	"context"
	"fmt"

	sysModel "github.com/hllkk/devops-admin/server/model/system"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// 部门/岗位建表与种子; 与菜单/授权无 context 依赖, 排在授权之后即可
const initOrderDepartment = initOrderRoleMenu + 1

type initDepartment struct{}

// auto run
func init() {
	system.RegisterInit(initOrderDepartment, &initDepartment{})
}

func (i *initDepartment) InitializerName() string {
	return sysModel.SysDepartment{}.TableName()
}

func (i *initDepartment) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	return ctx, db.AutoMigrate(&sysModel.SysDepartment{}, &sysModel.SysPost{}, &sysModel.SysDataAccessLog{}, &sysModel.SysRoleDepartment{})
}

func (i *initDepartment) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	m := db.Migrator()
	return m.HasTable(&sysModel.SysDepartment{}) && m.HasTable(&sysModel.SysPost{})
}

func (i *initDepartment) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	// 部门/岗位状态: '0' 正常 / '1' 停用(对齐前端 Api.System.Dept/Post)
	const statusNormal = "0"

	// 1. 顶级部门: XXX科技(祖级链固定为 "0")
	rootDepts := []sysModel.SysDepartment{
		{DeptName: "XXX科技", ParentId: 0, Ancestors: "0", OrderNum: 0, Status: statusNormal},
	}
	if err := db.Create(&rootDepts).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", sysModel.SysDepartment{}.TableName())
	}
	rootID := rootDepts[0].DeptId

	// 2. 子部门: 北京总部 / 天津工厂; 祖级链 = 父级祖级链 + "," + 父ID
	childAncestors := fmt.Sprintf("0,%d", rootID)
	childDepts := []sysModel.SysDepartment{
		{DeptName: "北京总部", ParentId: rootID, Ancestors: childAncestors, OrderNum: 1, Status: statusNormal},
		{DeptName: "天津工厂", ParentId: rootID, Ancestors: childAncestors, OrderNum: 2, Status: statusNormal},
	}
	if err := db.Create(&childDepts).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s子部门初始化失败!", sysModel.SysDepartment{}.TableName())
	}

	// 建立部门名 -> 部门ID 映射, 供岗位归属引用
	deptIDByName := map[string]int64{
		rootDepts[0].DeptName: rootDepts[0].DeptId,
	}
	for _, d := range childDepts {
		deptIDByName[d.DeptName] = d.DeptId
	}

	// 3. 岗位: XXX科技->总经理 / 北京总部->研发总监 / 天津工厂->普通员工
	posts := []sysModel.SysPost{
		{DeptId: deptIDByName["XXX科技"], PostCode: "ceo", PostName: "总经理", PostSort: 1, Status: statusNormal},
		{DeptId: deptIDByName["北京总部"], PostCode: "rd_director", PostName: "研发总监", PostSort: 1, Status: statusNormal},
		{DeptId: deptIDByName["天津工厂"], PostCode: "staff", PostName: "普通员工", PostSort: 1, Status: statusNormal},
	}
	if err := db.Create(&posts).Error; err != nil {
		return ctx, errors.Wrapf(err, "%s表数据初始化失败!", sysModel.SysPost{}.TableName())
	}

	// 组合部门结果写入 context(对齐 sys_menu 返回风格; 独立分配切片, 避免改动原数组)
	allDepts := make([]sysModel.SysDepartment, 0, len(rootDepts)+len(childDepts))
	allDepts = append(allDepts, rootDepts...)
	allDepts = append(allDepts, childDepts...)

	next := context.WithValue(ctx, i.InitializerName(), allDepts)
	return next, nil
}

func (i *initDepartment) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	// 检查顶级部门与岗位两层哨兵, 防止部分初始化误判为完整
	if errors.Is(db.Where("dept_name = ?", "XXX科技").
		First(&sysModel.SysDepartment{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	if errors.Is(db.Where("post_name = ?", "总经理").
		First(&sysModel.SysPost{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}
