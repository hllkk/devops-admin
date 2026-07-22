package system

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/utils/request"
)

type AiService struct{}

// ollamaChatRequest Ollama /api/chat 请求体
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatResponse Ollama /api/chat 响应体
type ollamaChatResponse struct {
	Model   string        `json:"model"`
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

const errorAnalysisPrompt = `你是一名资深运维专家。请分析以下系统错误日志，按结构给出回复：

## 错误来源
%s

## 错误详情
%s

请给出：
1. **错误原因**：根因分析，发生了什么
2. **解决方案**：具体可执行的操作步骤
3. **预防措施**：如何避免再次发生`

// AnalyzeError 使用 Ollama 本地模型分析错误日志并返回解决方案
func (a *AiService) AnalyzeError(ctx context.Context, form, info string) (string, error) {
	cfg := global.OPS_CONFIG.Ai.Ollama
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		return "", fmt.Errorf("ollama base-url 未配置")
	}
	if cfg.Model == "" {
		return "", fmt.Errorf("ollama model 未配置")
	}

	prompt := fmt.Sprintf(errorAnalysisPrompt, form, info)
	reqBody := ollamaChatRequest{
		Model: cfg.Model,
		Messages: []ollamaMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	timeout := 120 * time.Second
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	res, err := request.HttpRequestWithContextAndTimeout(
		ctx,
		baseURL+"/api/chat",
		"POST",
		nil,
		nil,
		reqBody,
		timeout,
	)
	if err != nil {
		return "", fmt.Errorf("调用 Ollama 失败: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("读取 Ollama 响应失败: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("Ollama 返回非 2xx: status=%d body=%s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var chatResp ollamaChatResponse
	if err = json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析 Ollama 响应失败: status=%d body=%s err=%w", res.StatusCode, previewLLMBody(body), err)
	}

	return chatResp.Message.Content, nil
}

// previewLLMBody 截断响应体用于错误日志(避免日志膨胀)
func previewLLMBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return "<empty>"
	}
	runes := []rune(text)
	if len(runes) > 300 {
		return string(runes[:300]) + "..."
	}
	return text
}
