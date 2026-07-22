package config

// Ai AI 服务配置,支持 Ollama 本地部署或外部 ai-path 代理两种后端
type Ai struct {
	Provider string   `mapstructure:"provider" json:"provider" yaml:"provider"` // ollama | external(走 autocode.ai-path)
	Ollama   OllamaAi `mapstructure:"ollama" json:"ollama" yaml:"ollama"`
}

// OllamaAi Ollama 本地模型配置
type OllamaAi struct {
	BaseURL string `mapstructure:"base-url" json:"base-url" yaml:"base-url"` // 如 http://localhost:11434
	Model   string `mapstructure:"model" json:"model" yaml:"model"`          // 如 qwen2.5:7b
	Timeout int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`    // 请求超时(秒),默认 120
}
