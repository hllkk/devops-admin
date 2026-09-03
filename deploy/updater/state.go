package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// 升级状态机：idle → downloading → verifying → unpacking → installing → success | failed
// （downloading/unpacking 由 daemon 写，installing 及终态由 install job 写；
//
//	状态文件挂在安装目录，是 daemon 与 job 的共享真源，容器重建不丢）
const (
	StateIdle        = "idle"
	StateDownloading = "downloading"
	StateVerifying   = "verifying"
	StateUnpacking   = "unpacking"
	StateInstalling  = "installing"
	StateSuccess     = "success"
	StateFailed      = "failed"
)

// state 活跃态：处于这些状态时拒绝新的升级请求（防并发升级）
func (s *UpgradeState) active() bool {
	switch s.State {
	case StateDownloading, StateVerifying, StateUnpacking, StateInstalling:
		return true
	}
	return false
}

// UpgradeState 升级状态（落盘 stack/upgrade-state.json）
type UpgradeState struct {
	State      string `json:"state"`             // 状态机当前态
	Progress   int    `json:"progress"`          // 0-100（downloading 阶段为下载百分比）
	Message    string `json:"message,omitempty"` // 进展/错误说明
	Version    string `json:"version,omitempty"` // 目标版本（终态时为已装版本）
	UpdateTime string `json:"updateTime"`        // 本条状态写入时间（RFC3339）
}

// statePath 状态文件路径（STACK_DIR 由 compose 注入，缺省 /opt/stack）
func statePath() string {
	return filepath.Join(stackDir(), "upgrade-state.json")
}

func stackDir() string {
	dir := os.Getenv("STACK_DIR")
	if dir == "" {
		dir = "/opt/stack"
	}
	return dir
}

// loadState 读状态文件；不存在视为 idle（首次部署/从未升级）
func loadState() (*UpgradeState, error) {
	data, err := os.ReadFile(statePath())
	if os.IsNotExist(err) {
		return &UpgradeState{State: StateIdle}, nil
	}
	if err != nil {
		return nil, err
	}
	var s UpgradeState
	if err := json.Unmarshal(data, &s); err != nil {
		// 状态文件损坏：视为 failed 而非 idle，避免误发起新升级掩盖问题
		return &UpgradeState{State: StateFailed, Message: "状态文件损坏: " + err.Error(), UpdateTime: nowRFC3339()}, nil
	}
	return &s, nil
}

// saveState 写状态文件（0644，daemon 与 install job 共同读写）
func saveState(s *UpgradeState) error {
	s.UpdateTime = nowRFC3339()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), data, 0o644)
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}
