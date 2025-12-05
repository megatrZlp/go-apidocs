package render

import (
	"encoding/json"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
)

// exampleJSON 基于 $ref 生成 JSON 示例文本（缩进 4 空格）。
func exampleJSON(j *gjson.Json, ref string) string {
	ex := exampleValue(j, ref)
	bs, _ := json.MarshalIndent(ex, "", "    ")
	return string(bs)
}

// exampleJSONFromSchema 基于内联 schema 生成 JSON 示例文本（缩进 4 空格）。
func exampleJSONFromSchema(j *gjson.Json, s *gjson.Json) string {
	ex := exampleValueFromSchema(j, s)
	bs, _ := json.MarshalIndent(ex, "", "    ")
	return string(bs)
}

// exampleValue 返回引用 schema 的示例结构（用于拼装示例 JSON）。
func exampleValue(j *gjson.Json, ref string) interface{} {
	name := componentNameFromRef(ref)
	sj := getSchema(j, name)
	if sj == nil {
		return nil
	}
	if sj.Get("type").String() == "object" || len(sj.GetJsonMap("properties")) > 0 {
		props := sj.GetJsonMap("properties")
		m := map[string]interface{}{}
		for k, pj := range props {
			if rr := pj.Get("$ref").String(); rr != "" {
				m[k] = exampleValue(j, rr)
				continue
			}
			typ := pj.Get("type").String()
			switch typ {
			case "string":
				m[k] = "string"
			case "integer":
				m[k] = 0
			case "number":
				m[k] = 0
			case "boolean":
				m[k] = false
			default:
				m[k] = nil
			}
		}
		return m
	}
	switch sj.Get("type").String() {
	case "string":
		return "string"
	case "integer":
		return 0
	case "number":
		return 0
	case "boolean":
		return false
	}
	return nil
}

// exampleValueFromSchema 返回内联 schema 的示例结构（用于拼装示例 JSON）。
func exampleValueFromSchema(j *gjson.Json, sj *gjson.Json) interface{} {
	if sj == nil {
		return nil
	}
	if rr := sj.Get("$ref").String(); rr != "" {
		sub := getRefJson(j, rr)
		return exampleValueFromSchema(j, sub)
	}
	if sj.Get("type").String() == "array" {
		it := sj.GetJson("items")
		if it != nil {
			if r2 := it.Get("$ref").String(); r2 != "" {
				sub := getRefJson(j, r2)
				return []interface{}{exampleValueFromSchema(j, sub)}
			} else if it.Get("type").String() == "object" {
				return []interface{}{exampleValueFromSchema(j, it)}
			} else {
				return []interface{}{}
			}
		}
		return []interface{}{}
	}
	if sj.Get("type").String() == "object" || len(sj.GetJsonMap("properties")) > 0 {
		props := sj.GetJsonMap("properties")
		m := map[string]interface{}{}
		for k, pj := range props {
			if rr := pj.Get("$ref").String(); rr != "" {
				sub := getRefJson(j, rr)
				m[k] = exampleValueFromSchema(j, sub)
				continue
			}
			typ := pj.Get("type").String()
			switch typ {
			case "string":
				m[k] = "string"
			case "integer":
				m[k] = 0
			case "number":
				m[k] = 0
			case "boolean":
				m[k] = false
			case "object":
				m[k] = exampleValueFromSchema(j, pj)
			case "array":
				it := pj.GetJson("items")
				if it != nil {
					if r2 := it.Get("$ref").String(); r2 != "" {
						sub := getRefJson(j, r2)
						m[k] = []interface{}{exampleValueFromSchema(j, sub)}
					} else if it.Get("type").String() == "object" {
						m[k] = []interface{}{exampleValueFromSchema(j, it)}
					} else {
						m[k] = []interface{}{}
					}
				} else {
					m[k] = []interface{}{}
				}
			default:
				m[k] = nil
			}
		}
		return m
	}
	switch sj.Get("type").String() {
	case "string":
		return "string"
	case "integer":
		return 0
	case "number":
		return 0
	case "boolean":
		return false
	}
	return nil
}

// filterExampleDataLeaves 按白名单过滤示例中的 data 字段：
// - 顶层 code/message/data 始终保留；
// - 允许完整路径与叶子名简写；数组路径 .items[] 与 [] 等价；
// - 仅保留叶子基础类型，跳过对象中间节点。
func filterExampleDataLeaves(ex interface{}, allowed []string) interface{} {
	aset := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		if a == "" {
			continue
		}
		a = strings.ReplaceAll(a, ".items", "")
		for strings.Contains(a, "[][]") {
			a = strings.ReplaceAll(a, "[][]", "[]")
		}
		aset[a] = struct{}{}
	}
	m, ok := ex.(map[string]interface{})
	if !ok {
		return ex
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		if k == "code" || k == "message" {
			out[k] = v
			continue
		}
		if k == "data" {
			out[k] = filterValueLeaves(v, aset, "data")
			continue
		}
		out[k] = v
	}
	return out
}

// filterValueLeaves 递归过滤对象/数组中的叶子字段，prefix 用于构建完整路径。
func filterValueLeaves(v interface{}, aset map[string]struct{}, prefix string) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		res := make(map[string]interface{}, len(t))
		for k, vv := range t {
			switch vv.(type) {
			case map[string]interface{}, []interface{}:
				pr := filterValueLeaves(vv, aset, pathJoin(prefix, k))
				if isNonEmpty(pr) {
					res[k] = pr
				}
			default:
				leafPath := pathJoin(prefix, k)
				if _, ok := aset[k]; ok || matchAllowedPath(aset, leafPath) {
					res[k] = vv
				}
			}
		}
		return res
	case []interface{}:
		if len(t) == 0 {
			return t
		}
		pr := filterValueLeaves(t[0], aset, prefix+"[]")
		if isNonEmpty(pr) {
			return []interface{}{pr}
		}
		return []interface{}{}
	default:
		return v
	}
}

// pathJoin 组合路径前缀与键名。
func pathJoin(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// matchAllowedPath 判断完整路径是否命中白名单（兼容 [] 与移除 [] 的两种形式）。
func matchAllowedPath(aset map[string]struct{}, path string) bool {
	if _, ok := aset[path]; ok {
		return true
	}
	p1 := strings.ReplaceAll(path, "[].", "[]")
	if _, ok := aset[p1]; ok {
		return true
	}
	p2 := strings.ReplaceAll(path, "[]", "")
	if _, ok := aset[p2]; ok {
		return true
	}
	return false
}

// isNonEmpty 判断对象或数组是否非空，用于过滤空分支。
func isNonEmpty(v interface{}) bool {
	switch x := v.(type) {
	case map[string]interface{}:
		return len(x) > 0
	case []interface{}:
		return len(x) > 0
	default:
		return v != nil
	}
}

// exampleJSONFromSchemaWithAllowed 基于内联 schema 生成示例并应用白名单过滤。
func exampleJSONFromSchemaWithAllowed(j *gjson.Json, s *gjson.Json, allowed []string) string {
	ex := exampleValueFromSchema(j, s)
	ex = filterExampleDataLeaves(ex, allowed)
	bs, _ := json.MarshalIndent(ex, "", "    ")
	return string(bs)
}

// exampleJSONWithAllowed 基于 $ref 生成示例并应用白名单过滤。
func exampleJSONWithAllowed(j *gjson.Json, ref string, allowed []string) string {
	ex := exampleValue(j, ref)
	ex = filterExampleDataLeaves(ex, allowed)
	bs, _ := json.MarshalIndent(ex, "", "    ")
	return string(bs)
}
