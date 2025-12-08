package render

import (
	"html/template"
	"strings"

	"github.com/megatrZlp/go-apidocs/apidocs/source"
	"github.com/megatrZlp/go-apidocs/apidocs/tools"

	"github.com/gogf/gf/v2/encoding/gjson"
)

// CustomizeReqAndRes 定义“按接口路径”的注入 Header 与请求/返回白名单规则。
type CustomizeReqAndRes struct {
	Headers  map[string]string
	Request  []string
	Response []string
}

// RenderConfig 渲染配置：包含外层路由配置映射与模板路径。
type RenderConfig struct {
	RouteMarkdown string
	Customize     map[string]CustomizeReqAndRes
	TemplateDir   string
}

// GenerateHTML 生成完整的 HTML 文档页面。
// 行为说明：
// - 读取 OpenAPI 的 info、paths、tags、requestBody、responses 等字段；
// - 侧边导航使用 OrderedPathsFromContent(contentRaw) 保持 paths 原始顺序；
// - 每个分组（一级/二级）与接口块都会生成可链接的锚点；
// - 请求/响应参数说明支持 $ref 与内联 schema，同时递归处理数组 items；
// - 页面包含最简样式与交互：展开/收起导航；滚动时菜单联动高亮。
func GenerateHTML(j *gjson.Json, contentRaw string) string {
	cfg := RenderConfig{}
	return GenerateHTMLWithConfig(j, contentRaw, cfg)
}

// applyCustomizeHeaders 为指定路径注入自定义 Header 参数，避免重复。
func applyCustomizeHeaders(path string, headers []paramInfo, cfg RenderConfig) []paramInfo {
	rules := customizeForPath(path, cfg)
	if len(rules.Headers) == 0 {
		return headers
	}
	exists := make(map[string]struct{}, len(headers))
	for _, h := range headers {
		exists[h.Name] = struct{}{}
	}
	for name, typ := range rules.Headers {
		if _, ok := exists[name]; ok {
			continue
		}
		t, req, desc := parseHeaderSpec(typ)
		headers = append(headers, paramInfo{Name: name, Required: req, Type: t, Desc: desc})
	}
	return headers
}

// parseHeaderSpec 解析自定义 Header 规格：形如 "type#required#desc" 或 "type#desc"。
func parseHeaderSpec(spec string) (typ string, required string, desc string) {
	parts := strings.Split(spec, "#")
	typ = ""
	required = "否"
	desc = ""
	if len(parts) > 0 {
		typ = parts[0]
	}
	if len(parts) >= 2 {
		p1 := strings.ToLower(strings.TrimSpace(parts[1]))
		if p1 == "required" || p1 == "必选" {
			required = "是"
			if len(parts) >= 3 {
				desc = parts[2]
			}
		} else if p1 == "optional" || p1 == "可选" {
			required = "否"
			if len(parts) >= 3 {
				desc = parts[2]
			} else {
				desc = parts[1]
			}
		} else {
			desc = parts[1]
		}
	}
	if len(parts) >= 3 && desc == "" {
		desc = parts[2]
	}
	return typ, required, desc
}

// pathLikeMatch 支持 * 通配的路径匹配。
func pathLikeMatch(pattern, path string) bool {
	if pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "*") {
		return pattern == path
	}
	segs := strings.Split(pattern, "*")
	i := 0
	for _, s := range segs {
		if s == "" {
			continue
		}
		idx := strings.Index(path[i:], s)
		if idx < 0 {
			return false
		}
		i += idx + len(s)
	}
	return true
}

// uniqueMerge 合并两个字符串切片，跳过空项并去重，保持原有顺序（后者追加到前者）。
func uniqueMerge(a []string, b []string) []string {
	if len(b) == 0 {
		return a
	}
	m := make(map[string]struct{}, len(a)+len(b))
	for _, x := range a {
		if x == "" {
			continue
		}
		m[x] = struct{}{}
	}
	for _, x := range b {
		if x == "" {
			continue
		}
		if _, ok := m[x]; !ok {
			a = append(a, x)
			m[x] = struct{}{}
		}
	}
	return a
}

