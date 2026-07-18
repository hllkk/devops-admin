package system

import (
	"context"
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
)

// DictTypeService 字典类型业务服务(对齐前端 /system/dict/type/* 资源)
type DictTypeService struct{}

// GetDictTypeList 分页查字典类型列表(对齐前端 GET /system/dict/type/list)。
// 按 dictName/dictType 模糊过滤;分页统一走 PageInfo.LimitOffset(内置 MaxPageSize=100 截断)。
func (s *DictTypeService) GetDictTypeList(ctx context.Context, q systemReq.DictTypeSearch) (list []system.SysDictType, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysDictType{})
	if q.DictName != "" {
		db = db.Where("dict_name LIKE ?", "%"+q.DictName+"%")
	}
	if q.DictType != "" {
		db = db.Where("dict_type LIKE ?", "%"+q.DictType+"%")
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order("dict_id DESC").Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order("dict_id DESC").Find(&list).Error
	}
	return
}

// CreateDictType 新增字典类型;dictType 唯一性校验,createBy 填审计字段。
func (s *DictTypeService) CreateDictType(ctx context.Context, req systemReq.DictTypeOperateParams, createBy int64) error {
	if req.DictType == "" {
		return errors.New("字典类型不能为空")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDictType{}).
		Where("dict_type = ?", req.DictType).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("字典类型已存在")
	}
	dt := system.SysDictType{
		DictName: req.DictName,
		DictType: req.DictType,
		Remark:   req.Remark,
	}
	// CreateBy/UpdateBy 为内嵌 OPS_AUDIT_MODEL 的提升字段,struct literal 中不可直接命名,改用赋值写入。
	dt.CreateBy = createBy
	dt.UpdateBy = createBy
	return global.OPS_DB.WithContext(ctx).Create(&dt).Error
}

// UpdateDictType 修改字典类型;dictId 必填,dictType 唯一性校验(排除自身),
// 若 dictType 变更则同步字典数据中的冗余 dict_type(对齐 RuoYi,保持引用一致),updateBy 填审计字段。
func (s *DictTypeService) UpdateDictType(ctx context.Context, req systemReq.DictTypeOperateParams, updateBy int64) error {
	if req.DictId == 0 {
		return errors.New("字典主键不能为空")
	}
	if req.DictType == "" {
		return errors.New("字典类型不能为空")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDictType{}).
		Where("dict_type = ? AND dict_id <> ?", req.DictType, req.DictId).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("字典类型已存在")
	}
	var old system.SysDictType
	if err := global.OPS_DB.WithContext(ctx).Where("dict_id = ?", req.DictId).First(&old).Error; err != nil {
		return err
	}
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDictType{}).Where("dict_id = ?", req.DictId).
		Updates(map[string]interface{}{
			"dict_name": req.DictName,
			"dict_type": req.DictType,
			"remark":    req.Remark,
			"update_by": updateBy,
		}).Error; err != nil {
		return err
	}
	// dictType 变更时同步 dict_data 冗余列,避免数据引用失联
	if old.DictType != "" && old.DictType != req.DictType {
		if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDictData{}).
			Where("dict_type = ?", old.DictType).Update("dict_type", req.DictType).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteDictType 批量删除字典类型;级联清理对应 dict_data(对齐 RuoYi,避免孤儿数据)。
func (s *DictTypeService) DeleteDictType(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	// 先取出涉及的 dict_type(删 type 后无法再反查)
	var types []string
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDictType{}).
		Where("dict_id IN ?", ids).Pluck("dict_type", &types).Error; err != nil {
		return err
	}
	if len(types) > 0 {
		if err := global.OPS_DB.WithContext(ctx).Where("dict_type IN ?", types).Delete(&system.SysDictData{}).Error; err != nil {
			return err
		}
	}
	return global.OPS_DB.WithContext(ctx).Where("dict_id IN ?", ids).Delete(&system.SysDictType{}).Error
}

