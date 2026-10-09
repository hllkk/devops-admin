package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayReq "github.com/hllkk/devops-admin/server/model/gateway/request"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/utils/litellm"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RouterSettingsService struct{}

// Get 读取全局路由策略(单行 id=1;不存在则按默认 FirstOrCreate)。
func (s *RouterSettingsService) Get(ctx context.Context) (gatewayResp.RouterSettingsView, error) {
	rs, err := s.getOrCreate(ctx)
	if err != nil {
		return gatewayResp.RouterSettingsView{}, err
	}
	return s.toView(rs), nil
}

// getOrCreate 取单行(id=1),不存在则建默认行(对齐 sys_security_config 单例模式)。
func (s *RouterSettingsService) getOrCreate(ctx context.Context) (gateway.RouterSettings, error) {
	if global.OPS_DB == nil {
		// 数据库未初始化时返回默认配置并带错误,调用方据此处理
		return gateway.DefaultRouterSettings(), errors.New("数据库未初始化")
	}
	var rs gateway.RouterSettings
	err := global.OPS_DB.WithContext(ctx).Where("id = ?", 1).First(&rs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rs = gateway.DefaultRouterSettings()
		rs.ID = 1
		if err = global.OPS_DB.WithContext(ctx).Create(&rs).Error; err != nil {
			return rs, err
		}
		return rs, nil
	}
	return rs, err
}

// Update 更新全局路由策略(整体覆盖;事务:落库 + 同步 LiteLLM /router/settings 热更新
// + 全局并发上限经 /config/field/update 热生效)。
func (s *RouterSettingsService) Update(ctx context.Context, req gatewayReq.RouterSettingsUpdate) (gatewayResp.RouterSettingsView, error) {
	prev, err := s.getOrCreate(ctx)
	if err != nil {
		return gatewayResp.RouterSettingsView{}, err
	}

	strategy := req.RoutingStrategy
	if strategy == "" {
		strategy = gateway.RoutingStrategySimpleShuffle
	}

	// fallbacks 对象数组 → DB JSON(空给 [],前端空数组而非 null)
	fbJSON := []byte("[]")
	if req.Fallbacks != nil {
		fbJSON, err = json.Marshal(req.Fallbacks)
		if err != nil {
			return gatewayResp.RouterSettingsView{}, fmt.Errorf("序列化降级链失败: %w", err)
		}
	}

	// config map → DB JSON(空给 {})
	cfgJSON := []byte("{}")
	if req.Config != nil {
		cfgJSON, err = json.Marshal(req.Config)
		if err != nil {
			return gatewayResp.RouterSettingsView{}, fmt.Errorf("序列化扩展配置失败: %w", err)
		}
	}

	rs := gateway.RouterSettings{
		RoutingStrategy: strategy,
		Fallbacks:       datatypes.JSON(fbJSON),
		AllowedFails:    req.AllowedFails,
		CooldownTime:    req.CooldownTime,
		NumRetries:      req.NumRetries,
		Timeout:         req.Timeout,
		GlobalMaxParallel: req.GlobalMaxParallel,
		Config:          datatypes.JSON(cfgJSON),
	}
	rs.OPS_MODEL = prev.OPS_MODEL // 保留基座(ID/时间戳)

	cli := litellm.Default()
	err = global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&rs).Error; err != nil {
			return err
		}
		// 投影同步:LiteLLM 未配置(sync-enabled=false)时静默跳过,仅落库
		if cli != nil {
			if err := cli.UpdateRouterSettings(ctx, s.toLitellm(rs)); err != nil {
				return fmt.Errorf("同步 LiteLLM 路由策略失败: %w", err)
			}
			// 全局并发独立投影(general_settings 非 router_settings):nil=删字段恢复无限制
			if err := s.syncGlobalConcurrency(ctx, cli, rs.GlobalMaxParallel); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return gatewayResp.RouterSettingsView{}, err
	}
	return s.toView(rs), nil
}

// toView DB 行 → 出网视图(fallbacks/config JSON 反序列化;空给空容器而非 null)。
func (s *RouterSettingsService) toView(rs gateway.RouterSettings) gatewayResp.RouterSettingsView {
	view := gatewayResp.RouterSettingsView{
		RoutingStrategy: rs.RoutingStrategy,
		AllowedFails:    rs.AllowedFails,
		CooldownTime:    rs.CooldownTime,
		NumRetries:      rs.NumRetries,
		Timeout:         rs.Timeout,
		GlobalMaxParallel: rs.GlobalMaxParallel,
		Fallbacks:       []gateway.FallbackItem{},
		Config:          map[string]any{},
	}
	if len(rs.Fallbacks) > 0 {
		_ = json.Unmarshal(rs.Fallbacks, &view.Fallbacks)
	}
	if view.Fallbacks == nil {
		view.Fallbacks = []gateway.FallbackItem{}
	}
	if len(rs.Config) > 0 {
		_ = json.Unmarshal(rs.Config, &view.Config)
	}
	if view.Config == nil {
		view.Config = map[string]any{}
	}
	return view
}

// syncGlobalConcurrency 全局并发上限投影同步(nil → 删 LiteLLM 字段恢复无限制)。
// 前置约束:LiteLLM yaml 不得声明该字段(config-owned 会让 API 写入被 400 拒绝)。
func (s *RouterSettingsService) syncGlobalConcurrency(ctx context.Context, cli *litellm.Client, v *int) error {
	if v == nil {
		if err := cli.DeleteGlobalConcurrency(ctx); err != nil {
			return fmt.Errorf("清除 LiteLLM 全局并发上限失败: %w", err)
		}
		return nil
	}
	if err := cli.UpdateGlobalConcurrency(ctx, *v); err != nil {
		return fmt.Errorf("同步 LiteLLM 全局并发上限失败: %w", err)
	}
	return nil
}

// toLitellm 平台格式 → LiteLLM 蛇形投影:
// fallbacks [{model,fallbacks}] → [[model,[fallbacks]]]；驼峰键 → 蛇形键。
func (s *RouterSettingsService) toLitellm(rs gateway.RouterSettings) map[string]any {
	var items []gateway.FallbackItem
	if len(rs.Fallbacks) > 0 {
		_ = json.Unmarshal(rs.Fallbacks, &items)
	}
	fbLitellm := make([][]any, 0, len(items))
	for _, f := range items {
		fbs := f.Fallbacks
		if fbs == nil {
			fbs = []string{}
		}
		fbLitellm = append(fbLitellm, []any{f.Model, fbs})
	}
	settings := map[string]any{
		"routing_strategy": rs.RoutingStrategy,
		"allowed_fails":    rs.AllowedFails,
		"cooldown_time":   rs.CooldownTime,
		"num_retries":     rs.NumRetries,
		"timeout":         rs.Timeout,
	}
	if len(fbLitellm) > 0 {
		settings["fallbacks"] = fbLitellm
	}
	if len(rs.Config) > 0 {
		var cfg map[string]any
		if err := json.Unmarshal(rs.Config, &cfg); err == nil && len(cfg) > 0 {
			settings["config"] = cfg
		}
	}
	return settings
}