// customizeForPath 根据路径匹配（支持 * 通配）聚合多个规则，返回合并后的 Headers/Request/Response 白名单。
func customizeForPath(path string, cfg RenderConfig) CustomizeReqAndRes {
	res := CustomizeReqAndRes{Headers: map[string]string{}}
	for pat, r := range cfg.Customize {
		if !pathLikeMatch(pat, path) {
			continue
		}
		if r.Headers != nil {
			for k, v := range r.Headers {
				if _, ok := res.Headers[k]; !ok {
					res.Headers[k] = v
				}
			}
		}
		if r.Request != nil {
			res.Request = uniqueMerge(res.Request, r.Request)
		}
		if r.Response != nil {
			res.Response = uniqueMerge(res.Response, r.Response)
		}
	}
	return res
}

// GenerateHTMLWithConfig 与 GenerateHTML 一致，但支持 RenderConfig.Customize 的 Header 注入与 Req/Res 过滤。
func GenerateHTMLWithConfig(j *gjson.Json, contentRaw string, cfg RenderConfig) string {
	// 标题：优先使用 OpenAPI info.title，缺省为 “API 文档”
	title := strings.TrimSpace(j.Get("info.title").String())
	if title == "" {
		title = "API 文档"
	}
	// 构建模板，若失败则采用降级路径输出布局 HTML
	t, terr := buildLayoutTemplate(cfg.TemplateDir)
	// paths 结构与其键原始顺序（通过流式解析 raw 内容保证菜单与正文一致）
	paths := j.GetJsonMap("paths")
	keys := source.OrderedPathsFromContent(contentRaw)
	groups := make(map[string]map[string][][2]string)
	pfxOrder := make([]string, 0, len(keys))
	sfxOrder := make(map[string][]string)
	// 遍历所有路径，按 tags[0] 的“主/次”分组，将 (summary, anchor) 聚合到分组树
	for _, p := range keys {
		pj := paths[p]
		methods := presentMethods(pj)
		for _, m := range methods {
			mj := pj.GetJson(m)
			if mj == nil {
				continue
			}
			tags := jsonArrayStrings(mj.Get("tags").Array())
			pre, suf := splitTagParts(tags)
			summary := strings.TrimSpace(mj.Get("summary").String())
			if summary == "" {
				summary = strings.ToUpper(m) + " " + p
			}
			anchor := anchorID(m, p)
			if _, ok := groups[pre]; !ok {
				groups[pre] = make(map[string][][2]string)
				pfxOrder = append(pfxOrder, pre)
			}
			if _, ok := groups[pre][suf]; !ok {
				groups[pre][suf] = make([][2]string, 0, 8)
				sfxOrder[pre] = append(sfxOrder[pre], suf)
			}
			groups[pre][suf] = append(groups[pre][suf], [2]string{summary, anchor})
		}
	}
	// 侧边导航视图模型（按主/次分组的树结构）
	navVM := buildTopNavGroups(pfxOrder, sfxOrder, groups)
	var bn strings.Builder
	if terr == nil {
		_ = t.ExecuteTemplate(&bn, "nav", NavData{Groups: navVM})
	}
	var bm strings.Builder
	mdRoute := cfg.RouteMarkdown
	if mdRoute == "" {
		mdRoute = "/docs.md"
	}
	if terr == nil {
		_ = t.ExecuteTemplate(&bm, "main_header", MainHeaderData{Title: tools.HTMLEscape(title), MdRoute: tools.HTMLEscape(mdRoute)})
	}
	var currPre string
	emittedSub := make(map[string]bool)
	// 主体正文：按菜单顺序生成分组标题与接口块
	for _, p := range keys {
		pj := paths[p]
		methods := presentMethods(pj)
		for _, m := range methods {
			mj := pj.GetJson(m)
			if mj == nil {
				continue
			}
			summary := strings.TrimSpace(mj.Get("summary").String())
			tags := tools.JSONArrayStrings(mj.Get("tags").Array())
			_, reqCT, reqRef := getRequestSchema(j, mj)
			_, _, resRef := getResponseSchema(j, mj)
			pre, _ := tools.SplitTagParts(tags)
			if pre != currPre {
				currPre = pre
				if terr == nil {
					_ = t.ExecuteTemplate(&bm, "group_heading", GroupHeadingData{Id: "group-" + slugify(pre), Name: htmlEscape(pre)})
				}
			}
			if terr == nil {
				_, suf := tools.SplitTagParts(tags)
				childAcc := "group-" + tools.Slugify(pre) + "-" + tools.Slugify(suf)
				if _, ok := emittedSub[childAcc]; !ok {
					// 次级分组标题（只输出一次）
					_ = t.ExecuteTemplate(&bm, "sub_heading", SubHeadingData{Id: childAcc, Name: htmlEscape(suf)})
					emittedSub[childAcc] = true
				}
			}
			anchor := tools.AnchorID(m, p)
			ct := reqCT
			if ct == "" {
				ct = "application/json"
			}
			// 合并 path/op 两层 parameters 并按 in 分类
			headers, pathsParams, queryParams := collectParameters(j, pj, mj)
			headers = applyCustomizeHeaders(p, headers, cfg)
			// 请求示例与参数表（优先内联 schema，其次 $ref）
			reqSchema, _, reqRef2 := getRequestSchema(j, mj)
			if reqRef == "" {
				reqRef = reqRef2
			}
			allowedReq := cfg.Customize[p].Request
			var reqExample string
			var reqTable string
			if reqSchema != nil {
				if allowedReq == nil {
					reqExample = exampleJSONFromSchema(j, reqSchema)
					reqTable = renderParamTableHTMLFromJson(j, reqSchema)
				} else {
					reqExample = exampleJSONFromSchemaWithAllowed(j, reqSchema, allowedReq)
					reqTable = renderParamTableHTMLFromJsonWithAllowed(j, reqSchema, allowedReq)
				}
			} else if reqRef != "" {
				if allowedReq == nil {
					reqExample = exampleJSON(j, reqRef)
					reqTable = renderParamTableHTML(j, reqRef)
				} else {
					reqExample = exampleJSONWithAllowed(j, reqRef, allowedReq)
					reqTable = renderParamTableHTMLWithAllowed(j, reqRef, allowedReq)
				}
			}
			// 返回示例与参数表（优先 200，首选 JSON 媒体类型）
			resSchema, _, resRef2 := getResponseSchema(j, mj)
			if resRef == "" {
				resRef = resRef2
			}
			allowedRes := cfg.Customize[p].Response
			var resExample string
			var resTable string
			if resSchema != nil {
				if allowedRes == nil {
					resExample = exampleJSONFromSchema(j, resSchema)
					resTable = tools.StripComponentTypeDecorations(renderResponseParamFlatTableHTMLFromJson(j, resSchema))
				} else {
					resExample = exampleJSONFromSchemaWithAllowed(j, resSchema, allowedRes)
					resTable = tools.StripComponentTypeDecorations(renderResponseParamFlatTableHTMLFromJsonWithAllowed(j, resSchema, allowedRes))
				}
			} else if resRef != "" {
				if allowedRes == nil {
					resExample = exampleJSON(j, resRef)
					resTable = tools.StripComponentTypeDecorations(renderResponseParamFlatTableHTMLFromRef(j, resRef))
				} else {
					rs := getRefJson(j, resRef)
					if rs != nil {
						resExample = exampleJSONWithAllowed(j, resRef, allowedRes)
						resTable = tools.StripComponentTypeDecorations(renderResponseParamFlatTableHTMLFromJsonWithAllowed(j, rs, allowedRes))
					}
				}
			}
			var headersHTML, pathParamsHTML, queryParamsHTML string
			if len(headers) > 0 {
				headersHTML = renderParamInfoTableHTML(headers)
			}
			if len(pathsParams) > 0 {
				pathParamsHTML = renderParamInfoTableHTML(pathsParams)
			}
			if len(queryParams) > 0 {
				queryParamsHTML = renderParamInfoTableHTML(queryParams)
			}
			if terr == nil {
				// 将当前端点数据注入模板片段
				_ = t.ExecuteTemplate(&bm, "endpoint", EndpointData{
					Anchor:          anchor,
					MethodUpper:     strings.ToUpper(m),
					Summary:         tools.HTMLEscape(summary),
					Path:            tools.HTMLEscape(p),
					ContentType:     tools.HTMLEscape(ct),
					HeadersHTML:     template.HTML(headersHTML),
					PathParamsHTML:  template.HTML(pathParamsHTML),
					QueryParamsHTML: template.HTML(queryParamsHTML),
					ReqExample:      template.HTML(tools.HTMLEscape(reqExample)),
					ReqTableHTML:    template.HTML(reqTable),
					ResExample:      template.HTML(htmlEscape(resExample)),
					ResTableHTML:    template.HTML(resTable),
				})
			} else {
				// 模板不可用时，降级输出空端点占位块
				bm.WriteString("<div class=\"endpoint\" id=\"" + anchor + "\"></div>")
			}
		}
	}
	if terr != nil {
		// 模板加载失败时，将导航与正文拼接为最简布局
		var fallback strings.Builder
		fallback.WriteString("<div class=\"layout\"><aside>")
		fallback.WriteString(bn.String())
		fallback.WriteString("</aside><main>")
		fallback.WriteString(bm.String())
		fallback.WriteString("</main></div>")
		return fallback.String()
	}
	var out strings.Builder
	_ = t.ExecuteTemplate(&out, "layout", pageData{Title: title, NavHTML: template.HTML(bn.String()), MainHTML: template.HTML(bm.String())})
	return out.String()
}

