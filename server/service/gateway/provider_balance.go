package gateway

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hllkk/devops-admin/server/global"
	"github.com/hllkk/devops-admin/server/model/gateway"
	gatewayResp "github.com/hllkk/devops-admin/server/model/gateway/response"
	"github.com/hllkk/devops-admin/server/utils/crypto"
	"github.com/hllkk/devops-admin/server/utils/logger"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// ProviderBalanceService 套餐余量旁路（设计稿 aiDoc/modules/ai-gateway-billing-integration.md「套餐真实余量旁路」）。
// 只读快照，不进 calcCosts、不触发预算停用、不并入 cost_summary_daily；
// 标价成本走部署层 model_info 四键（Credits 折 ¥ 人工维护），两口径互不并。
type ProviderBalanceService struct{}

const (
	bailianAPIVersion = "2026-02-10"                              // ModelStudio OpenAPI 版本
	bailianSeatsPath  = "/tokenplan/subscription/seat-detail"     // GetSubscriptionSeatDetails
	bailianPkgPath    = "/tokenplan/subscription/shared-packages" // ListSubscriptionSharedPackages(门户元数据确认复数)
	balancePageSize   = 100                                       // 采集翻页大小
	balanceMaxPages   = 10                                        // 翻页保护上限(坐席/共享包各最多1000条)
)

// GetProviderBalances 供应商余量明细（快照现状 + 汇总）。
func (s *ProviderBalanceService) GetProviderBalances(ctx context.Context, providerId int64) ([]gateway.ProviderBalance, gatewayResp.ProviderBalanceSummary, error) {
	summary, err := s.buildSummary(ctx, providerId)
	if err != nil {
		return nil, summary, err
	}
	var items []gateway.ProviderBalance
	if err := global.OPS_DB.WithContext(ctx).
		Where("provider_id = ?", providerId).
		Order("item_type ASC, used_value DESC").Find(&items).Error; err != nil {
		return nil, summary, err
	}
	return items, summary, nil
}

// buildSummary 汇总单供应商余量（快照按条目 SUM，无快照返回零值汇总）。
func (s *ProviderBalanceService) buildSummary(ctx context.Context, providerId int64) (gatewayResp.ProviderBalanceSummary, error) {
	p, err := (&ProviderService{}).GetProvider(ctx, providerId)
	if err != nil {
		return gatewayResp.ProviderBalanceSummary{}, err
	}
	return s.buildSummaryFromProvider(ctx, p)
}

// buildSummaryFromProvider 基于已取供应商对象汇总（避免重复查询）。
func (s *ProviderBalanceService) buildSummaryFromProvider(ctx context.Context, p gateway.Provider) (gatewayResp.ProviderBalanceSummary, error) {
	summary := gatewayResp.ProviderBalanceSummary{
		ProviderId:   p.ProviderId,
		ProviderName: p.Name,
		PlanLabel:    gateway.BalanceSyncProviderTypes[p.ProviderType],
	}
	if summary.PlanLabel == "" {
		summary.PlanLabel = p.ProviderType
	}
	type agg struct {
		Total   float64
		Used    float64
		Surplus float64
		Seats   int
		Pkgs    int
	}
	var a agg
	if err := global.OPS_DB.WithContext(ctx).Model(&gateway.ProviderBalance{}).
		Where("provider_id = ?", p.ProviderId).
		Select("COALESCE(SUM(total_value),0) AS total, COALESCE(SUM(used_value),0) AS used, COALESCE(SUM(surplus_value),0) AS surplus, " +
			"COUNT(*) FILTER (WHERE item_type = 'seat') AS seats, COUNT(*) FILTER (WHERE item_type = 'shared_package') AS pkgs").
		Scan(&a).Error; err != nil {
		return summary, err
	}
	summary.TotalValue, summary.UsedValue, summary.SurplusValue = a.Total, a.Used, a.Surplus
	summary.SeatCount, summary.PackageCount = a.Seats, a.Pkgs
	var last gateway.ProviderBalance
	if err := global.OPS_DB.WithContext(ctx).Where("provider_id = ?", p.ProviderId).
		Order("synced_at DESC").Limit(1).Select("synced_at").Scan(&last).Error; err == nil && !last.SyncedAt.IsZero() {
		t := last.SyncedAt
		summary.SyncedAt = &t
	}
	return summary, nil
}

