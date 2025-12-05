package tools

import (
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
)

// GetRefJSON 解析 components 下的 $ref（支持 schemas/parameters 两类引用）。
func GetRefJSON(j *gjson.Json, ref string) *gjson.Json {
	if strings.HasPrefix(ref, "#/components/schemas/") {
		if j == nil {
			return nil
		}
		return GetSchema(j, ComponentNameFromRef(ref))
	}
	if strings.HasPrefix(ref, "#/components/parameters/") {
		if j == nil {
			return nil
		}
		pm := j.GetJsonMap("components.parameters")
		return pm[ComponentNameFromRef(ref)]
	}
	return nil
}

// GetSchema 获取 components.schemas 下指定名称的 schema。
func GetSchema(j *gjson.Json, name string) *gjson.Json {
	if j == nil {
		return nil
	}
	sm := j.GetJsonMap("components.schemas")
	return sm[name]
}

// HTMLEscape 安全转义 HTML（避免表格与 code 中出现未转义的特殊字符）。
func HTMLEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// SetFromArray 将字符串数组构建为集合。
func SetFromArray(arr []interface{}) map[string]bool {
	m := make(map[string]bool)
	for _, v := range arr {
		if s, ok := v.(string); ok {
			m[s] = true
		}
	}
	return m
}

// JSONArrayStrings 提取字符串数组。
func JSONArrayStrings(arr []interface{}) []string {
	res := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			res = append(res, s)
		}
	}
	return res
}

// SortStrings 原地执行稳定的升序排序。
func SortStrings(a []string) {
	if len(a) < 2 {
		return
	}
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

// AnchorID 生成 endpoint 的锚点 ID。
func AnchorID(method, path string) string {
	s := method + "-" + path
	r := strings.NewReplacer(
		"/", "-",
		" ", "-",
		"{", "",
		"}", "",
		":", "-",
		"?", "-",
		"&", "-",
		"=", "-",
		".", "-",
		",", "-",
		"@", "-",
	)
	s = r.Replace(s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	return s
}

// Slugify 将分组标题转为锚点友好的短串。
func Slugify(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	r := strings.NewReplacer(" ", "-", "/", "-", ".", "-", "(", "", ")", "", "[", "", "]", "")
	s = r.Replace(s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	return s
}

// SplitTagParts 将 tags[0] 按“主/次”分组，例如 “用户模块/用户管理”。
func SplitTagParts(tags []string) (string, string) {
	if len(tags) == 0 {
		return "未分组", "默认"
	}
	s := tags[0]
	i := strings.Index(s, "/")
	if i < 0 {
		return s, "默认"
	}
	pre := s[:i]
	suf := s[i+1:]
	if suf == "" {
		suf = "默认"
	}
	return pre, suf
}

// ComponentNameFromRef 从 $ref 中提取末尾组件名。
func ComponentNameFromRef(ref string) string {
	i := strings.LastIndex(ref, "/")
	if i < 0 {
		return ""
	}
	return ref[i+1:]
}

// StripComponentTypeDecorations 规范化类型显示，移除如 object(Component) 与 array(object(Component)) 的组件名装饰，
// 使表格与 Markdown 输出中的类型更简洁（只保留 object/array(object) 等）。
func StripComponentTypeDecorations(s string) string {
	for {
		i := strings.Index(s, "array(object(")
		if i < 0 {
			break
		}
		j := strings.Index(s[i+len("array(object("):], ")")
		if j < 0 {
			break
		}
		s = s[:i] + "array(object)" + s[i+len("array(object(")+j+1:]
	}
	for {
		i := strings.Index(s, "object(")
		if i < 0 {
			break
		}
		j := strings.Index(s[i+len("object("):], ")")
		if j < 0 {
			break
		}
		s = s[:i] + "object" + s[i+len("object(")+j+1:]
	}
	return s
}