// GenerateMarkdownWithConfig 与 GenerateMarkdown 一致，但支持 RenderConfig.Customize 的 Header 注入与 Req/Res 过滤。
func GenerateMarkdownWithConfig(j *gjson.Json, contentRaw string, cfg RenderConfig) string {
	title := strings.TrimSpace(j.Get("info.title").String())
	if title == "" {
		title = "API 文档"
	}
	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	paths := j.GetJsonMap("paths")
	keys := source.OrderedPathsFromContent(contentRaw)
	if len(keys) == 0 {
		for k := range paths {
			keys = append(keys, k)
		}
		sortStrings(keys)
	}
	groups := make(map[string]map[string][]struct{ P, M string })
	pfxOrder := make([]string, 0, len(keys))
	sfxOrder := make(map[string][]string)
	for _, p := range keys {
		pj := paths[p]
		methods := presentMethods(pj)
		for _, m := range methods {
			mj := pj.GetJson(m)
			if mj == nil {
				continue
			}
			tags := tools.JSONArrayStrings(mj.Get("tags").Array())
			pre, suf := tools.SplitTagParts(tags)
			if _, ok := groups[pre]; !ok {
				groups[pre] = make(map[string][]struct{ P, M string })
				pfxOrder = append(pfxOrder, pre)
			}
			if _, ok := groups[pre][suf]; !ok {
				groups[pre][suf] = make([]struct{ P, M string }, 0, 8)
				sfxOrder[pre] = append(sfxOrder[pre], suf)
			}
			groups[pre][suf] = append(groups[pre][suf], struct{ P, M string }{P: p, M: m})
		}
	}
	for _, pre := range pfxOrder {
		b.WriteString("## " + pre + "\n\n")
		sfx := sfxOrder[pre]
		for _, suf := range sfx {
			b.WriteString("### " + suf + "\n\n")
			items := groups[pre][suf]
			for _, it := range items {
				p := it.P
				m := it.M
				pj := paths[p]
				mj := pj.GetJson(m)
				summary := strings.TrimSpace(mj.Get("summary").String())
				if summary == "" {
					summary = strings.ToUpper(m) + " " + p
				}
				b.WriteString("#### " + summary + "\n\n")
				b.WriteString("##### 请求URL\n\n`" + p + "`\n\n")
				reqSchema, reqCT, reqRef := getRequestSchema(j, mj)
				resSchema, _, resRef := getResponseSchema(j, mj)
				ct := reqCT
				if ct == "" {
					ct = "application/json"
				}
				b.WriteString("##### 请求方式\n\n- " + strings.ToUpper(m) + "  Content-Type: " + ct + "\n\n")
				headers, pathsParams, queryParams := collectParameters(j, pj, mj)
				headers = applyCustomizeHeaders(p, headers, cfg)
				if len(headers) > 0 {
					b.WriteString("##### Header参数\n\n" + renderParamInfoTableMarkdown(headers) + "\n")
				}
				if len(pathsParams) > 0 {
					b.WriteString("##### 路径参数\n\n" + renderParamInfoTableMarkdown(pathsParams) + "\n")
				}
				if len(queryParams) > 0 {
					b.WriteString("##### Query参数\n\n" + renderParamInfoTableMarkdown(queryParams) + "\n")
				}
				allowedReq := cfg.Customize[p].Request
				if reqSchema != nil {
					if allowedReq == nil {
						b.WriteString("##### 请求示例\n\n```json\n" + exampleJSONFromSchema(j, reqSchema) + "\n```\n\n")
						b.WriteString("##### 请求参数\n\n" + renderParamTableMarkdownFromJson(j, reqSchema) + "\n")
					} else {
						b.WriteString("##### 请求示例\n\n```json\n" + exampleJSONFromSchemaWithAllowed(j, reqSchema, allowedReq) + "\n```\n\n")
						b.WriteString("##### 请求参数\n\n" + renderParamTableMarkdownFromJsonWithAllowed(j, reqSchema, allowedReq) + "\n")
					}
				} else if reqRef != "" {
					if allowedReq == nil {
						b.WriteString("##### 请求示例\n\n```json\n" + exampleJSON(j, reqRef) + "\n```\n\n")
						b.WriteString("##### 请求参数\n\n" + renderParamTableMarkdown(j, reqRef) + "\n")
					} else {
						b.WriteString("##### 请求示例\n\n```json\n" + exampleJSONWithAllowed(j, reqRef, allowedReq) + "\n```\n\n")
						b.WriteString("##### 请求参数\n\n" + renderParamTableMarkdownWithAllowed(j, reqRef, allowedReq) + "\n")
					}
				}
				allowedRes := cfg.Customize[p].Response
				if resSchema != nil {
					if allowedRes == nil {
						b.WriteString("##### 返回示例\n\n```json\n" + exampleJSONFromSchema(j, resSchema) + "\n```\n\n")
						b.WriteString("##### 返回参数说明\n\n" + tools.StripComponentTypeDecorations(renderResponseParamFlatTableMarkdownFromJson(j, resSchema)) + "\n")
					} else {
						b.WriteString("##### 返回示例\n\n```json\n" + exampleJSONFromSchemaWithAllowed(j, resSchema, allowedRes) + "\n```\n\n")
						b.WriteString("##### 返回参数说明\n\n" + tools.StripComponentTypeDecorations(renderResponseParamFlatTableMarkdownFromJsonWithAllowed(j, resSchema, allowedRes)) + "\n")
					}
				} else if resRef != "" {
					if allowedRes == nil {
						b.WriteString("##### 返回示例\n\n```json\n" + exampleJSON(j, resRef) + "\n```\n\n")
						b.WriteString("##### 返回参数说明\n\n" + tools.StripComponentTypeDecorations(renderResponseParamFlatTableMarkdownFromRef(j, resRef)) + "\n")
					} else {
						rs := getRefJson(j, resRef)
						if rs != nil {
							b.WriteString("##### 返回示例\n\n```json\n" + exampleJSONWithAllowed(j, resRef, allowedRes) + "\n```\n\n")
							b.WriteString("##### 返回参数说明\n\n" + tools.StripComponentTypeDecorations(renderResponseParamFlatTableMarkdownFromJsonWithAllowed(j, rs, allowedRes)) + "\n")
						}
					}
				}
			}
		}
	}
	return b.String()
}