// GetBalanceSummaryAll 跨供应商余量汇总（看板汇总卡，最新快照口径）。
func (s *ProviderBalanceService) GetBalanceSummaryAll(ctx context.Context) ([]gatewayResp.ProviderBalanceSummary, error) {
	var providers []gateway.Provider
	if err := global.OPS_DB.WithContext(ctx).
		Where("provider_type IN ?", balanceSyncProviderTypes()).Find(&providers).Error; err != nil {
		return nil, err
	}
	out := make([]gatewayResp.ProviderBalanceSummary, 0, len(providers))
	for _, p := range providers {
		summary, err := s.buildSummaryFromProvider(ctx, p)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	return out, nil
}

// balanceSyncProviderTypes 白名单 key 列表。
func balanceSyncProviderTypes() []string {
	types := make([]string, 0, len(gateway.BalanceSyncProviderTypes))
	for t := range gateway.BalanceSyncProviderTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// GetBalanceConfig 读采集配置明文（内部使用：保存合并/同步）。
func (s *ProviderBalanceService) GetBalanceConfig(ctx context.Context, providerId int64) (gateway.BalanceSyncConfig, error) {
	p, err := (&ProviderService{}).GetProvider(ctx, providerId)
	if err != nil {
		return gateway.BalanceSyncConfig{}, err
	}
	return decryptBalanceConfig(p.BalanceSyncConfig)
}

// GetBalanceConfigView 读采集配置出网视图（AK/SK 掩码）。
func (s *ProviderBalanceService) GetBalanceConfigView(ctx context.Context, providerId int64) (gateway.BalanceSyncConfig, error) {
	cfg, err := s.GetBalanceConfig(ctx, providerId)
	if err != nil {
		return cfg, err
	}
	cfg.AccessKeyId = MaskSecret(cfg.AccessKeyId)
	cfg.AccessKeySecret = MaskSecret(cfg.AccessKeySecret)
	return cfg, nil
}

// SaveBalanceConfig 写采集配置（掩码占位保留旧明文，对齐凭证 MergeCredentialValues 语义）。
func (s *ProviderBalanceService) SaveBalanceConfig(ctx context.Context, providerId int64, incoming gateway.BalanceSyncConfig) error {
	p, err := (&ProviderService{}).GetProvider(ctx, providerId)
	if err != nil {
		return err
	}
	if _, ok := gateway.BalanceSyncProviderTypes[p.ProviderType]; !ok {
		return errors.Errorf("供应商类型 %s 暂不支持余量采集", p.ProviderType)
	}
	old, _ := decryptBalanceConfig(p.BalanceSyncConfig) // 未配置/解密失败均按空旧值处理
	merged := mergeBalanceConfig(old, incoming)
	if merged.AccessKeyId == "" || merged.AccessKeySecret == "" {
		return errors.New("AccessKeyId 与 AccessKeySecret 不能为空")
	}
	if merged.Region == "" {
		merged.Region = "cn-beijing" // Token Plan 仅支持华北2(北京)
	}
	enc, err := encryptBalanceConfig(merged)
	if err != nil {
		return err
	}
	return global.OPS_DB.WithContext(ctx).Model(&gateway.Provider{}).
		Where("provider_id = ?", providerId).
		Update("balance_sync_config", enc).Error
}

// mergeBalanceConfig 掩码合并：传入值等于旧值掩码 → 保留旧明文；其余覆盖。
func mergeBalanceConfig(oldCfg, incoming gateway.BalanceSyncConfig) gateway.BalanceSyncConfig {
	out := incoming
	if incoming.AccessKeyId == MaskSecret(oldCfg.AccessKeyId) {
		out.AccessKeyId = oldCfg.AccessKeyId
	}
	if incoming.AccessKeySecret == MaskSecret(oldCfg.AccessKeySecret) {
		out.AccessKeySecret = oldCfg.AccessKeySecret
	}
	return out
}

// encryptBalanceConfig 采集配置序列化后 AES-256-GCM 加密（密钥复用 litellm.credential-key）。
func encryptBalanceConfig(cfg gateway.BalanceSyncConfig) (string, error) {
	key := global.OPS_CONFIG.Litellm.CredentialKey
	if key == "" {
		return "", errors.New("凭证加密密钥未配置(litellm.credential-key)，拒绝写入")
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("采集配置序列化失败: %w", err)
	}
	return crypto.AESGCMEncrypt(string(raw), key)
}

// decryptBalanceConfig 解密采集配置；空串返回零值。
func decryptBalanceConfig(enc string) (gateway.BalanceSyncConfig, error) {
	cfg := gateway.BalanceSyncConfig{}
	if enc == "" {
		return cfg, nil
	}
	key := global.OPS_CONFIG.Litellm.CredentialKey
	if key == "" {
		return cfg, errors.New("凭证加密密钥未配置(litellm.credential-key)")
	}
	raw, err := crypto.AESGCMDecrypt(enc, key)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, fmt.Errorf("采集配置反序列化失败: %w", err)
	}
	return cfg, nil
}

// SyncProviderBalance 手动/定时同步入口：拉百炼坐席+共享包 → 事务整批重建快照。
func (s *ProviderBalanceService) SyncProviderBalance(ctx context.Context, providerId int64) (gatewayResp.ProviderBalanceSummary, error) {
	p, err := (&ProviderService{}).GetProvider(ctx, providerId)
	if err != nil {
		return gatewayResp.ProviderBalanceSummary{}, err
	}
	if _, ok := gateway.BalanceSyncProviderTypes[p.ProviderType]; !ok {
		return gatewayResp.ProviderBalanceSummary{}, errors.Errorf("供应商类型 %s 暂不支持余量采集", p.ProviderType)
	}
	summary, err := s.buildSummaryFromProvider(ctx, p)
	if err != nil {
		return summary, err
	}
	cfg, err := decryptBalanceConfig(p.BalanceSyncConfig)
	if err != nil {
		return summary, errors.Wrap(err, "读取采集配置失败")
	}
	if cfg.AccessKeyId == "" || cfg.AccessKeySecret == "" {
		return summary, errors.New("未配置采集凭证(阿里云 AK/SK)，请先在余量面板保存配置")
	}

	seats, pkgs, err := fetchBailianBalance(ctx, cfg)
	if err != nil {
		return summary, err
	}

	now := time.Now().UTC()
	rows := make([]gateway.ProviderBalance, 0, len(seats)+len(pkgs))
	for _, st := range seats {
		rows = append(rows, st.toBalance(providerId, now))
	}
	for _, pg := range pkgs {
		rows = append(rows, pg.toBalance(providerId, now))
	}
	if err := global.OPS_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 快照现状表：整批物理删除重建（软删会累积垃圾行，同 CostSummaryDaily 重建语义）
		if err := tx.Unscoped().Where("provider_id = ?", providerId).
			Delete(&gateway.ProviderBalance{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	}); err != nil {
		return summary, errors.Wrap(err, "余量快照写入失败")
	}
	logger.WithCtx(ctx).Mod("gateway").Field("providerId", providerId).
		Info(fmt.Sprintf("余量同步完成: 坐席 %d, 共享包 %d", len(seats), len(pkgs)))
	return s.buildSummary(ctx, providerId)
}

// SyncAllProviderBalances 定时任务入口：遍历白名单内已配置采集凭证的供应商，逐家同步（失败不阻断下一家）。
func (s *ProviderBalanceService) SyncAllProviderBalances(ctx context.Context) (map[string]int, error) {
	var providers []gateway.Provider
	if err := global.OPS_DB.WithContext(ctx).
		Where("provider_type IN ? AND balance_sync_config <> ''", balanceSyncProviderTypes()).
		Find(&providers).Error; err != nil {
		return nil, err
	}
	result := map[string]int{"total": len(providers), "ok": 0, "failed": 0}
	for _, p := range providers {
		if _, err := s.SyncProviderBalance(ctx, p.ProviderId); err != nil {
			result["failed"]++
			logger.WithCtx(ctx).Mod("gateway").Err(err).Field("providerId", p.ProviderId).Error("供应商余量同步失败(旁路,不阻断)")
			continue
		}
		result["ok"]++
	}
	return result, nil
}

// ---- 百炼 ModelStudio OpenAPI（ACS3-HMAC-SHA256 签名，GET）----

// bailianEquity 权益实例（TokenPlan 下每坐席/包仅一个 CREDITS 权益生效）。
type bailianEquity struct {
	EquityType        string  `json:"EquityType"`
	CycleStartTime    int64   `json:"CycleStartTime"`
	CycleEndTime      int64   `json:"CycleEndTime"`
	CycleTotalValue   float64 `json:"CycleTotalValue"`
	CycleSurplusValue float64 `json:"CycleSurplusValue"`
}

// bailianSeatItem GetSubscriptionSeatDetails.Items 元素。
type bailianSeatItem struct {
	InstanceCode   string          `json:"InstanceCode"`
	SeatId         string          `json:"SeatId"`
	SpecType       string          `json:"SpecType"`
	Status         string          `json:"Status"`
	AssignedStatus string          `json:"AssignedStatus"`
	AccountName    string          `json:"AccountName"`
	AccountEmail   string          `json:"AccountEmail"`
	StartTime      int64           `json:"StartTime"`
	EndTime        int64           `json:"EndTime"`
	EquityList     []bailianEquity `json:"EquityList"`
}

// bailianPackageItem ListSubscriptionSharedPackages.Items 元素。
type bailianPackageItem struct {
	InstanceCode string          `json:"InstanceCode"`
	Status       string          `json:"Status"`
	EquityList   []bailianEquity `json:"EquityList"`
}

// bailianEnvelope 通用响应壳。
type bailianEnvelope[T any] struct {
	Success bool   `json:"Success"`
	Code    string `json:"Code"`
	Message string `json:"Message"`
	Data    struct {
		Items []T `json:"Items"`
		Total int `json:"Total"`
	} `json:"Data"`
}

// toBalance 坐席条目 → 快照行（CREDITS 权益缺失时额度为 0，raw 保底排障）。
func (st bailianSeatItem) toBalance(providerId int64, now time.Time) gateway.ProviderBalance {
	b := gateway.ProviderBalance{
		ProviderId: providerId,
		PlanType:   "token_plan",
		ItemType:   gateway.BalanceItemTypeSeat,
		ItemKey:    st.SeatId,
		ItemName:   st.AccountName,
		SpecType:   st.SpecType,
		Status:     st.Status,
		SyncedAt:   now,
	}
	if raw, err := json.Marshal(st); err == nil {
		b.Raw = raw
	}
	if eq := firstEquity(st.EquityList); eq != nil {
		applyEquity(&b, eq)
	}
	return b
}

// toBalance 共享包条目 → 快照行。
func (pg bailianPackageItem) toBalance(providerId int64, now time.Time) gateway.ProviderBalance {
	b := gateway.ProviderBalance{
		ProviderId: providerId,
		PlanType:   "token_plan",
		ItemType:   gateway.BalanceItemTypePackage,
		ItemKey:    pg.InstanceCode,
		ItemName:   "共享用量包",
		Status:     pg.Status,
		SyncedAt:   now,
	}
	if raw, err := json.Marshal(pg); err == nil {
		b.Raw = raw
	}
	if eq := firstEquity(pg.EquityList); eq != nil {
		applyEquity(&b, eq)
	}
	return b
}

// firstEquity 取生效的 CREDITS 权益（无则取首个，再无返回 nil）。
func firstEquity(list []bailianEquity) *bailianEquity {
	for i := range list {
		if list[i].EquityType == "CREDITS" {
			return &list[i]
		}
	}
	if len(list) > 0 {
		return &list[0]
	}
	return nil
}

// applyEquity 权益额度落快照行（已用 = 总 - 剩余）。
func applyEquity(b *gateway.ProviderBalance, eq *bailianEquity) {
	b.EquityType = eq.EquityType
	b.CycleStart = normalizeUnixMillis(eq.CycleStartTime)
	b.CycleEnd = normalizeUnixMillis(eq.CycleEndTime)
	b.TotalValue = eq.CycleTotalValue
	b.SurplusValue = eq.CycleSurplusValue
	b.UsedValue = eq.CycleTotalValue - eq.CycleSurplusValue
}

// normalizeUnixMillis 厂商时间戳秒/毫秒两义（元数据标毫秒、示例给秒级），按量级自适应归一为 UTC 时间。
func normalizeUnixMillis(v int64) *time.Time {
	if v <= 0 {
		return nil
	}
	if v > 1e11 { // 毫秒量级(2001年后毫秒均 >1e11)
		t := time.UnixMilli(v).UTC()
		return &t
	}
	t := time.Unix(v, 0).UTC()
	return &t
}

// fetchBailianBalance 拉全量坐席 + 共享包。
func fetchBailianBalance(ctx context.Context, cfg gateway.BalanceSyncConfig) ([]bailianSeatItem, []bailianPackageItem, error) {
	seats, err := fetchBailianPaged[bailianSeatItem](ctx, cfg, bailianSeatsPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "拉取坐席明细失败")
	}
	pkgs, err := fetchBailianPaged[bailianPackageItem](ctx, cfg, bailianPkgPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "拉取共享用量包失败")
	}
	return seats, pkgs, nil
}

