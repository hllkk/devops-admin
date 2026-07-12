package config

// Snowflake 雪花算法 ID 生成器配置
type Snowflake struct {
	Node  int64  `mapstructure:"node" json:"node" yaml:"node"`    // worker id（机器节点标识，0~1023，多实例时唯一）
	Epoch string `mapstructure:"epoch" json:"epoch" yaml:"epoch"` // 起始纪元（RFC3339，如 2024-01-01T00:00:00Z）
}
