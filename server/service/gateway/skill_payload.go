package gateway

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"

	"github.com/hllkk/devops-admin/server/model/gateway"
)

// 本文件是 Skill 的投影与校验纯函数层（零 DB/配置依赖，可单测）。
// Skill 是平台自有资源（不经 LiteLLM），纯函数只做元数据归一/zip 文件名安全/
// tags 序列化，无投影换算。

// skillVersionRE 版本号宽松 semver：1~3 段数字（1 / 1.0 / 1.0.0），段长≤5。
var skillVersionRE = regexp.MustCompile(`^\d{1,5}(\.\d{1,5}){0,2}$`)

// NormalizeSkillVersion 版本号归一：空→默认 1.0.0；格式非法报错。
func NormalizeSkillVersion(v string) (string, error) {
	if v == "" {
		return "1.0.0", nil
	}
	if !skillVersionRE.MatchString(v) {
		return "", fmt.Errorf("版本号格式非法(应为 1 / 1.0 / 1.0.0 形式)")
	}
	return v, nil
}

// NormalizeSkillCategory 分类归一：空→general；去首尾空白；超长截断到 50。
func NormalizeSkillCategory(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return "general"
	}
	if len(c) > 50 {
		c = c[:50]
	}
	return c
}

// CleanSkillTags 标签清洗：逐项去空白、剔空串、去重保序。
func CleanSkillTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// MarshalSkillTags 标签序列化落库（nil 给空数组）。
func MarshalSkillTags(tags []string) datatypes.JSON {
	if tags == nil {
		tags = []string{}
	}
	raw, _ := json.Marshal(tags)
	return datatypes.JSON(raw)
}

// UnmarshalSkillTags 标签反序列化出网（空给空数组，坏值容错为空）。
func UnmarshalSkillTags(raw datatypes.JSON) []string {
	out := []string{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

// SkillZipFilename zip 存储键：{skillId}_{yyyyMMddHHmmss}.zip。
// 不掺原始文件名（防特殊字符/路径段），原始名另存 zip_origin_name 供下载回显。
func SkillZipFilename(skillId int64, now time.Time) string {
	return fmt.Sprintf("%d_%s.zip", skillId, now.Format("20060102150405"))
}

// ValidSkillUploadFilename 校验上传文件名：非空、.zip 后缀、无路径段/穿越字符。
func ValidSkillUploadFilename(name string) bool {
	if name == "" || strings.ToLower(filepath.Ext(name)) != ".zip" {
		return false
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return false
	}
	return true
}

// ValidateSkillZipStructure 解包校验技能包结构：zip 内须含 SKILL.md 入口文件
// (skill 包规范，位于包内任意目录均可，文件名大小写不敏感)。防上传无效包/
// 非 skill 包——发布前强制有包，包本身不合法会让"发布→下载→安装"链路落空。
func ValidateSkillZipStructure(p string) error {
	r, err := zip.OpenReader(p)
	if err != nil {
		return fmt.Errorf("zip 包无法解析: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && strings.EqualFold(filepath.Base(f.Name), "SKILL.md") {
			return nil
		}
	}
	return errors.New("技能包内未找到 SKILL.md 入口文件，请按 skill 包规范打包")
}

// SkillIdentityOf skill → 授权锚点字符串（AiKey.skills JSONB 元素）。
// ID 不可变，无改名级联；与 models(modelKey)/mcps(serverName) 的"字符串锚点"约定对齐。
func SkillIdentityOf(s gateway.Skill) string {
	return fmt.Sprintf("%d", s.SkillId)
}

// ExtractAgentToken Agent 直连端点凭证提取：Authorization: Bearer <key> 优先，
// 回落查询参数 ?token=<key>(curl 简单场景免 -H；AIHelms 同款双通道)。
// Bearer 后空串/纯空白视为未提供。
func ExtractAgentToken(authHeader, queryToken string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		if t := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")); t != "" {
			return t
		}
	}
	return strings.TrimSpace(queryToken)
}

// BuildAgentDownloadURL Agent 直连下载绝对 URL：{origin}{routerPrefix}/gateway/skill/agent/{id}/zip。
// origin 尾斜杠归一；prefix 为空(router-prefix 默认 "")时直连拼接。
func BuildAgentDownloadURL(origin, routerPrefix string, skillId int64) string {
	return strings.TrimRight(origin, "/") + routerPrefix +
		"/gateway/skill/agent/" + strconv.FormatInt(skillId, 10) + "/zip"
}
