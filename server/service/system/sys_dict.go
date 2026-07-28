package system

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
	systemReq "github.com/hllkk/devops-admin/server/model/system/request"
	"gorm.io/gorm"
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
	dictTypeChanged := old.DictType != "" && old.DictType != req.DictType
	// 主表更新与 dict_data 冗余列同步须原子:中途失败会让 dict_data 仍指向旧 dictType,
	// GetDictDataByType 按新 type 查不到(引用失联),故整体走事务。
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&system.SysDictType{}).Where("dict_id = ?", req.DictId).
			Updates(map[string]interface{}{
				"dict_name": req.DictName,
				"dict_type": req.DictType,
				"remark":    req.Remark,
				"update_by": updateBy,
			}).Error; err != nil {
			return err
		}
		// dictType 变更时同步 dict_data 冗余列,避免数据引用失联
		if dictTypeChanged {
			return tx.Model(&system.SysDictData{}).
				Where("dict_type = ?", old.DictType).Update("dict_type", req.DictType).Error
		}
		return nil
	}); err != nil {
		return err
	}
	// dictType 变更:失效旧 key(数据已迁走)与新 key(按新 type 重建),防缓存陈旧
	if dictTypeChanged {
		invalidateDictDataCache(ctx, []string{old.DictType, req.DictType})
	}
	return nil
}

// DeleteDictType 批量删除字典类型;事务内级联清理对应 dict_data(对齐 RuoYi,避免孤儿数据),删后失效缓存。
func (s *DictTypeService) DeleteDictType(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("未选择删除项")
	}
	// 先取出涉及的 dict_type(删 type 后无法再反查;亦用于删后失效缓存)
	var types []string
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDictType{}).
		Where("dict_id IN ?", ids).Pluck("dict_type", &types).Error; err != nil {
		return err
	}
	// 级联删 dict_data + 删 type 须原子:中途失败会留孤儿 data 或空壳 type,故走事务
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(types) > 0 {
			if err := tx.Where("dict_type IN ?", types).Delete(&system.SysDictData{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("dict_id IN ?", ids).Delete(&system.SysDictType{}).Error
	}); err != nil {
		return err
	}
	invalidateDictDataCache(ctx, types)
	return nil
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

// CreateDictData 新增字典数据;dictType/dictLabel/dictValue 必填 + dictType 存在性校验(与 CreateDictType 对齐,
// 避免空数据或挂在不存在类型上的孤儿数据),createBy 填审计字段,新增后失效对应 dictType 缓存。
func (s *DictDataService) CreateDictData(ctx context.Context, req systemReq.DictDataOperateParams, createBy int64) error {
	if req.DictType == "" {
		return errors.New("字典类型不能为空")
	}
	if req.DictLabel == "" {
		return errors.New("字典标签不能为空")
	}
	if req.DictValue == "" {
		return errors.New("字典键值不能为空")
	}
	var cnt int64
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDictType{}).
		Where("dict_type = ?", req.DictType).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return errors.New("字典类型不存在")
	}
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
	if err := global.OPS_DB.WithContext(ctx).Create(&d).Error; err != nil {
		return err
	}
	invalidateDictDataCache(ctx, []string{req.DictType})
	return nil
}

// UpdateDictData 修改字典数据;dictCode/dictType/dictLabel/dictValue 必填,updateBy 填审计字段;
// dictType 变更时失效新旧两 key,否则失效当前 key,防缓存陈旧。
func (s *DictDataService) UpdateDictData(ctx context.Context, req systemReq.DictDataOperateParams, updateBy int64) error {
	if req.DictCode == 0 {
		return errors.New("字典编码不能为空")
	}
	if req.DictType == "" {
		return errors.New("字典类型不能为空")
	}
	if req.DictLabel == "" {
		return errors.New("字典标签不能为空")
	}
	if req.DictValue == "" {
		return errors.New("字典键值不能为空")
	}
	// 取旧 dictType 用于缓存失效(改 type 时新旧两 key 都要失效)
	var old system.SysDictData
	_ = global.OPS_DB.WithContext(ctx).Select("dict_type").Where("dict_code = ?", req.DictCode).First(&old).Error
	if err := global.OPS_DB.WithContext(ctx).Model(&system.SysDictData{}).Where("dict_code = ?", req.DictCode).
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
		}).Error; err != nil {
		return err
	}
	types := []string{req.DictType}
	if old.DictType != "" && old.DictType != req.DictType {
		types = append(types, old.DictType)
	}
	invalidateDictDataCache(ctx, types)
	return nil
}

