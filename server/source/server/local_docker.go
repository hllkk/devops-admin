package server

import (
	"context"
	"encoding/json"

	serverModel "github.com/hllkk/devops-admin/server/model/server"
	"github.com/hllkk/devops-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// 服务器模块种子：本机 Docker 内置资产（「先纳管本机」决策的开箱即用部分）。
// 本机 Docker 是平台自有确定性资源（部署形态决定，sock 路径固定），不让用户手动录入；
// 采集能力(slice6)上线前它是占位资产，monitor/agent 状态由采集链路自然回写。
//
// 幂等语义：按 channel_config 的 internalKey 查重（LIKE 子串匹配，跨库通用），
// **含软删行**——用户显式删除内置资产后重启不复活（软删行占位）；
// 用户改名不影响幂等（internalKey 在行数据里，与 asset_name 解耦）。

// 服务器模块初始化器排 system 链之后的独立区段（gateway 用 +200，server 用 +300 避撞）。
const initOrderServerSeed = system.InitOrderSystem + 300

type initServerSeed struct{}

// auto run
func init() {
	system.RegisterInit(initOrderServerSeed, &initServerSeed{})
}

func (i *initServerSeed) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(
		&serverModel.Asset{},
		&serverModel.Credential{},
	)
}

func (i *initServerSeed) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&serverModel.Asset{})
}

func (i *initServerSeed) InitializerName() string {
	return serverModel.Asset{}.TableName()
}

func (i *initServerSeed) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	if err := EnsureBuiltinLocalDocker(db); err != nil {
		return ctx, err
	}
	return ctx, nil
}

func (i *initServerSeed) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	var count int64
	db.Model(&serverModel.Asset{}).Count(&count)
	return count > 0
}

// builtinDockerInternalKey 本机 Docker 内置资产的幂等标记（channel_config 内键值对）。
const builtinDockerInternalKey = "builtin-docker-local"

// builtinDockerSocket 本机 Docker daemon socket 路径（linux 标准路径；slice6 采集用）。
const builtinDockerSocket = "/var/run/docker.sock"

// EnsureBuiltinLocalDocker 幂等登记「本机 Docker」内置资产。
// 重启路径(initialize/gorm.go)与 /initdb 路径(initializer 链)双路调用；
// 失败仅由调用方记日志不阻断启动。
func EnsureBuiltinLocalDocker(db *gorm.DB) error {
	// 含软删查重：软删行占位（用户删除后不复活），改名不影响（internalKey 在行数据里）。
	// LIKE 用 internalKey 值本身做特异子串（PG jsonb 读出是规范化文本 "key": "value" 带空格、
	// 键序还会重排，Go json.Marshal 无空格——不能按完整键值对匹配；值串足够特异，跨库通用）。
	var cnt int64
	if err := db.Unscoped().Model(&serverModel.Asset{}).
		Where("channel_config LIKE ?", "%"+builtinDockerInternalKey+"%").
		Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	channel, err := json.Marshal(map[string]string{
		"internalKey": builtinDockerInternalKey,
		"socket":      builtinDockerSocket,
	})
	if err != nil {
		return errors.Wrap(err, "marshal builtin docker channel config failed")
	}
	// manage_ip 留空：本机资产走 sock 直连，IP 无意义；宿主信息由 slice6 采集回填。
	// 凭据不关联：sock 本身即凭证。
	asset := serverModel.Asset{
		AssetName:     "本机 Docker",
		AssetType:     serverModel.AssetTypeDockerHost,
		OsType:        "linux",
		IsActive:      true,
		ChannelConfig: datatypes.JSON(channel),
	}
	if err := db.Create(&asset).Error; err != nil {
		return errors.Wrap(err, "create builtin local docker asset failed")
	}
	return nil
}