// GenerateMarkdown 生成与 HTML 顺序一致的 Markdown 文档（用于下载）。
func GenerateMarkdown(j *gjson.Json, contentRaw string) string {
	return GenerateMarkdownWithConfig(j, contentRaw, RenderConfig{})
}

// presentMethods 返回 path 项中有效的 HTTP 方法列表（小写，排序）。
func presentMethods(pj *gjson.Json) []string {
	allowed := map[string]struct{}{"get": {}, "post": {}, "put": {}, "delete": {}, "patch": {}, "options": {}, "head": {}}
	mp := pj.Map()
	res := make([]string, 0, len(mp))
	for k := range mp {
		if _, ok := allowed[strings.ToLower(k)]; ok {
			res = append(res, strings.ToLower(k))
		}
	}
	sortStrings(res)
	return res
}

// getRequestSchema 提取请求体的 schema（优先 $ref），返回 (schema, contentType, schemaRef)。
// 说明：遍历 requestBody.content 的媒体类型，记录 contentType；
// - 若 schema.$ref 存在，直接解析引用；
// - 若为内联 schema，返回该对象。
func getRequestSchema(j *gjson.Json, op *gjson.Json) (schema *gjson.Json, contentType, schemaRef string) {
	mp := op.GetJsonMap("requestBody.content")
	for ct, v := range mp {
		contentType = ct
		schemaRef = v.Get("schema.$ref").String()
		if schemaRef != "" {
			return getRefJson(j, schemaRef), contentType, schemaRef
		}
		s := v.GetJson("schema")
		if s != nil {
			return s, contentType, ""
		}
	}
	return nil, "", ""
}

