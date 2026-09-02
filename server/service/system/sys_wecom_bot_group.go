package system

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/system"
)

// WecomBotGroupService 企微群机器人群登记(sys_wecom_bot_group)维护:
// 设置页群表格 新增/列表/删除(暂无编辑,录错删除重录);发送链路按 id 查询本表。
type WecomBotGroupService struct{}

// List 群列表(创建时间倒序,最新录入在前)。
func (s *WecomBotGroupService) List(ctx context.Context) ([]system.SysWecomBotGroup, error) {
	var groups []system.SysWecomBotGroup
	err := global.OPS_DB.WithContext(ctx).
		Order("create_time DESC").Find(&groups).Error
	return groups, err
}

// Create 新增群(群名+webhook 必填,webhook 须为企微机器人地址)。
func (s *WecomBotGroupService) Create(ctx context.Context, groupName, webhookUrl string) (system.SysWecomBotGroup, error) {
	groupName = strings.TrimSpace(groupName)
	webhookUrl = strings.TrimSpace(webhookUrl)
	if groupName == "" {
		return system.SysWecomBotGroup{}, errors.New("群聊名称不能为空")
	}
	if !strings.HasPrefix(webhookUrl, "https://qyapi.weixin.qq.com/cgi-bin/webhook/send") {
		return system.SysWecomBotGroup{}, errors.New("webhook 地址非法(须为 https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx)")
	}
	g := system.SysWecomBotGroup{GroupName: groupName, WebhookUrl: webhookUrl}
	if err := global.OPS_DB.WithContext(ctx).Create(&g).Error; err != nil {
		return system.SysWecomBotGroup{}, err
	}
	return g, nil
}

// Delete 按主键删除群(软删,发送链路按 id 查询自动排除;关联晨报策略里的失效 id 发送时跳过)。
func (s *WecomBotGroupService) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("群 ID 不能为空")
	}
	if err := global.OPS_DB.WithContext(ctx).
		Delete(&system.SysWecomBotGroup{}, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("目标群不存在或已删除")
		}
		return err
	}
	return nil
}
