package gateway

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"testing"

	"github.com/hllkk/devops-admin/server/model/gateway"
)

// TestAcs3CanonicalRequest 验证规范请求拼接：方法/路径/query 各占一行，
// canonical headers 逐行 name:value\n，signedHeaders 与 payloadHash 收尾。
func TestAcs3CanonicalRequest(t *testing.T) {
	payloadHash := hex.EncodeToString(sha256Sum(nil))
	headers := map[string]string{
		"host":                  "modelstudio.cn-beijing.aliyuncs.com",
		"content-type":          "application/json",
		"x-acs-action":          "GetSubscriptionSeatDetails",
		"x-acs-version":         bailianAPIVersion,
		"x-acs-date":            "2026-08-27T08:00:00Z",
		"x-acs-signature-nonce": "nonce-123",
		"x-acs-content-sha256":  payloadHash,
	}
	cr := acs3CanonicalRequest("GET", bailianSeatsPath, "PageNo=1&PageSize=100", headers, payloadHash)
	lines := strings.Split(cr, "\n")
	// 3(method/path/query) + 7(headers) + 1(headers 尾\n 与 Join 分隔产生的空行) + 1(signedHeaders) + 1(payloadHash)
	if len(lines) != 13 {
		t.Fatalf("规范请求行数不符: got %d\n%s", len(lines), cr)
	}
	if lines[0] != "GET" || lines[1] != bailianSeatsPath || lines[2] != "PageNo=1&PageSize=100" {
		t.Fatalf("规范请求前3行不符:\n%s", cr)
	}
	if lines[3] != "content-type:application/json" || lines[9] != "x-acs-version:"+bailianAPIVersion {
		t.Fatalf("canonical header 段不符:\n%s", cr)
	}
	if lines[10] != "" {
		t.Fatalf("headers 段应以空行收尾:\n%s", cr)
	}
	if lines[11] != strings.Join(acs3SignedHeaders, ";") {
		t.Fatalf("signedHeaders 行不符: %s", lines[11])
	}
	if lines[12] != payloadHash {
		t.Fatalf("payloadHash 行不符")
	}
}

// TestAcs3Authorization 验证签名可复算（HMAC-SHA256 over ACS3-HMAC-SHA256\n+hex(sha256(cr))）。
func TestAcs3Authorization(t *testing.T) {
	cr := "GET\n/path\nk=v\nhost:h\n"
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte("ACS3-HMAC-SHA256\n" + hex.EncodeToString(sha256Sum([]byte(cr)))))
	want := "ACS3-HMAC-SHA256 Credential=ak, SignedHeaders=" + strings.Join(acs3SignedHeaders, ";") +
		", Signature=" + hex.EncodeToString(mac.Sum(nil))
	if got := acs3Authorization(cr, "ak", "secret"); got != want {
		t.Fatalf("Authorization 不符:\ngot  %s\nwant %s", got, want)
	}
}

// TestCanonicalQueryString 验证 query 字典序 + RFC3986 风格编码（空格 %20 不用 +）。
func TestCanonicalQueryString(t *testing.T) {
	q := url.Values{}
	q.Set("PageSize", "100")
	q.Set("PageNo", "1")
	q.Set("StatusList", `["NORMAL"]`)
	got := canonicalQueryString(q)
	want := `PageNo=1&PageSize=100&StatusList=%5B%22NORMAL%22%5D`
	if got != want {
		t.Fatalf("canonical query 不符:\ngot  %s\nwant %s", got, want)
	}
}

// TestNormalizeUnixMillis 秒/毫秒两义自适应（元数据标毫秒、示例给秒级）。
func TestNormalizeUnixMillis(t *testing.T) {
	if r := normalizeUnixMillis(0); r != nil {
		t.Fatalf("0 应返回 nil")
	}
	sec := normalizeUnixMillis(1775232000)
	if sec == nil || sec.Year() != 2026 {
		t.Fatalf("秒级解析不符: %v", sec)
	}
	ms := normalizeUnixMillis(1775232000123)
	if ms == nil || ms.Year() != 2026 || ms.Nanosecond() != 123000000 {
		t.Fatalf("毫秒级解析不符: %v", ms)
	}
}

// TestMergeBalanceConfig 掩码占位保留旧明文（对齐凭证 MergeCredentialValues 语义）。
func TestMergeBalanceConfig(t *testing.T) {
	old := gateway.BalanceSyncConfig{AccessKeyId: "LTAI5tXXXXXXXXXXXX1234", AccessKeySecret: "abcdef1234567890abcdef", Region: "cn-beijing"}
	// 传回掩码 → 保留旧明文
	got := mergeBalanceConfig(old, gateway.BalanceSyncConfig{
		AccessKeyId: MaskSecret(old.AccessKeyId), AccessKeySecret: MaskSecret(old.AccessKeySecret), Region: "cn-beijing",
	})
	if got.AccessKeyId != old.AccessKeyId || got.AccessKeySecret != old.AccessKeySecret {
		t.Fatalf("掩码占位应保留旧明文: %+v", got)
	}
	// 传新值 → 覆盖
	got = mergeBalanceConfig(old, gateway.BalanceSyncConfig{AccessKeyId: "NEWAK", AccessKeySecret: "NEWSECRET", Region: "cn-beijing"})
	if got.AccessKeyId != "NEWAK" || got.AccessKeySecret != "NEWSECRET" {
		t.Fatalf("新值应覆盖: %+v", got)
	}
}
