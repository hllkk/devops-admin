package gateway

import (
	"reflect"
	"testing"
)

// toLitellmModels 纯函数单测：空授权投影为 LiteLLM deny-all 哨兵，防"空 models=全量放行"。

func TestToLitellmModels_EmptyBecomesDenyAllSentinel(t *testing.T) {
	if got := toLitellmModels(nil); !reflect.DeepEqual(got, []string{litellmNoDefaultModels}) {
		t.Errorf("nil 授权应投影为哨兵 deny-all: %v", got)
	}
	if got := toLitellmModels([]string{}); !reflect.DeepEqual(got, []string{litellmNoDefaultModels}) {
		t.Errorf("空列表应投影为哨兵 deny-all: %v", got)
	}
}

func TestToLitellmModels_NormalPassThrough(t *testing.T) {
	in := []string{"qwen-max", "gpt-4o"}
	if got := toLitellmModels(in); !reflect.DeepEqual(got, in) {
		t.Errorf("非空授权应原样下发: %v, 期望 %v", got, in)
	}
}

func TestToLitellmModels_SentinelAndEmptyFiltered(t *testing.T) {
	// 脏数据防御：哨兵/空串混入不透传，其余模型保序保留
	got := toLitellmModels([]string{litellmNoDefaultModels, "", "claude-3"})
	if want := []string{"claude-3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("应过滤哨兵与空串: %v, 期望 %v", got, want)
	}
	// 过滤后为空同样落哨兵，保证恒非空(下发两路的判定都依赖)
	if got := toLitellmModels([]string{litellmNoDefaultModels, ""}); !reflect.DeepEqual(got, []string{litellmNoDefaultModels}) {
		t.Errorf("过滤后为空应落哨兵: %v", got)
	}
}