// fetchBailianPaged 分页拉取泛型条目（按 Total 翻页，带上限保护）。
func fetchBailianPaged[T any](ctx context.Context, cfg gateway.BalanceSyncConfig, path string) ([]T, error) {
	var all []T
	for page := 1; page <= balanceMaxPages; page++ {
		query := url.Values{}
		query.Set("PageNo", fmt.Sprintf("%d", page))
		query.Set("PageSize", fmt.Sprintf("%d", balancePageSize))
		var resp bailianEnvelope[T]
		if err := callBailianOpenAPI(ctx, cfg, path, query, &resp); err != nil {
			return nil, err
		}
		if !resp.Success {
			return nil, errors.Errorf("百炼 OpenAPI 返回失败: code=%s message=%s", resp.Code, resp.Message)
		}
		all = append(all, resp.Data.Items...)
		if len(all) >= resp.Data.Total || len(resp.Data.Items) == 0 {
			break
		}
	}
	return all, nil
}

// acs3SignedHeaders 参与签名的 header 集合（ACS3 要求字母序）。
var acs3SignedHeaders = []string{"content-type", "host", "x-acs-action", "x-acs-content-sha256", "x-acs-date", "x-acs-signature-nonce", "x-acs-version"}

// acs3CanonicalRequest 构造 ACS3 规范请求（纯函数，可单测）。
func acs3CanonicalRequest(method, path, canonicalQuery string, headers map[string]string, payloadHash string) string {
	var canonicalHeaders strings.Builder
	for _, h := range acs3SignedHeaders {
		canonicalHeaders.WriteString(h)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(headers[h])
		canonicalHeaders.WriteString("\n")
	}
	return strings.Join([]string{
		method,
		path,
		canonicalQuery,
		canonicalHeaders.String(),
		strings.Join(acs3SignedHeaders, ";"),
		payloadHash,
	}, "\n")
}

