package render

import (
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
)

// sanitizeType 统一类型字符串的展示形式，收敛为 object/array(object) 等简洁写法，
// 避免在文档中出现过长的组件名装饰。
func sanitizeType(t string) string {
	if strings.HasPrefix(t, "object(") {
		return "object"
	}
	if strings.HasPrefix(t, "array(object(") {
		return "array(object)"
	}
	return t
}

func titleDescription(s *gjson.Json) string {
	if s == nil {
		return ""
	}
	t := strings.TrimSpace(s.Get("title").String())
	d := strings.TrimSpace(s.Get("description").String())
	if t != "" && d != "" {
		return t + " " + d
	}
	if t != "" {
		return t
	}
	return d
}

// mergedProperties 合并 schema 的属性集：当 properties 为空时，
// 会遍历 allOf/oneOf/anyOf 的每个元素并递归解析 $ref，最终返回统一的属性映射。
func mergedProperties(j *gjson.Json, s *gjson.Json) map[string]*gjson.Json {
	props := s.GetJsonMap("properties")
	if len(props) == 0 {
		for _, arrName := range []string{"allOf", "oneOf", "anyOf"} {
			for _, it := range s.Get(arrName).Array() {
				elem := gjson.New(it)
				if r := elem.Get("$ref").String(); r != "" {
					sub := getRefJson(j, r)
					if sub != nil {
						for k, v := range sub.GetJsonMap("properties") {
							props[k] = v
						}
					}
				} else {
					for k, v := range elem.GetJsonMap("properties") {
						props[k] = v
					}
				}
			}
		}
	}
	return props
}

// FieldInfo 用于参数/返回说明的扁平表结构。
type FieldInfo struct {
	Path     string
	Required bool
	Type     string
	Desc     string
}

// paramInfo 用于 Header/Path/Query 参数列表的展示结构。
type paramInfo struct {
	Name     string
	Required string
	Type     string
	Desc     string
}