// getResponseSchema 提取响应体的 schema（优先选 200 状态码）。
// 说明：
// - 若不存在 200，则选择第一个状态码；
// - 若 schema.$ref 存在，解析引用；否则返回内联 schema；
// - 返回 (schema, contentType, schemaRef)。
func getResponseSchema(j *gjson.Json, op *gjson.Json) (schema *gjson.Json, contentType, schemaRef string) {
	resps := op.GetJsonMap("responses")
	codes := make([]string, 0, len(resps))
	for c := range resps {
		codes = append(codes, c)
	}
	sortStrings(codes)
	pick := ""
	for _, c := range codes {
		if c == "200" {
			pick = c
			break
		}
	}
	if pick == "" && len(codes) > 0 {
		pick = codes[0]
	}
	if pick == "" {
		return nil, "", ""
	}
	mp := resps[pick].GetJsonMap("content")
	keys := make([]string, 0, len(mp))
	for k := range mp {
		keys = append(keys, k)
	}
	prefer := []string{"application/json", "application/problem+json", "application/ld+json"}
	for _, p := range prefer {
		if v, ok := mp[p]; ok {
			contentType = p
			schemaRef = v.Get("schema.$ref").String()
			if schemaRef != "" {
				return getRefJson(j, schemaRef), contentType, schemaRef
			}
			s := v.GetJson("schema")
			if s != nil {
				return s, contentType, ""
			}
		}
	}
	for _, k := range keys {
		if strings.Contains(k, "json") {
			v := mp[k]
			contentType = k
			schemaRef = v.Get("schema.$ref").String()
			if schemaRef != "" {
				return getRefJson(j, schemaRef), contentType, schemaRef
			}
			s := v.GetJson("schema")
			if s != nil {
				return s, contentType, ""
			}
		}
	}
	sortStrings(keys)
	for _, k := range keys {
		v := mp[k]
		contentType = k
		schemaRef = v.Get("schema.$ref").String()
		if schemaRef != "" {
			return getRefJson(j, schemaRef), contentType, schemaRef
		}
		s := v.GetJson("schema")
		if s != nil {
			return s, contentType, ""
		}
	}
	return nil, "", ""
}

