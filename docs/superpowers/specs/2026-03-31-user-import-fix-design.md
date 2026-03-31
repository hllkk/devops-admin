---
name: 用户导入功能修复
description: 修复用户导入功能的字段映射问题和软删除用户唯一键冲突问题
type: project
created: 2026-03-31
---

# 用户导入功能修复设计文档

## 问题概述

用户导入功能存在以下问题：
1. 未勾选更新时：唯一键冲突 - 软删除的用户记录仍在索引中
2. 勾选更新时：字段名错误 (`nick_name`) + 唯一键冲突

## 问题根因分析

### 字段映射不一致

| 模型字段 | GORM默认映射 | 代码使用 | 结果 |
|---------|-------------|---------|------|
| `Nickname` | `nickname` | `nick_name` | 字段不存在错误 |
| `Phone` | `phone` | `user_phone` | 字段不存在错误 |

代码中使用 `nick_name` 和 `user_phone` 作为数据库列名，但模型未显式指定 `column` 标签，导致 GORM 使用字段名的蛇形转换（`nickname` → `nickname`, `Phone` → `phone`）。

### 软删除唯一键冲突

- `OPS_MODEL` 包含 `DeletedAt gorm.DeletedAt`（软删除字段）
- `SysUser` 模型定义唯一索引：`UserName string `gorm:"uniqueIndex;..."`
- 软删除后记录仍在唯一索引中
- 导入查询默认过滤软删除记录，无法检测已删除用户
- 创建新用户时触发唯一键冲突

## 解决方案

采用方案A：修复字段映射 + 改进导入逻辑。

### 修改文件

1. `backend/model/system/sys_user.go` - 模型字段映射
2. `backend/service/system/sys_user.go` - 导入逻辑

## 详细设计

### 1. 模型字段映射修复

修改 `backend/model/system/sys_user.go`：

```go
// 修改前（第26行）
Nickname string `gorm:"size:32;default:系统用户;comment:用户昵称" json:"nickName"`
Phone    string `gorm:"type:char(11);comment:手机号" json:"userPhone"`

// 修改后
Nickname string `gorm:"column:nick_name;size:32;default:系统用户;comment:用户昵称" json:"nickName"`
Phone    string `gorm:"column:user_phone;type:char(11);comment:手机号" json:"userPhone"`
```

添加 `column:nick_name` 和 `column:user_phone` 标签，显式指定数据库列名。

### 2. 导入逻辑改进

修改 `backend/service/system/sys_user.go` 的 `ImportUserList` 函数（约第426-547行）：

#### 2.1 使用 Unscoped() 查询包含软删除用户

```go
// 修改前（约第470行）
err := global.OPS_DB.Where("user_name = ?", userName).First(&existingUser).Error

// 修改后
err := global.OPS_DB.Unscoped().Where("user_name = ?", userName).First(&existingUser).Error
```

#### 2.2 处理软删除用户场景

在查询结果处理后，新增软删除用户判断逻辑：

```go
// 检查用户是否被软删除
if err == nil && existingUser.DeletedAt.Valid {
    if updateSupport {
        // 恢复软删除用户（清除 deleted_at）
        existingUser.DeletedAt = gorm.DeletedAt{}
        if err := global.OPS_DB.Model(&existingUser).Update("deleted_at", nil).Error; err != nil {
            errorMsgs = append(errorMsgs, fmt.Sprintf("第%d行恢复用户失败: %v", i+1, err))
            result.FailCount++
            continue
        }
        // 继续执行更新逻辑...
    } else {
        // 未勾选更新，提示用户已被删除
        errorMsgs = append(errorMsgs, fmt.Sprintf("第%d行用户名'%s'已被删除，勾选更新可恢复", i+1, userName))
        result.FailCount++
        continue
    }
}
```

#### 2.3 更新逻辑调整

更新已存在用户时，需要区分正常用户和刚恢复的软删除用户：

```go
if err == nil {
    // 用户存在（包括刚恢复的软删除用户）
    if !updateSupport && !existingUser.DeletedAt.Valid {
        // 正常用户且未勾选更新，跳过
        errorMsgs = append(errorMsgs, fmt.Sprintf("第%d行用户名'%s'已存在", i+1, userName))
        result.FailCount++
        continue
    }

    // 更新用户（勾选更新或刚恢复的软删除用户）
    updates := map[string]interface{}{
        "nick_name": nickName,
        "gender":    gender,
        "status":    status,
        "email":     email,
        "user_phone": phone,
    }
    if err := global.OPS_DB.Model(&existingUser).Updates(updates).Error; err != nil {
        errorMsgs = append(errorMsgs, fmt.Sprintf("第%d行更新失败: %v", i+1, err))
        result.FailCount++
        continue
    }
    result.SuccessCount++
}
```

## 测试验证

修复后需验证以下场景：

1. **未勾选更新 + 新用户**：正常创建
2. **未勾选更新 + 已存在正常用户**：跳过，提示已存在
3. **未勾选更新 + 软删除用户**：跳过，提示已删除可恢复
4. **勾选更新 + 已存在正常用户**：正常更新
5. **勾选更新 + 软删除用户**：恢复并更新
6. **勾选更新 + 新用户**：正常创建

## 风险评估

- **低风险**：仅修改模型字段映射和导入逻辑，不影响其他业务逻辑
- **数据库兼容**：无需修改数据库结构，现有数据兼容
- **回滚简单**：可通过 Git 回滚代码变更