// renderParamInfoTableHTML 将参数列表渲染为 HTML 表格。
func renderParamInfoTableHTML(list []paramInfo) string {
	var b strings.Builder
	b.WriteString("<table><thead><tr><th>参数名</th><th>必选</th><th>类型</th><th>说明</th></tr></thead><tbody>")
	for _, it := range list {
		b.WriteString("<tr><td>" + htmlEscape(it.Name) + "</td><td>" + it.Required + "</td><td>" + htmlEscape(it.Type) + "</td><td>" + htmlEscape(it.Desc) + "</td></tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

// renderParamInfoTableMarkdown 将参数列表渲染为 Markdown 表格。
func renderParamInfoTableMarkdown(list []paramInfo) string {
	var b strings.Builder
	b.WriteString("| 参数名 | 必选 | 类型 | 说明 |\n|---|---|---|---|\n")
	for _, it := range list {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", it.Name, it.Required, it.Type, it.Desc))
	}
	return b.String()
}

// filterFieldInfos 根据白名单过滤字段行；支持顶层保留与 data 叶子名简写。
func filterFieldInfos(fields []FieldInfo, allowed []string) []FieldInfo {
	if len(allowed) == 0 {
		return fields
	}
	norm := make([]string, 0, len(allowed))
	for _, a := range allowed {
		if a == "" {
			continue
		}
		a = strings.ReplaceAll(a, ".items", "")
		for strings.Contains(a, "[][]") {
			a = strings.ReplaceAll(a, "[][]", "[]")
		}
		norm = append(norm, a)
	}
	out := make([]FieldInfo, 0, len(fields))
	for _, f := range fields {
		if f.Path == "code" || f.Path == "message" || f.Path == "data" {
			out = append(out, f)
			continue
		}
		keep := false
		for _, a := range norm {
			if a == "" {
				continue
			}
			if f.Path == a || strings.HasPrefix(f.Path, a+".") || strings.HasPrefix(f.Path, a+"[]") || strings.HasPrefix(f.Path, a+"[].") {
				keep = true
				break
			}
			if strings.HasPrefix(f.Path, "data") && !strings.ContainsAny(a, ".[]") {
				clean := strings.ReplaceAll(f.Path, "[]", "")
				parts := strings.Split(clean, ".")
				last := parts[len(parts)-1]
				if last == a {
					t := f.Type
					if t != "object" && !strings.HasPrefix(t, "object(") {
						keep = true
						break
					}
				}
			}
		}
		if keep {
			out = append(out, f)
		}
	}
	return out
}

// renderParamTableHTMLWithAllowed 渲染请求参数（通过 $ref）并按白名单过滤。
func renderParamTableHTMLWithAllowed(j *gjson.Json, ref string, allowed []string) string {
	fields := flattenSchemaFields(j, ref, "")
	fields = filterFieldInfos(fields, allowed)
	var b strings.Builder
	b.WriteString("<table><thead><tr><th>参数名</th><th>必选</th><th>类型</th><th>说明</th></tr></thead><tbody>")
	for _, f := range fields {
		req := "否"
		if f.Required {
			req = "是"
		}
		b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + req + "</td><td>" + htmlEscape(f.Type) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

// renderParamTableMarkdownWithAllowed 渲染请求参数为 Markdown（通过 $ref）并按白名单过滤。
func renderParamTableMarkdownWithAllowed(j *gjson.Json, ref string, allowed []string) string {
	fields := flattenSchemaFields(j, ref, "")
	fields = filterFieldInfos(fields, allowed)
	var b strings.Builder
	b.WriteString("| 参数名 | 必选 | 类型 | 说明 |\n|---|---|---|---|\n")
	for _, f := range fields {
		req := "否"
		if f.Required {
			req = "是"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", f.Path, req, f.Type, f.Desc))
	}
	return b.String()
}

// renderParamTableHTMLFromJsonWithAllowed 渲染请求参数（内联 schema）并按白名单过滤。
func renderParamTableHTMLFromJsonWithAllowed(j *gjson.Json, sj *gjson.Json, allowed []string) string {
	fields := flattenSchemaFieldsFromJson(j, sj, "")
	fields = filterFieldInfos(fields, allowed)
	var b strings.Builder
	b.WriteString("<table><thead><tr><th>参数名</th><th>必选</th><th>类型</th><th>说明</th></tr></thead><tbody>")
	for _, f := range fields {
		req := "否"
		if f.Required {
			req = "是"
		}
		b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + req + "</td><td>" + htmlEscape(f.Type) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

// renderParamTableHTML 渲染请求参数（通过 $ref）。
func renderParamTableHTML(j *gjson.Json, ref string) string {
	fields := flattenSchemaFields(j, ref, "")
	var b strings.Builder
	b.WriteString("<table><thead><tr><th>参数名</th><th>必选</th><th>类型</th><th>说明</th></tr></thead><tbody>")
	for _, f := range fields {
		req := "否"
		if f.Required {
			req = "是"
		}
		b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + req + "</td><td>" + htmlEscape(f.Type) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

// renderParamTableMarkdown 渲染请求参数为 Markdown（通过 $ref）。
func renderParamTableMarkdown(j *gjson.Json, ref string) string {
	fields := flattenSchemaFields(j, ref, "")
	var b strings.Builder
	b.WriteString("| 参数名 | 必选 | 类型 | 说明 |\n|---|---|---|---|\n")
	for _, f := range fields {
		req := "否"
		if f.Required {
			req = "是"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", f.Path, req, f.Type, f.Desc))
	}
	return b.String()
}

// renderParamTableHTMLFromJson 渲染请求参数（内联 schema）。
func renderParamTableHTMLFromJson(j *gjson.Json, sj *gjson.Json) string {
	fields := flattenSchemaFieldsFromJson(j, sj, "")
	var b strings.Builder
	b.WriteString("<table><thead><tr><th>参数名</th><th>必选</th><th>类型</th><th>说明</th></tr></thead><tbody>")
	for _, f := range fields {
		req := "否"
		if f.Required {
			req = "是"
		}
		b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + req + "</td><td>" + htmlEscape(f.Type) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

// renderParamTableMarkdownFromJson 渲染请求参数为 Markdown（内联 schema）。
func renderParamTableMarkdownFromJson(j *gjson.Json, sj *gjson.Json) string {
	fields := flattenSchemaFieldsFromJson(j, sj, "")
	var b strings.Builder
	b.WriteString("| 参数名 | 必选 | 类型 | 说明 |\n|---|---|---|---|\n")
	for _, f := range fields {
		req := "否"
		if f.Required {
			req = "是"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", f.Path, req, f.Type, f.Desc))
	}
	return b.String()
}

// renderParamTableMarkdownFromJsonWithAllowed 渲染“请求参数”为 Markdown 表格（内联 schema），
// 并按 allowed 白名单过滤，仅保留顶层 code/message/data 与命中的 data 下叶子或完整路径字段。
func renderParamTableMarkdownFromJsonWithAllowed(j *gjson.Json, sj *gjson.Json, allowed []string) string {
	fields := flattenSchemaFieldsFromJson(j, sj, "")
	fields = filterFieldInfos(fields, allowed)
	var b strings.Builder
	b.WriteString("| 参数名 | 必选 | 类型 | 说明 |\n|---|---|---|---|\n")
	for _, f := range fields {
		req := "否"
		if f.Required {
			req = "是"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", f.Path, req, f.Type, f.Desc))
	}
	return b.String()
}

// renderResponseParamFlatTableHTMLFromJson 渲染返回参数说明为 HTML（自动展开 data）。
func renderResponseParamFlatTableHTMLFromJson(j *gjson.Json, sj *gjson.Json) string {
	target := sj
	prefix := ""
	for _, key := range []string{"data", "payload", "result", "content"} {
		if !sj.Get("properties." + key).IsNil() {
			d := sj.GetJson("properties." + key)
			if r := d.Get("$ref").String(); r != "" {
				target = getSchema(j, componentNameFromRef(r))
			} else {
				target = d
			}
			if target != sj {
				prefix = key
			}
			break
		}
	}
	var fields []FieldInfo
	if prefix != "" {
		t := target.Get("type").String()
		if t == "array" {
			it := target.GetJson("items")
			if it != nil {
				if r := it.Get("$ref").String(); r != "" {
					t = "array(object)"
				} else if it.Get("type").String() != "" {
					t = "array(" + it.Get("type").String() + ")"
				} else {
					t = "array"
				}
			} else {
				t = "array"
			}
		} else if t == "" {
			t = "object"
		}
		desc := sj.Get("properties." + prefix + ".description").String()
		fields = append(fields, FieldInfo{Path: prefix, Type: t, Desc: desc})
	}
	top := mergedProperties(j, sj)
	if len(top) > 0 {
		keysTop := make([]string, 0, len(top))
		for k := range top {
			if prefix == "" || k != prefix {
				keysTop = append(keysTop, k)
			}
		}
		sortStrings(keysTop)
		for _, k := range keysTop {
			p := top[k]
			t := p.Get("type").String()
			if r := p.Get("$ref").String(); r != "" {
				t = "object"
			} else if t == "array" {
				it0 := p.GetJson("items")
				if it0 != nil {
					if r0 := it0.Get("$ref").String(); r0 != "" {
						t = "array(object)"
					} else if it0.Get("type").String() != "" {
						t = "array(" + it0.Get("type").String() + ")"
					} else {
						t = "array"
					}
				} else {
					t = "array"
				}
			} else if t == "" {
				t = "object"
			}
			fields = append(fields, FieldInfo{Path: k, Type: t, Desc: p.Get("description").String()})
		}
	}
	if target.Get("type").String() == "array" {
		it := target.GetJson("items")
		if it != nil {
			if r := it.Get("$ref").String(); r != "" {
				sub := getRefJson(j, r)
				if sub != nil {
					for k, p := range mergedProperties(j, sub) {
						t := p.Get("type").String()
						if rr := p.Get("$ref").String(); rr != "" {
							t = "object"
						}
						if t == "array" {
							it2 := p.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									t = "array(object)"
								} else if it2.Get("type").String() != "" {
									t = "array(" + it2.Get("type").String() + ")"
								} else {
									t = "array"
								}
							} else {
								t = "array"
							}
						}
						fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
					}
				}
			} else if it.Get("type").String() == "object" {
				props := mergedProperties(j, it)
				keys := make([]string, 0, len(props))
				for k := range props {
					keys = append(keys, k)
				}
				sortStrings(keys)
				for _, k := range keys {
					p := props[k]
					t := p.Get("type").String()
					if rr := p.Get("$ref").String(); rr != "" {
						t = "object"
					}
					if t == "array" {
						it2 := p.GetJson("items")
						if it2 != nil {
							if r2 := it2.Get("$ref").String(); r2 != "" {
								t = "array(object)"
							} else if it2.Get("type").String() != "" {
								t = "array(" + it2.Get("type").String() + ")"
							} else {
								t = "array"
							}
						} else {
							t = "array"
						}
					} else if t == "" {
						t = "object"
					}
					fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
				}
			} else {
				tt := it.Get("type").String()
				if tt != "" {
					fields = append(fields, FieldInfo{Path: prefix + "[]", Type: "array(" + tt + ")", Desc: it.Get("description").String()})
				}
			}
		}
	} else {
		props := mergedProperties(j, target)
		keys := make([]string, 0, len(props))
		for k := range props {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			p := props[k]
			cur := k
			if prefix != "" {
				cur = prefix + "." + k
			}
			t := p.Get("type").String()
			if r := p.Get("$ref").String(); r != "" {
				fields = append(fields, FieldInfo{Path: cur, Type: "object", Desc: p.Get("description").String()})
				sub := getRefJson(j, r)
				if sub != nil {
					for sk, sp := range mergedProperties(j, sub) {
						st := sp.Get("type").String()
						if rr := sp.Get("$ref").String(); rr != "" {
							st = "object"
						}
						if st == "array" {
							it2 := sp.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									st = "array(object)"
								} else if it2.Get("type").String() != "" {
									st = "array(" + it2.Get("type").String() + ")"
								} else {
									st = "array"
								}
							} else {
								st = "array"
							}
						}
						fields = append(fields, FieldInfo{Path: cur + "." + sk, Type: st, Desc: sp.Get("description").String()})
					}
				}
				continue
			}
			if t == "array" {
				it := p.GetJson("items")
				if it != nil {
					if r2 := it.Get("$ref").String(); r2 != "" {
						fields = append(fields, FieldInfo{Path: cur + "[]", Type: "array(object)", Desc: p.Get("description").String()})
						sub := getRefJson(j, r2)
						if sub != nil {
							for sk, sp := range sub.GetJsonMap("properties") {
								st := sp.Get("type").String()
								if rr := sp.Get("$ref").String(); rr != "" {
									st = "object"
								}
								if st == "array" {
									it2 := sp.GetJson("items")
									if it2 != nil {
										if r3 := it2.Get("$ref").String(); r3 != "" {
											st = "array(object)"
										} else if it2.Get("type").String() != "" {
											st = "array(" + it2.Get("type").String() + ")"
										} else {
											st = "array"
										}
									} else {
										st = "array"
									}
								}
								fields = append(fields, FieldInfo{Path: cur + "[]." + sk, Type: st, Desc: sp.Get("description").String()})
							}
						}
						continue
					}
					if it.Get("type").String() == "object" {
						props2 := mergedProperties(j, it)
						k2 := make([]string, 0, len(props2))
						for kk := range props2 {
							k2 = append(k2, kk)
						}
						sortStrings(k2)
						for _, kk := range k2 {
							sp := props2[kk]
							st := sp.Get("type").String()
							if rr := sp.Get("$ref").String(); rr != "" {
								st = "object"
							}
							if st == "array" {
								it3 := sp.GetJson("items")
								if it3 != nil {
									if r3 := it3.Get("$ref").String(); r3 != "" {
										st = "array(object)"
									} else if it3.Get("type").String() != "" {
										st = "array(" + it3.Get("type").String() + ")"
									} else {
										st = "array"
									}
								} else {
									st = "array"
								}
							}
							fields = append(fields, FieldInfo{Path: cur + "[]." + kk, Type: st, Desc: sp.Get("description").String()})
						}
						continue
					}
					t = "array(" + it.Get("type").String() + ")"
				} else {
					t = "array"
				}
			} else if t == "" {
				t = "object"
			}
			fields = append(fields, FieldInfo{Path: cur, Type: t, Desc: p.Get("description").String()})
		}
	}
	var b strings.Builder
	b.WriteString("<table><thead><tr><th>字段</th><th>类型</th><th>说明</th></tr></thead><tbody>")
	if len(fields) == 0 {
		if target.Get("type").String() == "array" {
			it := target.GetJson("items")
			if it != nil {
				props := it.GetJsonMap("properties")
				if len(props) == 0 {
					tt := it.Get("type").String()
					desc := it.Get("description").String()
					if tt != "" {
						b.WriteString("<tr><td>" + htmlEscape(prefix+"[]") + "</td><td>" + htmlEscape("array("+tt+")") + "</td><td>" + htmlEscape(desc) + "</td></tr>")
					} else {
						b.WriteString("<tr><td colspan=3>无字段</td></tr>")
					}
				} else {
					keys := make([]string, 0, len(props))
					for k := range props {
						keys = append(keys, k)
					}
					sortStrings(keys)
					for _, k := range keys {
						p := props[k]
						t := p.Get("type").String()
						if r := p.Get("$ref").String(); r != "" {
							t = "object"
						} else if t == "array" {
							it2 := p.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									t = "array(object)"
								} else if it2.Get("type").String() != "" {
									t = "array(" + it2.Get("type").String() + ")"
								} else {
									t = "array"
								}
							} else {
								t = "array"
							}
						} else if t == "" {
							t = "object"
						}
						desc := p.Get("description").String()
						path := prefix + "[]." + k
						b.WriteString("<tr><td>" + htmlEscape(path) + "</td><td>" + htmlEscape(sanitizeType(t)) + "</td><td>" + htmlEscape(desc) + "</td></tr>")
					}
				}
			} else {
				b.WriteString("<tr><td colspan=3>无字段</td></tr>")
			}
		} else {
			props := target.GetJsonMap("properties")
			if len(props) == 0 {
				b.WriteString("<tr><td colspan=3>无字段</td></tr>")
			} else {
				keys := make([]string, 0, len(props))
				for k := range props {
					keys = append(keys, k)
				}
				sortStrings(keys)
				for _, k := range keys {
					p := props[k]
					t := p.Get("type").String()
					if r := p.Get("$ref").String(); r != "" {
						t = "object"
					} else if t == "array" {
						it2 := p.GetJson("items")
						if it2 != nil {
							if r2 := it2.Get("$ref").String(); r2 != "" {
								t = "array(object)"
							} else if it2.Get("type").String() != "" {
								t = "array(" + it2.Get("type").String() + ")"
							} else {
								t = "array"
							}
						} else {
							t = "array"
						}
					} else if t == "" {
						t = "object"
					}
					desc := p.Get("description").String()
					path := k
					if prefix != "" {
						path = prefix + "." + k
					}
					b.WriteString("<tr><td>" + htmlEscape(path) + "</td><td>" + htmlEscape(sanitizeType(t)) + "</td><td>" + htmlEscape(desc) + "</td></tr>")
				}
			}
		}
	}
	for _, f := range fields {
		b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + htmlEscape(sanitizeType(f.Type)) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

func renderResponseParamFlatTableHTMLFromJsonWithAllowed(j *gjson.Json, sj *gjson.Json, allowed []string) string {
	target := sj
	if !sj.Get("properties.data").IsNil() {
		d := sj.GetJson("properties.data")
		if r := d.Get("$ref").String(); r != "" {
			target = getSchema(j, componentNameFromRef(r))
		} else {
			target = d
		}
	}
	prefix := ""
	if target != sj {
		prefix = "data"
	}
	var fields []FieldInfo
	if prefix != "" {
		tt := target.Get("type").String()
		if tt == "array" {
			it := target.GetJson("items")
			if it != nil {
				if r := it.Get("$ref").String(); r != "" {
					tt = "array(object)"
				} else if it.Get("type").String() != "" {
					tt = "array(" + it.Get("type").String() + ")"
				} else {
					tt = "array"
				}
			} else {
				tt = "array"
			}
		} else if tt == "" {
			tt = "object"
		}
		desc := sj.Get("properties." + prefix + ".description").String()
		fields = append(fields, FieldInfo{Path: prefix, Type: tt, Desc: desc})
	}
	top := mergedProperties(j, sj)
	if len(top) > 0 {
		keysTop := make([]string, 0, len(top))
		for k := range top {
			if k != "data" {
				keysTop = append(keysTop, k)
			}
		}
		sortStrings(keysTop)
		for _, k := range keysTop {
			p := top[k]
			t := p.Get("type").String()
			if r := p.Get("$ref").String(); r != "" {
				t = "object"
			} else if t == "array" {
				it0 := p.GetJson("items")
				if it0 != nil {
					if r0 := it0.Get("$ref").String(); r0 != "" {
						t = "array(object)"
					} else if it0.Get("type").String() != "" {
						t = "array(" + it0.Get("type").String() + ")"
					} else {
						t = "array"
					}
				} else {
					t = "array"
				}
			} else if t == "" {
				t = "object"
			}
			fields = append(fields, FieldInfo{Path: k, Type: t, Desc: p.Get("description").String()})
		}
	}
	if target.Get("type").String() == "array" {
		it := target.GetJson("items")
		if it != nil {
			if r := it.Get("$ref").String(); r != "" {
				sub := getRefJson(j, r)
				if sub != nil {
					for k, p := range mergedProperties(j, sub) {
						t := p.Get("type").String()
						if rr := p.Get("$ref").String(); rr != "" {
							t = "object"
						}
						if t == "array" {
							it2 := p.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									t = "array(object)"
								} else if it2.Get("type").String() != "" {
									t = "array(" + it2.Get("type").String() + ")"
								} else {
									t = "array"
								}
							} else {
								t = "array"
							}
						}
						fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
					}
				}
			} else if it.Get("type").String() == "object" {
				props := mergedProperties(j, it)
				keys := make([]string, 0, len(props))
				for k := range props {
					keys = append(keys, k)
				}
				sortStrings(keys)
				for _, k := range keys {
					p := props[k]
					t := p.Get("type").String()
					if rr := p.Get("$ref").String(); rr != "" {
						t = "object"
					}
					if t == "array" {
						it2 := p.GetJson("items")
						if it2 != nil {
							if r2 := it2.Get("$ref").String(); r2 != "" {
								t = "array(object)"
							} else if it2.Get("type").String() != "" {
								t = "array(" + it2.Get("type").String() + ")"
							} else {
								t = "array"
							}
						} else {
							t = "array"
						}
					} else if t == "" {
						t = "object"
					}
					fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
				}
			} else {
				tt := it.Get("type").String()
				if tt != "" {
					fields = append(fields, FieldInfo{Path: prefix + "[]", Type: "array(" + tt + ")", Desc: it.Get("description").String()})
				}
			}
		}
	} else {
		props := mergedProperties(j, target)
		keys := make([]string, 0, len(props))
		for k := range props {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			p := props[k]
			cur := k
			if prefix != "" {
				cur = prefix + "." + k
			}
			t := p.Get("type").String()
			if r := p.Get("$ref").String(); r != "" {
				fields = append(fields, FieldInfo{Path: cur, Type: "object", Desc: p.Get("description").String()})
				sub := getRefJson(j, r)
				if sub != nil {
					for sk, sp := range mergedProperties(j, sub) {
						st := sp.Get("type").String()
						if rr := sp.Get("$ref").String(); rr != "" {
							st = "object"
						}
						if st == "array" {
							it2 := sp.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									st = "array(object)"
								} else if it2.Get("type").String() != "" {
									st = "array(" + it2.Get("type").String() + ")"
								} else {
									st = "array"
								}
							} else {
								st = "array"
							}
						}
						fields = append(fields, FieldInfo{Path: cur + "." + sk, Type: st, Desc: sp.Get("description").String()})
					}
				}
				continue
			}
			if t == "array" {
				it := p.GetJson("items")
				if it != nil {
					if r2 := it.Get("$ref").String(); r2 != "" {
						fields = append(fields, FieldInfo{Path: cur + "[]", Type: "array(object)", Desc: p.Get("description").String()})
						sub := getRefJson(j, r2)
						if sub != nil {
							for sk, sp := range sub.GetJsonMap("properties") {
								st := sp.Get("type").String()
								if rr := sp.Get("$ref").String(); rr != "" {
									st = "object"
								}
								if st == "array" {
									it2 := sp.GetJson("items")
									if it2 != nil {
										if r3 := it2.Get("$ref").String(); r3 != "" {
											st = "array(object)"
										} else if it2.Get("type").String() != "" {
											st = "array(" + it2.Get("type").String() + ")"
										} else {
											st = "array"
										}
									} else {
										st = "array"
									}
								}
								fields = append(fields, FieldInfo{Path: cur + "[]." + sk, Type: st, Desc: sp.Get("description").String()})
							}
						}
						continue
					}
					if it.Get("type").String() == "object" {
						props2 := it.GetJsonMap("properties")
						k2 := make([]string, 0, len(props2))
						for kk := range props2 {
							k2 = append(k2, kk)
						}
						sortStrings(k2)
						for _, kk := range k2 {
							sp := props2[kk]
							st := sp.Get("type").String()
							if rr := sp.Get("$ref").String(); rr != "" {
								st = "object"
							}
							if st == "array" {
								it3 := sp.GetJson("items")
								if it3 != nil {
									if r3 := it3.Get("$ref").String(); r3 != "" {
										st = "array(object)"
									} else if it3.Get("type").String() != "" {
										st = "array(" + it3.Get("type").String() + ")"
									} else {
										st = "array"
									}
								} else {
									st = "array"
								}
							}
							fields = append(fields, FieldInfo{Path: cur + "[]." + kk, Type: st, Desc: sp.Get("description").String()})
						}
						continue
					}
					t = "array(" + it.Get("type").String() + ")"
				} else {
					t = "array"
				}
			} else if t == "" {
				t = "object"
			}
			fields = append(fields, FieldInfo{Path: cur, Type: t, Desc: p.Get("description").String()})
		}
	}
	fields = filterFieldInfos(fields, allowed)
	var b strings.Builder
	b.WriteString("<table><thead><tr><th>字段</th><th>类型</th><th>说明</th></tr></thead><tbody>")
	if len(fields) == 0 {
		b.WriteString("<tr><td colspan=3>无字段</td></tr>")
	}
	for _, f := range fields {
		b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + htmlEscape(sanitizeType(f.Type)) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

// renderResponseParamFlatTableHTMLFromRef 渲染返回参数说明为 HTML（$ref）。
func renderResponseParamFlatTableHTMLFromRef(j *gjson.Json, ref string) string {
	sj := getSchema(j, componentNameFromRef(ref))
	if sj == nil {
		return ""
	}
	return renderResponseParamFlatTableHTMLFromJson(j, sj)
}

// renderResponseParamFlatTableMarkdownFromJson 渲染返回参数说明为 Markdown（自动展开 data）。
func renderResponseParamFlatTableMarkdownFromJson(j *gjson.Json, sj *gjson.Json) string {
	target := sj
	prefix := ""
	for _, key := range []string{"data", "payload", "result", "content"} {
		if !sj.Get("properties." + key).IsNil() {
			d := sj.GetJson("properties." + key)
			if r := d.Get("$ref").String(); r != "" {
				target = getSchema(j, componentNameFromRef(r))
			} else {
				target = d
			}
			if target != sj {
				prefix = key
			}
			break
		}
	}
	var fields []FieldInfo
	if prefix != "" {
		t := target.Get("type").String()
		if t == "array" {
			it := target.GetJson("items")
			if it != nil {
				if r := it.Get("$ref").String(); r != "" {
					t = "array(object)"
				} else if it.Get("type").String() != "" {
					t = "array(" + it.Get("type").String() + ")"
				} else {
					t = "array"
				}
			} else {
				t = "array"
			}
		} else if t == "" {
			t = "object"
		}
		desc := sj.Get("properties." + prefix + ".description").String()
		fields = append(fields, FieldInfo{Path: prefix, Type: t, Desc: desc})
	}
	top := mergedProperties(j, sj)
	if len(top) > 0 {
		keysTop := make([]string, 0, len(top))
		for k := range top {
			if prefix == "" || k != prefix {
				keysTop = append(keysTop, k)
			}
		}
		sortStrings(keysTop)
		for _, k := range keysTop {
			p := top[k]
			t := p.Get("type").String()
			if r := p.Get("$ref").String(); r != "" {
				t = "object"
			}
			if t == "array" {
				it0 := p.GetJson("items")
				if it0 != nil {
					if r0 := it0.Get("$ref").String(); r0 != "" {
						t = "array(object)"
					} else if it0.Get("type").String() != "" {
						t = "array(" + it0.Get("type").String() + ")"
					} else {
						t = "array"
					}
				} else {
					t = "array"
				}
			} else if t == "" {
				t = "object"
			}
			fields = append(fields, FieldInfo{Path: k, Type: t, Desc: p.Get("description").String()})
		}
	}
	if target.Get("type").String() == "array" {
		it := target.GetJson("items")
		if it != nil {
			if r := it.Get("$ref").String(); r != "" {
				sub := getRefJson(j, r)
				if sub != nil {
					for k, p := range mergedProperties(j, sub) {
						t := p.Get("type").String()
						if rr := p.Get("$ref").String(); rr != "" {
							t = "object"
						}
						if t == "array" {
							it2 := p.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									t = "array(object)"
								} else if it2.Get("type").String() != "" {
									t = "array(" + it2.Get("type").String() + ")"
								} else {
									t = "array"
								}
							} else {
								t = "array"
							}
						}
						fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
					}
				}
			} else if it.Get("type").String() == "object" {
				props := mergedProperties(j, it)
				keys := make([]string, 0, len(props))
				for k := range props {
					keys = append(keys, k)
				}
				sortStrings(keys)
				for _, k := range keys {
					p := props[k]
					t := p.Get("type").String()
					if rr := p.Get("$ref").String(); rr != "" {
						t = "object"
					}
					if t == "array" {
						it2 := p.GetJson("items")
						if it2 != nil {
							if r2 := it2.Get("$ref").String(); r2 != "" {
								t = "array(object)"
							} else if it2.Get("type").String() != "" {
								t = "array(" + it2.Get("type").String() + ")"
							} else {
								t = "array"
							}
						} else {
							t = "array"
						}
					} else if t == "" {
						t = "object"
					}
					fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
				}
			} else {
				tt := it.Get("type").String()
				if tt != "" {
					fields = append(fields, FieldInfo{Path: prefix + "[]", Type: "array(" + tt + ")", Desc: it.Get("description").String()})
				}
			}
		}
	} else {
		props := mergedProperties(j, target)
		keys := make([]string, 0, len(props))
		for k := range props {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			p := props[k]
			cur := k
			if prefix != "" {
				cur = prefix + "." + k
			}
			t := p.Get("type").String()
			if r := p.Get("$ref").String(); r != "" {
				fields = append(fields, FieldInfo{Path: cur, Type: "object", Desc: p.Get("description").String()})
				sub := getRefJson(j, r)
				if sub != nil {
					for sk, sp := range mergedProperties(j, sub) {
						st := sp.Get("type").String()
						if rr := sp.Get("$ref").String(); rr != "" {
							st = "object"
						}
						if st == "array" {
							it2 := sp.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									st = "array(object)"
								} else if it2.Get("type").String() != "" {
									st = "array(" + it2.Get("type").String() + ")"
								} else {
									st = "array"
								}
							} else {
								st = "array"
							}
						}
						fields = append(fields, FieldInfo{Path: cur + "." + sk, Type: st, Desc: sp.Get("description").String()})
					}
				}
				continue
			}
			if t == "array" {
				it := p.GetJson("items")
				if it != nil {
					if r2 := it.Get("$ref").String(); r2 != "" {
						fields = append(fields, FieldInfo{Path: cur + "[]", Type: "array(object)", Desc: p.Get("description").String()})
						sub := getRefJson(j, r2)
						if sub != nil {
							for sk, sp := range mergedProperties(j, sub) {
								st := sp.Get("type").String()
								if rr := sp.Get("$ref").String(); rr != "" {
									st = "object"
								}
								if st == "array" {
									it2 := sp.GetJson("items")
									if it2 != nil {
										if r3 := it2.Get("$ref").String(); r3 != "" {
											st = "array(object)"
										} else if it2.Get("type").String() != "" {
											st = "array(" + it2.Get("type").String() + ")"
										} else {
											st = "array"
										}
									} else {
										st = "array"
									}
								}
								fields = append(fields, FieldInfo{Path: cur + "[]." + sk, Type: st, Desc: sp.Get("description").String()})
							}
						}
						continue
					}
					if it.Get("type").String() == "object" {
						props2 := mergedProperties(j, it)
						k2 := make([]string, 0, len(props2))
						for kk := range props2 {
							k2 = append(k2, kk)
						}
						sortStrings(k2)
						for _, kk := range k2 {
							sp := props2[kk]
							st := sp.Get("type").String()
							if rr := sp.Get("$ref").String(); rr != "" {
								st = "object"
							}
							if st == "array" {
								it3 := sp.GetJson("items")
								if it3 != nil {
									if r3 := it3.Get("$ref").String(); r3 != "" {
										st = "array(object)"
									} else if it3.Get("type").String() != "" {
										st = "array(" + it3.Get("type").String() + ")"
									} else {
										st = "array"
									}
								} else {
									st = "array"
								}
							}
							fields = append(fields, FieldInfo{Path: cur + "[]." + kk, Type: st, Desc: sp.Get("description").String()})
						}
						continue
					}
					t = "array(" + it.Get("type").String() + ")"
				} else {
					t = "array"
				}
			} else if t == "" {
				t = "object"
			}
			fields = append(fields, FieldInfo{Path: cur, Type: t, Desc: p.Get("description").String()})
		}
	}
	var b2 strings.Builder
	b2.WriteString("<table><thead><tr><th>字段</th><th>类型</th><th>说明</th></tr></thead><tbody>")
	if len(fields) == 0 {
		b2.WriteString("<tr><td colspan=3>无字段</td></tr>")
	}
	for _, f := range fields {
		b2.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td></tr>", htmlEscape(f.Path), htmlEscape(sanitizeType(f.Type)), htmlEscape(f.Desc)))
	}
	b2.WriteString("</tbody></table>")
	return b2.String()
}

// renderResponseParamFlatTableMarkdownFromRef 渲染返回参数说明为 Markdown（$ref）。
func renderResponseParamFlatTableMarkdownFromRef(j *gjson.Json, ref string) string {
	sj := getSchema(j, componentNameFromRef(ref))
	if sj == nil {
		return ""
	}
	return renderResponseParamFlatTableMarkdownFromJson(j, sj)
}

// renderResponseParamFlatTableMarkdownFromJsonWithAllowed 渲染“返回参数说明”为 Markdown 表格（内联 schema），
// 首先输出容器行 data，随后展开并按 allowed 白名单过滤保留的 data 子字段；
// 顶层字段 code/message/data 始终保留。
func renderResponseParamFlatTableMarkdownFromJsonWithAllowed(j *gjson.Json, sj *gjson.Json, allowed []string) string {
	target := sj
	if !sj.Get("properties.data").IsNil() {
		d := sj.GetJson("properties.data")
		if r := d.Get("$ref").String(); r != "" {
			target = getSchema(j, componentNameFromRef(r))
		} else {
			target = d
		}
	}
	prefix := ""
	if target != sj {
		prefix = "data"
	}
	var fields []FieldInfo
	if prefix != "" {
		t := target.Get("type").String()
		if t == "array" {
			it := target.GetJson("items")
			if it != nil {
				if r := it.Get("$ref").String(); r != "" {
					t = "array(object)"
				} else if it.Get("type").String() != "" {
					t = "array(" + it.Get("type").String() + ")"
				} else {
					t = "array"
				}
			} else {
				t = "array"
			}
		} else if t == "" {
			t = "object"
		}
		desc := sj.Get("properties." + prefix + ".description").String()
		fields = append(fields, FieldInfo{Path: prefix, Type: t, Desc: desc})
	}
	top := sj.GetJsonMap("properties")
	if len(top) > 0 {
		keysTop := make([]string, 0, len(top))
		for k := range top {
			if k != "data" {
				keysTop = append(keysTop, k)
			}
		}
		sortStrings(keysTop)
		for _, k := range keysTop {
			p := top[k]
			t := p.Get("type").String()
			if r := p.Get("$ref").String(); r != "" {
				t = "object"
			} else if t == "array" {
				it0 := p.GetJson("items")
				if it0 != nil {
					if r0 := it0.Get("$ref").String(); r0 != "" {
						t = "array(object)"
					} else if it0.Get("type").String() != "" {
						t = "array(" + it0.Get("type").String() + ")"
					} else {
						t = "array"
					}
				} else {
					t = "array"
				}
			} else if t == "" {
				t = "object"
			}
			fields = append(fields, FieldInfo{Path: k, Type: t, Desc: p.Get("description").String()})
		}
	}
	if target.Get("type").String() == "array" {
		it := target.GetJson("items")
		if it != nil {
			if r := it.Get("$ref").String(); r != "" {
				sub := getRefJson(j, r)
				if sub != nil {
					for k, p := range sub.GetJsonMap("properties") {
						t := p.Get("type").String()
						if rr := p.Get("$ref").String(); rr != "" {
							t = "object"
						}
						if t == "array" {
							it2 := p.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									t = "array(object)"
								} else if it2.Get("type").String() != "" {
									t = "array(" + it2.Get("type").String() + ")"
								} else {
									t = "array"
								}
							} else {
								t = "array"
							}
						}
						fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
					}
				}
			} else if it.Get("type").String() == "object" {
				props := it.GetJsonMap("properties")
				keys := make([]string, 0, len(props))
				for k := range props {
					keys = append(keys, k)
				}
				sortStrings(keys)
				for _, k := range keys {
					p := props[k]
					t := p.Get("type").String()
					if rr := p.Get("$ref").String(); rr != "" {
						t = "object"
					}
					if t == "array" {
						it2 := p.GetJson("items")
						if it2 != nil {
							if r2 := it2.Get("$ref").String(); r2 != "" {
								t = "array(object)"
							} else if it2.Get("type").String() != "" {
								t = "array(" + it2.Get("type").String() + ")"
							} else {
								t = "array"
							}
						} else {
							t = "array"
						}
					} else if t == "" {
						t = "object"
					}
					fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
				}
			} else {
				tt := it.Get("type").String()
				if tt != "" {
					fields = append(fields, FieldInfo{Path: prefix + "[]", Type: "array(" + tt + ")", Desc: it.Get("description").String()})
				}
			}
		}
	} else {
		props := target.GetJsonMap("properties")
		keys := make([]string, 0, len(props))
		for k := range props {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			p := props[k]
			cur := k
			if prefix != "" {
				cur = prefix + "." + k
			}
			t := p.Get("type").String()
			if r := p.Get("$ref").String(); r != "" {
				fields = append(fields, FieldInfo{Path: cur, Type: "object", Desc: p.Get("description").String()})
				sub := getRefJson(j, r)
				if sub != nil {
					for sk, sp := range sub.GetJsonMap("properties") {
						st := sp.Get("type").String()
						if rr := sp.Get("$ref").String(); rr != "" {
							st = "object"
						}
						if st == "array" {
							it2 := sp.GetJson("items")
							if it2 != nil {
								if r2 := it2.Get("$ref").String(); r2 != "" {
									st = "array(object)"
								} else if it2.Get("type").String() != "" {
									st = "array(" + it2.Get("type").String() + ")"
								} else {
									st = "array"
								}
							} else {
								st = "array"
							}
						}
						fields = append(fields, FieldInfo{Path: cur + "." + sk, Type: st, Desc: sp.Get("description").String()})
					}
				}
				continue
			}
			if t == "array" {
				it := p.GetJson("items")
				if it != nil {
					if r2 := it.Get("$ref").String(); r2 != "" {
						fields = append(fields, FieldInfo{Path: cur + "[]", Type: "array(object)", Desc: p.Get("description").String()})
						sub := getRefJson(j, r2)
						if sub != nil {
							for sk, sp := range sub.GetJsonMap("properties") {
								st := sp.Get("type").String()
								if rr := sp.Get("$ref").String(); rr != "" {
									st = "object"
								}
								if st == "array" {
									it2 := sp.GetJson("items")
									if it2 != nil {
										if r3 := it2.Get("$ref").String(); r3 != "" {
											st = "array(object)"
										} else if it2.Get("type").String() != "" {
											st = "array(" + it2.Get("type").String() + ")"
										} else {
											st = "array"
										}
									} else {
										st = "array"
									}
								}
								fields = append(fields, FieldInfo{Path: cur + "[]." + sk, Type: st, Desc: sp.Get("description").String()})
							}
						}
						continue
					}
					if it.Get("type").String() == "object" {
						props2 := it.GetJsonMap("properties")
						k2 := make([]string, 0, len(props2))
						for kk := range props2 {
							k2 = append(k2, kk)
						}
						sortStrings(k2)
						for _, kk := range k2 {
							sp := props2[kk]
							st := sp.Get("type").String()
							if rr := sp.Get("$ref").String(); rr != "" {
								st = "object"
							}
							if st == "array" {
								it3 := sp.GetJson("items")
								if it3 != nil {
									if r3 := it3.Get("$ref").String(); r3 != "" {
										st = "array(object)"
									} else if it3.Get("type").String() != "" {
										st = "array(" + it3.Get("type").String() + ")"
									} else {
										st = "array"
									}
								} else {
									st = "array"
								}
							}
							fields = append(fields, FieldInfo{Path: cur + "[]." + kk, Type: st, Desc: sp.Get("description").String()})
						}
						continue
					}
					t = "array(" + it.Get("type").String() + ")"
				} else {
					t = "array"
				}
			} else if t == "" {
				t = "object"
			}
			fields = append(fields, FieldInfo{Path: cur, Type: t, Desc: p.Get("description").String()})
		}
	}
	fields = filterFieldInfos(fields, allowed)
	var b strings.Builder
	b.WriteString("| 字段 | 类型 | 说明 |\n|---|---|---|\n")
	if len(fields) == 0 {
		b.WriteString("| 无字段 |  |  |\n")
	}
	for _, f := range fields {
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", f.Path, sanitizeType(f.Type), f.Desc))
	}
	return b.String()
}

// flattenSchemaFields 基于 $ref 展开对象的所有子字段到扁平行（请求参数）。
func flattenSchemaFields(j *gjson.Json, ref string, prefix string) []FieldInfo {
	name := componentNameFromRef(ref)
	sj := getSchema(j, name)
	if sj == nil {
		return nil
	}
	fields := make([]FieldInfo, 0, 16)
	props := sj.GetJsonMap("properties")
	requiredSet := setFromArray(sj.Get("required").Array())
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		pj := props[k]
		req := requiredSet[k]
		typ := pj.Get("type").String()
		desc := titleDescription(pj)
		ref2 := pj.Get("$ref").String()
		curPath := k
		if prefix != "" {
			curPath = prefix + "." + k
		}
		if typ == "array" {
			it := pj.GetJson("items")
			if it != nil {
				if r := it.Get("$ref").String(); r != "" {
					fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(object)", Desc: desc})
					sub := flattenSchemaFields(j, r, curPath+"[]")
					fields = append(fields, sub...)
					continue
				}
				inlineObj := it.Get("type").String() == "object" || len(it.GetJsonMap("properties")) > 0 || len(it.Get("allOf").Array()) > 0 || len(it.Get("oneOf").Array()) > 0 || len(it.Get("anyOf").Array()) > 0
				if inlineObj {
					fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(object)", Desc: desc})
					props2 := mergedProperties(j, it)
					itemReq := setFromArray(it.Get("required").Array())
					k2 := make([]string, 0, len(props2))
					for kk := range props2 {
						k2 = append(k2, kk)
					}
					sortStrings(k2)
					for _, kk := range k2 {
						sp := props2[kk]
						st := sp.Get("type").String()
						if rr := sp.Get("$ref").String(); rr != "" {
							st = "object"
						}
						if st == "array" {
							it3 := sp.GetJson("items")
							if it3 != nil {
								if r3 := it3.Get("$ref").String(); r3 != "" {
									st = "array(object)"
								} else if it3.Get("type").String() != "" {
									st = "array(" + it3.Get("type").String() + ")"
								} else {
									st = "array"
								}
							} else {
								st = "array"
							}
						} else if st == "" {
							st = "object"
						}
						fields = append(fields, FieldInfo{Path: curPath + "[]." + kk, Required: itemReq[kk], Type: st, Desc: titleDescription(sp)})
					}
					continue
				}
				fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(" + it.Get("type").String() + ")", Desc: desc})
				continue
			}
			fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array", Desc: desc})
			continue
		}
		if ref2 != "" {
			fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: "object", Desc: desc})
			sub := flattenSchemaFields(j, ref2, curPath)
			fields = append(fields, sub...)
			continue
		}
		if typ == "object" {
			fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: "object", Desc: desc})
			nestedProps := pj.GetJsonMap("properties")
			if len(nestedProps) > 0 {
				nestedReq := setFromArray(pj.Get("required").Array())
				nkeys := make([]string, 0, len(nestedProps))
				for nk := range nestedProps {
					nkeys = append(nkeys, nk)
				}
				sortStrings(nkeys)
				for _, nk := range nkeys {
					np := nestedProps[nk]
					nreq := nestedReq[nk]
					ntyp := np.Get("type").String()
					ndesc := titleDescription(np)
					nref := np.Get("$ref").String()
					npath := curPath + "." + nk
					if ntyp == "array" {
						it := np.GetJson("items")
						if it != nil {
							if r := it.Get("$ref").String(); r != "" {
								fields = append(fields, FieldInfo{Path: npath + "[]", Required: nreq, Type: "array(object)", Desc: ndesc})
								sub := flattenSchemaFields(j, r, npath+"[]")
								fields = append(fields, sub...)
								continue
							}
							fields = append(fields, FieldInfo{Path: npath + "[]", Required: nreq, Type: "array(" + it.Get("type").String() + ")", Desc: ndesc})
							continue
						}
						fields = append(fields, FieldInfo{Path: npath + "[]", Required: nreq, Type: "array", Desc: ndesc})
						continue
					}
					if nref != "" {
						fields = append(fields, FieldInfo{Path: npath, Required: nreq, Type: "object", Desc: ndesc})
						sub := flattenSchemaFields(j, nref, npath)
						fields = append(fields, sub...)
						continue
					}
					fields = append(fields, FieldInfo{Path: npath, Required: nreq, Type: ntyp, Desc: ndesc})
				}
			}
			continue
		}
		fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: typ, Desc: desc})
	}
	return fields
}