// collectParameters 汇总 path+op 层的参数（header/path/query），并解析 $ref。
func collectParameters(j *gjson.Json, pathItem *gjson.Json, op *gjson.Json) (headers, pathsParams, queryParams []paramInfo) {
	arr := append(pathItem.Get("parameters").Array(), op.Get("parameters").Array()...)
	for _, v := range arr {
		pj := gjson.New(v)
		if ref := pj.Get("$ref").String(); ref != "" {
			rp := getRefJson(j, ref)
			if rp == nil {
				continue
			}
			pj = rp
		}
		name := pj.Get("name").String()
		in := pj.Get("in").String()
		req := pj.Get("required").Bool()
		required := "否"
		if req {
			required = "是"
		}
		desc := pj.Get("description").String()
		typ := paramSchemaType(j, pj.GetJson("schema"))
		info := paramInfo{Name: name, Required: required, Type: typ, Desc: desc}
		switch in {
		case "header":
			headers = append(headers, info)
		case "path":
			pathsParams = append(pathsParams, info)
		case "query":
			queryParams = append(queryParams, info)
		}
	}
	return
}

// paramSchemaType 解析参数 schema 的类型字符串。
// - 兼容 $ref（显示为 object(<Component>)）
// - 数组 items 兼容 $ref 与基础类型；对空 items 返回 "array"
func paramSchemaType(j *gjson.Json, s *gjson.Json) string {
	if s == nil {
		return ""
	}
	if r := s.Get("$ref").String(); r != "" {
		return "object"
	}
	t := s.Get("type").String()
	if t == "array" {
		it := s.GetJson("items")
		if it != nil {
			if r := it.Get("$ref").String(); r != "" {
				return "array(object)"
			}
			itt := it.Get("type").String()
			if it.Get("format").String() != "" {
				return "array(" + it.Get("format").String() + ")"
			}
			if itt != "" {
				return "array(" + itt + ")"
			}
			return "array"
		}
		return "array"
	}
	f := s.Get("format").String()
	if f != "" {
		return t
	}
	return t
}