// DeleteDictData 批量删除字典数据(按 dictCode);删后失效涉及 dictType 的缓存。
func (s *DictDataService) DeleteDictData(ctx context.Context, codes []int64) error {
	if len(codes) == 0 {
		return errors.New("未选择删除项")
	}
	// 删前取出涉及的 dictType(删后无法反查),用于失效缓存
	var types []string
	global.OPS_DB.WithContext(ctx).Model(&system.SysDictData{}).
		Where("dict_code IN ?", codes).Distinct("dict_type").Pluck("dict_type", &types)
	if err := global.OPS_DB.WithContext(ctx).Where("dict_code IN ?", codes).Delete(&system.SysDictData{}).Error; err != nil {
		return err
	}
	invalidateDictDataCache(ctx, types)
	return nil
}

// GetDictDataByType 按字典类型查全部字典数据(DictTag/DictRadio 渲染用,不分页,对齐前端 GET /system/dict/data/type/{dictType})。
// 该接口在 Casbin 白名单内、所有登录用户每个下拉框渲染都打,故优先读 Redis 缓存;未命中查 DB 后回填。
// 缓存不可用(未配 Redis/异常)一律降级为直查 DB,不阻断渲染。
func (s *DictDataService) GetDictDataByType(ctx context.Context, dictType string) (list []system.SysDictData, err error) {
	if key := dictDataCacheKey(dictType); global.OPS_REDIS != nil {
		if raw, e := global.OPS_REDIS.Get(ctx, key).Bytes(); e == nil && len(raw) > 0 {
			if json.Unmarshal(raw, &list) == nil && list != nil {
				return list, nil
			}
		}
	}
	err = global.OPS_DB.WithContext(ctx).Where("dict_type = ?", dictType).Order(dictDataOrder).Find(&list).Error
	if err == nil && global.OPS_REDIS != nil {
		// 永久缓存 + 写操作主动失效,避免 TTL 窗口内返回陈旧/含已删数据
		if b, e := json.Marshal(list); e == nil {
			global.OPS_REDIS.Set(ctx, dictDataCacheKey(dictType), b, 0)
		}
	}
	return
}

// dictDataCacheKeyPrefix 字典数据缓存 key 前缀,key = dict:data:{dictType}
const dictDataCacheKeyPrefix = "dict:data:"

// dictDataCacheKey 构造单个 dictType 的缓存 key。
func dictDataCacheKey(dictType string) string {
	return dictDataCacheKeyPrefix + dictType
}

// invalidateDictDataCache 失效指定 dictType 集合的数据缓存(字典写操作后调用,防返回陈旧/含已删数据)。
// Redis 不可用或 types 为空时静默跳过(缓存降级,不影响业务)。
func invalidateDictDataCache(ctx context.Context, types []string) {
	if global.OPS_REDIS == nil || len(types) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(types))
	keys := make([]string, 0, len(types))
	for _, t := range types {
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		keys = append(keys, dictDataCacheKey(t))
	}
	if len(keys) > 0 {
		global.OPS_REDIS.Del(ctx, keys...)
	}
}

// RefreshDictCache 清空全部字典数据缓存(对齐前端 DELETE /system/dict/type/refreshCache,手动刷新按钮用)。
func (s *DictDataService) RefreshDictCache(ctx context.Context) error {
	if global.OPS_REDIS == nil {
		return nil
	}
	iter := global.OPS_REDIS.Scan(ctx, 0, dictDataCacheKeyPrefix+"*", -1).Iterator()
	batch := make([]string, 0, 100)
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= 100 {
			global.OPS_REDIS.Del(ctx, batch...)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		global.OPS_REDIS.Del(ctx, batch...)
	}
	return iter.Err()
}

// ExportDictTypeList 按条件导出字典类型(全量,不分页;条件与 GetDictTypeList 一致,加导出上限)。
func (s *DictTypeService) ExportDictTypeList(ctx context.Context, q systemReq.DictTypeSearch) (list []system.SysDictType, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysDictType{})
	if q.DictName != "" {
		db = db.Where("dict_name LIKE ?", "%"+q.DictName+"%")
	}
	if q.DictType != "" {
		db = db.Where("dict_type LIKE ?", "%"+q.DictType+"%")
	}
	err = db.Order("dict_id DESC").Limit(ExportMaxRows).Find(&list).Error
	return
}

// ExportDictDataList 按条件导出字典数据(全量,不分页;条件与 GetDictDataList 一致,加导出上限)。
func (s *DictDataService) ExportDictDataList(ctx context.Context, q systemReq.DictDataSearch) (list []system.SysDictData, err error) {
	db := global.OPS_DB.WithContext(ctx).Model(&system.SysDictData{})
	if q.DictLabel != "" {
		db = db.Where("dict_label LIKE ?", "%"+q.DictLabel+"%")
	}
	if q.DictType != "" {
		db = db.Where("dict_type = ?", q.DictType)
	}
	err = db.Order(dictDataOrder).Limit(ExportMaxRows).Find(&list).Error
	return
}