// flattenSchemaFieldsFromJson 从内联 schema 展开到扁平行（请求/返回参数）。
func flattenSchemaFieldsFromJson(j *gjson.Json, sj *gjson.Json, prefix string) []FieldInfo {
	if sj == nil {
		return nil
	}
	fields := make([]FieldInfo, 0, 16)
	props := sj.GetJsonMap("properties")
	if len(props) == 0 {
		for _, arrName := range []string{"allOf", "oneOf", "anyOf"} {
			for _, it := range sj.Get(arrName).Array() {
				elem := gjson.New(it)
				if r := elem.Get("$ref").String(); r != "" {
					sub := getRefJson(j, r)
					if sub != nil {
						for k, v := range sub.GetJsonMap("properties") {
							props[k] = v
						}
					}
				} else {
					for k, v := range elem.GetJsonMap("properties") {
						props[k] = v
					}
				}
			}
		}
	}
	requiredSet := setFromArray(sj.Get("required").Array())
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		pj := props[k]
		req := requiredSet[k]
		typ := pj.Get("type").String()
		desc := titleDescription(pj)
		ref2 := pj.Get("$ref").String()
		curPath := k
		if prefix != "" {
			curPath = prefix + "." + k
		}
		if typ == "array" {
			it := pj.GetJson("items")
			if it != nil {
				if r := it.Get("$ref").String(); r != "" {
					fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(object)", Desc: desc})
					sub := getRefJson(j, r)
					if sub != nil {
						fields = append(fields, flattenSchemaFieldsFromJson(j, sub, curPath+"[]")...)
					}
					continue
				}
				inlineObj := it.Get("type").String() == "object" || len(it.GetJsonMap("properties")) > 0 || len(it.Get("allOf").Array()) > 0 || len(it.Get("oneOf").Array()) > 0 || len(it.Get("anyOf").Array()) > 0
				if inlineObj {
					fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(object)", Desc: desc})
					props2 := mergedProperties(j, it)
					itemReq := setFromArray(it.Get("required").Array())
					k2 := make([]string, 0, len(props2))
					for kk := range props2 {
						k2 = append(k2, kk)
					}
					sortStrings(k2)
					for _, kk := range k2 {
						sp := props2[kk]
						st := sp.Get("type").String()
						if rr := sp.Get("$ref").String(); rr != "" {
							st = "object"
						}
						if st == "array" {
							it3 := sp.GetJson("items")
							if it3 != nil {
								if r3 := it3.Get("$ref").String(); r3 != "" {
									st = "array(object)"
								} else if it3.Get("type").String() != "" {
									st = "array(" + it3.Get("type").String() + ")"
								} else {
									st = "array"
								}
							} else {
								st = "array"
							}
						} else if st == "" {
							st = "object"
						}
						fields = append(fields, FieldInfo{Path: curPath + "[]." + kk, Required: itemReq[kk], Type: st, Desc: titleDescription(sp)})
					}
					continue
				}
				fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(" + it.Get("type").String() + ")", Desc: desc})
				continue
			}
			fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array", Desc: desc})
			continue
		}
		if ref2 != "" {
			fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: "object", Desc: desc})
			sub := getRefJson(j, ref2)
			if sub != nil {
				fields = append(fields, flattenSchemaFieldsFromJson(j, sub, curPath)...)
			}
			continue
		}
		if typ == "object" {
			fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: "object", Desc: desc})
			fields = append(fields, flattenSchemaFieldsFromJson(j, pj, curPath)...)
			continue
		}
		fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: typ, Desc: desc})
	}
	return fields
}