// acs3Authorization 由规范请求生成 Authorization 头（纯函数，可单测）。
func acs3Authorization(canonicalRequest, accessKeyId, accessKeySecret string) string {
	stringToSign := "ACS3-HMAC-SHA256\n" + hex.EncodeToString(sha256Sum([]byte(canonicalRequest)))
	mac := hmac.New(sha256.New, []byte(accessKeySecret))
	mac.Write([]byte(stringToSign))
	return fmt.Sprintf("ACS3-HMAC-SHA256 Credential=%s, SignedHeaders=%s, Signature=%s",
		accessKeyId, strings.Join(acs3SignedHeaders, ";"), hex.EncodeToString(mac.Sum(nil)))
}

// callBailianOpenAPI ACS3-HMAC-SHA256 签名调百炼 ModelStudio OpenAPI（GET，空 body）。
// 参考 api.aliyun.com 门户 GetSubscriptionSeatDetails / ListSubscriptionSharedPackages 元数据。
func callBailianOpenAPI(ctx context.Context, cfg gateway.BalanceSyncConfig, path string, query url.Values, out any) error {
	host := fmt.Sprintf("modelstudio.%s.aliyuncs.com", cfg.Region)
	action := map[string]string{
		bailianSeatsPath: "GetSubscriptionSeatDetails",
		bailianPkgPath:   "ListSubscriptionSharedPackages",
	}[path]
	if action == "" {
		return errors.Errorf("未知的百炼 OpenAPI 路径: %s", path)
	}
	endpoint := "https://" + host + path
	payloadHash := hex.EncodeToString(sha256Sum(nil)) // GET 空 body
	headers := map[string]string{
		"host":                  host,
		"content-type":          "application/json",
		"x-acs-action":          action,
		"x-acs-version":         bailianAPIVersion,
		"x-acs-date":            time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"x-acs-signature-nonce": uuid.NewString(),
		"x-acs-content-sha256":  payloadHash,
	}
	canonicalQuery := canonicalQueryString(query)
	authorization := acs3Authorization(
		acs3CanonicalRequest(http.MethodGet, path, canonicalQuery, headers, payloadHash),
		cfg.AccessKeyId, cfg.AccessKeySecret)
	headers["authorization"] = authorization

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+canonicalQuery, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		if k == "host" {
			continue // http.Request 已带 Host
		}
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "百炼 OpenAPI 请求失败")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return errors.Wrap(err, "读取百炼 OpenAPI 响应失败")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("百炼 OpenAPI HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return errors.Wrapf(err, "百炼 OpenAPI 响应解析失败: %s", truncate(string(body), 300))
	}
	return nil
}

// canonicalQueryString 规范化 query：按 key 字典序，RFC3986 编码（+ 与 %20 不混用）。
func canonicalQueryString(query url.Values) string {
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, v := range query[k] {
			pairs = append(pairs, url.PathEscape(k)+"="+url.PathEscape(v))
		}
	}
	return strings.Join(pairs, "&")
}

// sha256Sum 摘要辅助。
func sha256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

// truncate 截断错误文本。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
