# 模型管理页细小 BUG 修复（6 项）

- 日期：2026-09-02
- 状态：已实现（探测体经真实上游验证通过）

## 需求

用户报模型管理页 6 个细小 BUG：
1. 模型名称过长需悬停展示
2. 模型 ID 过长导致左侧条目高度不一致
3. 新增供应商后必须刷新页面，渠道对话框才能看到
4. 左侧列表要用 Naive UI 滚动条，不用原生滚动条
5. 多模态模型联通性测试大多报错（MiMo-v2.5-ASR 语音识别模型必失败，纯对话 MiMo-v2.5 正常）
6. 无渠道的模型不应展示「发布设置」按钮

## 实现口径

- **BUG1/2**（`models/model/index.vue`）：名称/模型 ID 两行改 `NEllipsis`（溢出才出 tooltip，天然满足悬停+定高）；第二行拆 `NEllipsis 弹性截断 modelKey` + `shrink-0 计数后缀`，条目高度恒定。
- **BUG4**：左侧列表 `overflow-y-auto + max-height` 换 `NScrollbar style="max-height: calc(100vh - 300px)"`（sticky 分组头在 NScrollbar 内正常工作）。
- **BUG3**（`deployment-operate-dialog.vue`）：`loadProviders/loadCredentials` 从 `onMounted`（仅挂载时一次）挪到 `watch(visible)` 打开时拉取；新增分支补 `loadCredentials(null)`。
- **BUG6**（`model-detail-panel.vue`）：发布设置按钮加 `v-if="deploymentList.length"`。
- **BUG5**（`server/service/gateway/deployment.go`）：语音识别模型不可用纯文本 ping 探测（上游业务校验要求消息含 `input_audio` part，400 拒绝）。新增 `isSpeechRecognitionModel`（capabilities 关键词启发式：语音识别/语音转/asr/stt/transcri/speech）→ ASR 形态探测体改带 `input_audio`（data URL 极小静音 wav 常量 `asrProbeAudio`，OpenAI 兼容格式，max_tokens 16）；ASR 且上游 400 时错误信息追加「语音识别部署须走 OpenAI 兼容端点」提示。新增 `deployment_test.go` 单测。

## 关键决策

- **ASR 识别用 capabilities 关键词启发式**而非新增模型类别：类别体系（chat/embedding/rerank）重构不在本次范围；用户标签自由输入，中英文常见写法覆盖即可。
- **实测结论（LiteLLM 1.99 + 小米 token-plan）**：MiMo-ASR 走 OpenAI Chat Completions 兼容格式，`input_audio.data` 为 data URL；**LiteLLM 的 anthropic 协议通道会把 audio part 丢弃**（上游报 found: 0），anthropic 格式 `/v1/messages` 直通也 500。即：**语音识别部署必须绑 OpenAI 兼容格式凭证（api_base=`https://token-plan-cn.xiaomimimo.com/v1`）**。探测体在该通道实测 200（识别出静音）。
- 用户现有小米凭证是 anthropic 端点，ASR 部署需另建 openai 格式凭证（同一 API Key）并在渠道上绑定，改后端代码解决不了配置问题——失败信息已带指引。

相关：[[ai-gateway-implementation-notes]]
