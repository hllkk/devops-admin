// aiops-agent 服务器模块采集 agent（slice2：注册+心跳；slice3：指标上报）。
//
// 部署形态：单二进制 + systemd（安装流自动注册 unit，环境变量注入接入配置）。
// 通信全主动外连，agent 侧零监听端口：
//
//	启动 → 携一次性注册 token 调 /server/agent/register 换持久 token（落盘缓存，
//	重启复用）→ 按 server 下发间隔循环心跳；401（token 失效）→ 回退注册流程
//	（注册 token 同步失效则退出等待人工重装）。
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

const (
	// tokenCachePath 持久 token 缓存（重启复用；换 token 仅注册时发生一次）
	tokenCachePath = "/var/lib/aiops-agent/token"
	// defaultInterval 注册响应未带间隔时的兜底心跳周期
	defaultInterval = 30 * time.Second
	// requestTimeout 单请求超时
	requestTimeout = 15 * time.Second
)

type config struct {
	serverURL     string
	registerToken string
}

type registerResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AgentId           string `json:"agentId"`
		Token             string `json:"token"`
		HeartbeatInterval int    `json:"heartbeatInterval"`
	} `json:"data"`
}

type apiResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

var (
	agentVersion = "dev" // 构建期 ldflags 注入（与平台版本一致）
	client       = &http.Client{Timeout: requestTimeout}
)

func main() {
	cfg := loadConfig()
	if cfg.serverURL == "" {
		fatalf("AIOPS_SERVER_URL 未配置")
	}
	for {
		token, interval, err := ensureRegistered(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[aiops-agent] 注册失败: %v, 30s 后重试\n", err)
			time.Sleep(30 * time.Second)
			continue
		}
		if err := heartbeatLoop(cfg, token, interval); err != nil {
			// 401 类失效：回注册流程；其他错误(网络闪断)继续心跳循环
			if errors.Is(err, errUnauthorized) {
				fmt.Fprintf(os.Stderr, "[aiops-agent] token 失效(%v), 重新注册\n", err)
				_ = os.Remove(tokenCachePath)
				continue
			}
			fmt.Fprintf(os.Stderr, "[aiops-agent] 心跳异常: %v, 继续重试\n", err)
			time.Sleep(interval)
			continue
		}
	}
}

// loadConfig 从命令行/环境变量取接入配置（systemd unit 注入 env）。
func loadConfig() config {
	fs := flag.NewFlagSet("aiops-agent", flag.ExitOnError)
	serverURL := fs.String("server", "", "平台接入地址(http://host:port)")
	registerToken := fs.String("register-token", "", "一次性注册 token(首次安装)")
	_ = fs.Parse(os.Args[1:])
	if *serverURL == "" {
		*serverURL = os.Getenv("AIOPS_SERVER_URL")
	}
	if *registerToken == "" {
		*registerToken = os.Getenv("AIOPS_REGISTER_TOKEN")
	}
	return config{serverURL: *serverURL, registerToken: *registerToken}
}

// ensureRegistered 确保 agent 已注册：本地缓存 token 优先，无缓存/失效时走注册流程。
func ensureRegistered(cfg config) (token string, interval time.Duration, err error) {
	if cached, rerr := os.ReadFile(tokenCachePath); rerr == nil && len(bytes.TrimSpace(cached)) > 0 {
		return string(bytes.TrimSpace(cached)), defaultInterval, nil
	}
	if cfg.registerToken == "" {
		return "", 0, errors.New("无缓存 token 且未配置注册 token(需重新安装 agent)")
	}
	body, _ := json.Marshal(map[string]string{
		"token":        cfg.registerToken,
		"hostname":     hostname(),
		"arch":         runtime.GOARCH,
		"os":           runtime.GOOS,
		"agentVersion": agentVersion,
	})
	resp, err := postJSON(cfg.serverURL+"/server/agent/register", "", body)
	if err != nil {
		return "", 0, err
	}
	var rr registerResponse
	if err := json.Unmarshal(resp, &rr); err != nil {
		return "", 0, fmt.Errorf("注册响应解析失败: %w", err)
	}
	if rr.Code != "0000" {
		return "", 0, fmt.Errorf("注册被拒绝: %s", rr.Msg)
	}
	if err := saveToken(rr.Data.Token); err != nil {
		fmt.Fprintf(os.Stderr, "[aiops-agent] token 缓存失败(重启将重新注册): %v\n", err)
	}
	if rr.Data.HeartbeatInterval > 0 {
		return rr.Data.Token, time.Duration(rr.Data.HeartbeatInterval) * time.Second, nil
	}
	return rr.Data.Token, defaultInterval, nil
}

// heartbeatLoop 心跳主循环（直到 token 失效或不可恢复错误;每帧带轻量指标快照）。
func heartbeatLoop(cfg config, token string, interval time.Duration) error {
	for {
		body, _ := json.Marshal(map[string]any{
			"agentVersion": agentVersion,
			"metrics":      collectorInst.collect(),
		})
		resp, err := postJSON(cfg.serverURL+"/server/agent/heartbeat", token, body)
		if err != nil {
			// 401 语义：服务端明确拒绝（token 失效）→ 上抛触发重注册
			var apiErr *apiError
			if errors.As(err, &apiErr) && apiErr.code == 401 {
				return fmt.Errorf("%w: %s", errUnauthorized, apiErr.msg)
			}
			// 网络类错误：本循环内重试
			fmt.Fprintf(os.Stderr, "[aiops-agent] 心跳请求失败: %v\n", err)
		} else if err := checkOK(resp); err != nil {
			var apiErr *apiError
			if errors.As(err, &apiErr) && apiErr.code == 401 {
				return fmt.Errorf("%w: %s", errUnauthorized, apiErr.msg)
			}
			return err
		}
		time.Sleep(interval)
	}
}

// errUnauthorized 401 语义（触发重注册）。
var errUnauthorized = errors.New("unauthorized")

type apiError struct {
	code int
	msg  string
}

func (e *apiError) Error() string { return fmt.Sprintf("api %d: %s", e.code, e.msg) }

// postJSON 发请求；非 2xx 返回 apiError（NoAuth 401 场景）。
func postJSON(url, token string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &apiError{code: resp.StatusCode, msg: string(raw)}
	}
	return raw, nil
}

// checkOK 业务码校验（code != 0000 视为失败；401 业务拒绝需上抛）。
func checkOK(raw []byte) error {
	var ar apiResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		return fmt.Errorf("响应解析失败: %w", err)
	}
	if ar.Code != "0000" {
		return &apiError{code: 500, msg: ar.Msg}
	}
	return nil
}

// saveToken 持久 token 落盘（目录不存在则建）。
func saveToken(token string) error {
	if err := os.MkdirAll("/var/lib/aiops-agent", 0o700); err != nil {
		return err
	}
	return os.WriteFile(tokenCachePath, []byte(token), 0o600)
}

// hostname 目标机主机名。
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

// fatalf 致命错误退出。
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[aiops-agent] "+format+"\n", args...)
	os.Exit(1)
}