// GetDictTypeOptionList 获取全部字典类型(下拉选择框用,不分页,对齐前端 optionselect)。
func (s *DictTypeService) GetDictTypeOptionList(ctx context.Context) (list []system.SysDictType, err error) {
	err = global.OPS_DB.WithContext(ctx).Order("dict_id DESC").Find(&list).Error
	return
}

// DictDataService 字典数据业务服务(对齐前端 /system/dict/data/* 资源)
type DictDataService struct{}

// dictDataOrder 字典数据统一排序:dict_sort 升序,同序按 dict_code 升序(对齐 RuoYi/前端展示惯例)。
const dictDataOrder = "dict_sort ASC, dict_code ASC"

// GetDictDataList 分页查字典数据列表(对齐前端 GET /system/dict/data/list)。
// dictLabel 模糊、dictType 精确(页面右侧按选定 type 过滤);分页走 PageInfo.LimitOffset。
func (s *DictDataService) GetDictDataList(ctx context.Context, q systemReq.DictDataSearch) (list []system.SysDictData, total int64, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysDictData{})
	if q.DictLabel != "" {
		db = db.Where("dict_label LIKE ?", "%"+q.DictLabel+"%")
	}
	if q.DictType != "" {
		db = db.Where("dict_type = ?", q.DictType)
	}
	limit, offset := q.LimitOffset()
	if limit > 0 {
		err = db.Count(&total).Order(dictDataOrder).Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = db.Count(&total).Order(dictDataOrder).Find(&list).Error
	}
	return
}

// CreateDictData 新增字典数据;createBy 填审计字段。
func (s *DictDataService) CreateDictData(ctx context.Context, req systemReq.DictDataOperateParams, createBy int64) error {
	d := system.SysDictData{
		DictSort:  req.DictSort,
		DictLabel: req.DictLabel,
		DictValue: req.DictValue,
		DictType:  req.DictType,
		CssClass:  req.CssClass,
		ListClass: req.ListClass,
		IsDefault: req.IsDefault,
		Remark:    req.Remark,
	}
	// CreateBy/UpdateBy 为内嵌 OPS_AUDIT_MODEL 的提升字段,struct literal 中不可直接命名,改用赋值写入。
	d.CreateBy = createBy
	d.UpdateBy = createBy
	return global.OPS_DB.WithContext(ctx).Create(&d).Error
}

// UpdateDictData 修改字典数据;dictCode 必填,updateBy 填审计字段。
func (s *DictDataService) UpdateDictData(ctx context.Context, req systemReq.DictDataOperateParams, updateBy int64) error {
	if req.DictCode == 0 {
		return errors.New("字典编码不能为空")
	}
	return global.OPS_DB.WithContext(ctx).Model(&system.SysDictData{}).Where("dict_code = ?", req.DictCode).
		Updates(map[string]interface{}{
			"dict_sort":  req.DictSort,
			"dict_label": req.DictLabel,
			"dict_value": req.DictValue,
			"dict_type":  req.DictType,
			"css_class":  req.CssClass,
			"list_class": req.ListClass,
			"is_default": req.IsDefault,
			"remark":     req.Remark,
			"update_by":  updateBy,
		}).Error
}

// DeleteDictData 批量删除字典数据(按 dictCode)。
func (s *DictDataService) DeleteDictData(ctx context.Context, codes []int64) error {
	if len(codes) == 0 {
		return errors.New("未选择删除项")
	}
	return global.OPS_DB.WithContext(ctx).Where("dict_code IN ?", codes).Delete(&system.SysDictData{}).Error
}

// GetDictDataByType 按字典类型查全部字典数据(DictTag/DictRadio 渲染用,不分页,对齐前端 GET /system/dict/data/type/{dictType})。
func (s *DictDataService) GetDictDataByType(ctx context.Context, dictType string) (list []system.SysDictData, err error) {
	err = global.OPS_DB.WithContext(ctx).Where("dict_type = ?", dictType).Order(dictDataOrder).Find(&list).Error
	return
}
