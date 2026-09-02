package gateway

import (
	"encoding/json"
	"testing"

	"gorm.io/datatypes"
)

func TestIsSpeechRecognitionModel(t *testing.T) {
	cases := []struct {
		caps  []string
		want  bool
		desc  string
	}{
		{[]string{"语音识别"}, true, "中文标签"},
		{[]string{"推理", "图像", "语音识别", "长上下文"}, true, "混合标签命中"},
		{[]string{"ASR"}, true, "英文大写(大小写不敏感)"},
		{[]string{"speech-to-text"}, true, "speech 关键词"},
		{[]string{"推理", "工具调用"}, false, "普通对话能力不命中"},
		{nil, false, "空能力"},
	}
	for _, c := range cases {
		raw, _ := json.Marshal(c.caps)
		if got := isSpeechRecognitionModel(datatypes.JSON(raw)); got != c.want {
			t.Fatalf("isSpeechRecognitionModel(%v)(%s) = %v, 期望 %v", c.caps, c.desc, got, c.want)
		}
	}

	// 非法 JSON 不误判
	if isSpeechRecognitionModel(datatypes.JSON("not-json")) {
		t.Fatal("非法 JSON 应返回 false")
	}
}
