package apidocs

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"

    "github.com/gogf/gf/v2/encoding/gjson"
    "github.com/gogf/gf/v2/net/ghttp"
    "github.com/gogf/gf/v2/os/gfile"
)

// Server 持有默认的 OpenAPI 解析对象与其原始文本，用于路由处理时回退。
// - spec: 解析后的 OpenAPI 结构
// - raw: 原始 JSON 文本（保持 paths 原始顺序）
type Server struct {
    spec *gjson.Json
    raw  string
}

// Register 在服务器上注册默认的文档与导出路由（/docs 与 /docs.md）。
// - s: GoFrame 服务器
// - defaultContent: 默认 OpenAPI 文本（当未提供 src 参数时使用）
// 返回：Server 结构体
func Register(s *ghttp.Server, defaultContent string) *Server {
    var j *gjson.Json
    if defaultContent != "" {
        jj, err := gjson.LoadContent(defaultContent)
        if err == nil { j = jj }
    }
    srv := &Server{spec: j, raw: defaultContent}
    s.BindHandler("GET:/docs", func(r *ghttp.Request) {
        spec := srv.spec
        contentRaw := srv.raw
        if src := r.Get("src").String(); src != "" {
            if j2, raw, err := loadSpecFromSource(src); err == nil && j2 != nil { spec = j2; contentRaw = raw }
        }
        r.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
        r.Response.Write(generateHTML(spec, contentRaw))
    })
    s.BindHandler("GET:/docs.md", func(r *ghttp.Request) {
        spec := srv.spec
        contentRaw := srv.raw
        if src := r.Get("src").String(); src != "" {
            if j2, raw, err := loadSpecFromSource(src); err == nil && j2 != nil { spec = j2; contentRaw = raw }
        }
        md := generateMarkdown(spec, contentRaw)
        r.Response.Header().Set("Content-Type", "text/markdown; charset=utf-8")
        r.Response.Header().Set("Content-Disposition", "attachment; filename=api-docs.md")
        r.Response.Write(md)
    })
    return srv
}

func generateHTML(j *gjson.Json, contentRaw string) string {
    title := strings.TrimSpace(j.Get("info.title").String())
    if title == "" { title = "API 文档" }
    var b strings.Builder
    b.WriteString("<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\">")
    b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
    b.WriteString("<title>" + htmlEscape(title) + "</title>")
    b.WriteString("<style>*{box-sizing:border-box} body{font-family:Arial, sans-serif;line-height:1.6;margin:0;padding-left:280px}")
    b.WriteString(".layout{min-height:100vh}")
    b.WriteString("aside{position:fixed;left:0;top:0;bottom:0;height:100vh;width:280px;background:#f5f7fa;border-right:1px solid #e5e9f2;padding:16px;overflow:auto;z-index:10}")
    b.WriteString("main{background:#fff;padding:24px;position:relative;z-index:1}")
    b.WriteString("h1{margin:0 0 12px 0;font-size:20px}")
    b.WriteString("h2{margin-top:24px;font-size:18px}")
    b.WriteString("h3{margin-top:16px;font-size:16px}")
    b.WriteString("code,pre{background:#f8f9fb;border:1px solid #e5e9f2;border-radius:6px}")
    b.WriteString("pre{padding:12px;overflow:auto}")
    b.WriteString("table{border-collapse:collapse;width:100%;margin:12px 0}")
    b.WriteString("th,td{border:1px solid #dfe3e8;padding:8px;text-align:left}")
    b.WriteString("th{background:#fafbfc}")
    b.WriteString(".endpoint{margin-bottom:32px}")
    b.WriteString(".method{display:inline-block;background:#eef2ff;color:#3f51b5;border:1px solid #c7d2fe;border-radius:12px;padding:2px 8px;margin-right:6px;font-size:12px}")
    b.WriteString(".export-fixed{position:fixed;top:10px;right:12px;background:#3f51b5;color:#fff;border:none;border-radius:20px;padding:8px 14px;box-shadow:0 2px 6px rgba(0,0,0,.15);text-decoration:none;z-index:999}")
    b.WriteString(".nav details{margin:4px 0}")
    b.WriteString(".nav summary{cursor:pointer;padding:4px 2px;color:#333;font-size:13px;display:flex;align-items:center;gap:6px}")
    b.WriteString(".nav summary a{color:#2c2c2c;text-decoration:none}")
    b.WriteString(".nav summary a:hover{color:#000}")
    b.WriteString(".nav details > summary::before{content:'\\25B6';display:inline-block;width:8px;color:#888;margin-right:4px;transition:transform .2s;transform:rotate(0deg);font-size:9px}")
    b.WriteString(".nav details[open] > summary::before{content:'\\25B6';color:#555;transform:rotate(90deg);font-size:9px}")
    b.WriteString(".nav details.grp > summary{font-weight:600;font-size:15px}")
    b.WriteString(".nav details.subgrp > summary{padding-left:14px;color:#555;font-size:14px}")
    b.WriteString(".nav .item{padding-left:28px}")
    b.WriteString(".nav .item-link{display:block;color:#2c2c2c;text-decoration:none;padding:3px 2px;font-size:12px}")
    b.WriteString(".nav .item-link::before{content:'•';display:inline-block;margin-right:6px;color:#9aa0a6}")
    b.WriteString(".nav .item-link.active{color:#16C06E;font-weight:600}")
    b.WriteString(".nav .item-link.active::before{color:#16C06E}")
    b.WriteString("</style></head><body>")
    b.WriteString("<div class=\"layout\"><aside><div class=\"tabs\"><div class=\"tab active\">菜单</div></div>")
    // 构建侧边导航（保持 paths 的原始顺序，避免排序导致菜单与内容不一致）
    paths := j.GetJsonMap("paths")
    keys := orderedPathsFromContent(contentRaw)
    groups := make(map[string]map[string][][2]string)
    pfxOrder := make([]string, 0, len(keys))
    sfxOrder := make(map[string][]string)
    for _, p := range keys {
        pj := paths[p]
        methods := presentMethods(pj)
        for _, m := range methods {
            mj := pj.GetJson(m)
            if mj == nil { continue }
            tags := jsonArrayStrings(mj.Get("tags").Array())
            pre, suf := splitTagParts(tags)
            summary := strings.TrimSpace(mj.Get("summary").String())
            if summary == "" { summary = strings.ToUpper(m) + " " + p }
            anchor := anchorID(m, p)
            if _, ok := groups[pre]; !ok { groups[pre] = make(map[string][][2]string); pfxOrder = append(pfxOrder, pre) }
            if _, ok := groups[pre][suf]; !ok { groups[pre][suf] = make([][2]string, 0, 8); sfxOrder[pre] = append(sfxOrder[pre], suf) }
            groups[pre][suf] = append(groups[pre][suf], [2]string{summary, anchor})
        }
    }
    b.WriteString("<div class=\"nav\"><div style=\"display:flex;gap:8px;align-items:center;margin-bottom:8px\"><button id=\"expandAll\" style=\"padding:4px 8px;border:1px solid #c7d2fe;background:#eef2ff;border-radius:6px;\">全部展开</button><button id=\"collapseAll\" style=\"padding:4px 8px;border:1px solid #e5e9f2;background:#f8f9fb;border-radius:6px;\">全部收起</button></div>")
    for _, pre := range pfxOrder {
        b.WriteString("<details class=\"grp\" open><summary><a href=\"#group-" + slugify(pre) + "\">" + htmlEscape(pre) + "</a></summary>")
        sfx := sfxOrder[pre]
        for _, suf := range sfx {
            b.WriteString("<details class=\"subgrp\" open><summary><a href=\"#group-" + slugify(pre) + "-" + slugify(suf) + "\">" + htmlEscape(suf) + "</a></summary>")
            items := groups[pre][suf]
            for _, it := range items {
                b.WriteString("<div class=\"item\"><a class=\"item-link\" href=\"#" + it[1] + "\">" + htmlEscape(it[0]) + "</a></div>")
            }
            b.WriteString("</details>")
        }
        b.WriteString("</details>")
    }
    b.WriteString("</div></aside>")
    b.WriteString("<main><div style=\"display:flex;align-items:center;justify-content:space-between\"><h1>" + htmlEscape(title) + "</h1></div>")
    b.WriteString("<a class=\"export-fixed\" href=\"/docs.md\" title=\"导出为 Markdown\">导出为 Markdown</a>")
    var currPre, currSuf string
    for _, p := range keys {
        pj := paths[p]
        methods := presentMethods(pj)
        for _, m := range methods {
            mj := pj.GetJson(m)
            if mj == nil { continue }
            summary := strings.TrimSpace(mj.Get("summary").String())
            tags := jsonArrayStrings(mj.Get("tags").Array())
            _, reqCT, reqRef := getRequestSchema(j, mj)
            _, _, resRef := getResponseSchema(j, mj)
            pre, suf := splitTagParts(tags)
            if pre != currPre { currPre = pre; currSuf = ""; b.WriteString("<h1 id=\"group-" + slugify(pre) + "\">" + htmlEscape(pre) + "</h1>") }
            if suf != currSuf { currSuf = suf; b.WriteString("<h2 id=\"group-" + slugify(pre) + "-" + slugify(suf) + "\">" + htmlEscape(suf) + "</h2>") }
            anchor := anchorID(m, p)
            b.WriteString("<div class=\"endpoint\" id=\"" + anchor + "\"><h2><span class=\"method\">" + strings.ToUpper(m) + "</span> " + htmlEscape(summary) + "</h2>")
            b.WriteString("<h3 id=\"" + anchor + "-url\">请求URL</h3><pre><code>" + htmlEscape(p) + "</code></pre>")
            ct := reqCT
            if ct == "" { ct = "application/json" }
            b.WriteString("<h3 id=\"" + anchor + "-method\">请求方式</h3><ul><li>" + strings.ToUpper(m) + " <em>Content-Type: " + htmlEscape(ct) + "</em></li></ul>")
            ph := pj
            headers, pathsParams, queryParams := collectParameters(j, ph, mj)
            if len(headers) > 0 { b.WriteString("<h3 id=\"" + anchor + "-headers\">Header参数</h3>" + renderParamInfoTableHTML(headers)) }
            if len(pathsParams) > 0 { b.WriteString("<h3 id=\"" + anchor + "-path-params\">路径参数</h3>" + renderParamInfoTableHTML(pathsParams)) }
            if len(queryParams) > 0 { b.WriteString("<h3 id=\"" + anchor + "-query-params\">Query参数</h3>" + renderParamInfoTableHTML(queryParams)) }
            reqSchema, _, reqRef2 := getRequestSchema(j, mj)
            if reqRef == "" { reqRef = reqRef2 }
            if reqSchema != nil {
                exReq := exampleJSONFromSchema(j, reqSchema)
                b.WriteString("<h3 id=\"" + anchor + "-req-example\">请求示例</h3><pre><code>" + htmlEscape(exReq) + "</code></pre><h3 id=\"" + anchor + "-req\">请求参数</h3>" + renderParamTableHTMLFromJson(j, reqSchema))
            } else if reqRef != "" {
                exReq := exampleJSON(j, reqRef)
                b.WriteString("<h3 id=\"" + anchor + "-req-example\">请求示例</h3><pre><code>" + htmlEscape(exReq) + "</code></pre><h3 id=\"" + anchor + "-req\">请求参数</h3>" + renderParamTableHTML(j, reqRef))
            }
            resSchema, _, resRef2 := getResponseSchema(j, mj)
            if resRef == "" { resRef = resRef2 }
            if resSchema != nil {
                ex := exampleJSONFromSchema(j, resSchema)
                b.WriteString("<h3 id=\"" + anchor + "-res-example\">返回示例</h3><pre><code>" + htmlEscape(ex) + "</code></pre><h3 id=\"" + anchor + "-res-params\">返回参数说明</h3>" + renderResponseParamFlatTableHTMLFromJson(j, resSchema))
            } else if resRef != "" {
                ex := exampleJSON(j, resRef)
                b.WriteString("<h3 id=\"" + anchor + "-res-example\">返回示例</h3><pre><code>" + htmlEscape(ex) + "</code></pre><h3 id=\"" + anchor + "-res-params\">返回参数说明</h3>" + renderResponseParamFlatTableHTMLFromRef(j, resRef))
            }
            b.WriteString("</div>")
        }
    }
    b.WriteString("</main><script>function setActive(){var h=location.hash;document.querySelectorAll('aside .nav .item-link').forEach(function(a){a.classList.toggle('active',a.getAttribute('href')===h);});}window.addEventListener('hashchange',setActive);setActive();document.getElementById('expandAll').onclick=function(){document.querySelectorAll('aside .nav details').forEach(function(d){d.open=true;});};document.getElementById('collapseAll').onclick=function(){document.querySelectorAll('aside .nav details').forEach(function(d){d.open=false;});};</script></div></body></html>")
    return b.String()
}

// generateMarkdown 生成与 HTML 顺序一致的 Markdown 文档（用于下载）。
// - j: 解析后的 OpenAPI
// - contentRaw: 原始 JSON 文本，用于还原 paths 顺序
func generateMarkdown(j *gjson.Json, contentRaw string) string {
    title := strings.TrimSpace(j.Get("info.title").String())
    if title == "" { title = "API 文档" }
    var b strings.Builder
    b.WriteString("# " + title + "\n\n")
    paths := j.GetJsonMap("paths")
    keys := orderedPathsFromContent(contentRaw)
    groups := make(map[string]map[string][]struct{ P, M string })
    pfxOrder := make([]string, 0, len(keys))
    sfxOrder := make(map[string][]string)
    for _, p := range keys {
        pj := paths[p]
        methods := presentMethods(pj)
        for _, m := range methods {
            mj := pj.GetJson(m)
            if mj == nil { continue }
            tags := jsonArrayStrings(mj.Get("tags").Array())
            pre, suf := splitTagParts(tags)
            if _, ok := groups[pre]; !ok { groups[pre] = make(map[string][]struct{ P, M string }); pfxOrder = append(pfxOrder, pre) }
            if _, ok := groups[pre][suf]; !ok { groups[pre][suf] = make([]struct{ P, M string }, 0, 8); sfxOrder[pre] = append(sfxOrder[pre], suf) }
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
                p := it.P; m := it.M
                pj := paths[p]; mj := pj.GetJson(m)
                summary := strings.TrimSpace(mj.Get("summary").String())
                if summary == "" { summary = strings.ToUpper(m) + " " + p }
                b.WriteString("#### " + summary + "\n\n")
                b.WriteString("##### 请求URL\n\n`" + p + "`\n\n")
                reqSchema, reqCT, reqRef := getRequestSchema(j, mj)
                resSchema, _, resRef := getResponseSchema(j, mj)
                ct := reqCT; if ct == "" { ct = "application/json" }
                b.WriteString("##### 请求方式\n\n- " + strings.ToUpper(m) + "  Content-Type: " + ct + "\n\n")
                headers, pathsParams, queryParams := collectParameters(j, pj, mj)
                if len(headers) > 0 { b.WriteString("##### Header参数\n\n" + renderParamInfoTableMarkdown(headers) + "\n") }
                if len(pathsParams) > 0 { b.WriteString("##### 路径参数\n\n" + renderParamInfoTableMarkdown(pathsParams) + "\n") }
                if len(queryParams) > 0 { b.WriteString("##### Query参数\n\n" + renderParamInfoTableMarkdown(queryParams) + "\n") }
                if reqSchema != nil {
                    b.WriteString("##### 请求示例\n\n```json\n" + exampleJSONFromSchema(j, reqSchema) + "\n```\n\n")
                    b.WriteString("##### 请求参数\n\n" + renderParamTableMarkdownFromJson(j, reqSchema) + "\n")
                } else if reqRef != "" {
                    b.WriteString("##### 请求示例\n\n```json\n" + exampleJSON(j, reqRef) + "\n```\n\n")
                    b.WriteString("##### 请求参数\n\n" + renderParamTableMarkdown(j, reqRef) + "\n")
                }
                if resSchema != nil {
                    b.WriteString("##### 返回示例\n\n```json\n" + exampleJSONFromSchema(j, resSchema) + "\n```\n\n")
                    b.WriteString("##### 返回参数说明\n\n" + renderResponseParamFlatTableMarkdownFromJson(j, resSchema) + "\n")
                } else if resRef != "" {
                    b.WriteString("##### 返回示例\n\n```json\n" + exampleJSON(j, resRef) + "\n```\n\n")
                    b.WriteString("##### 返回参数说明\n\n" + renderResponseParamFlatTableMarkdownFromRef(j, resRef) + "\n")
                }
            }
        }
    }
    return b.String()
}

func fetchURL(u string) (string, error) {
    resp, err := http.Get(u)
    if err != nil { return "", err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK { return "", fmt.Errorf("http status %s", resp.Status) }
    b, err := io.ReadAll(resp.Body)
    if err != nil { return "", err }
    return string(b), nil
}

// loadSpecFromSource 按 src 加载 OpenAPI 文档，支持本地、HTTP、file://，并归一化 Windows 路径与片段。
func loadSpecFromSource(src string) (*gjson.Json, string, error) {
    // 归一化 Windows 路径，并移除片段（例如：/d:/...#L9-92）
    s := strings.TrimSpace(src)
    if strings.HasPrefix(s, "#/") { s = s[1:] } else if strings.HasPrefix(s, "#") { s = s[1:] }
    if i := strings.Index(s, "#"); i >= 0 { s = s[:i] }
    if strings.HasPrefix(s, "/") && len(s) >= 3 && s[2] == ':' { s = s[1:] }
    if strings.HasPrefix(strings.ToLower(s), "file://") {
        s = strings.TrimPrefix(s, "file://")
        if strings.HasPrefix(s, "/") && len(s) >= 3 && s[2] == ':' { s = s[1:] }
    }
    var content string
    var err error
    if strings.HasPrefix(strings.ToLower(s), "http") {
        content, err = fetchURL(s); if err != nil { return nil, "", err }
    } else {
        content = gfile.GetContents(s)
        if content == "" { return nil, "", fmt.Errorf("empty content from %s", s) }
    }
    j, e := gjson.LoadContent(content)
    if e != nil { return nil, "", e }
    return j, content, nil
}

// orderedPathsFromContent 使用流式解码提取 paths 键的原始顺序。
func orderedPathsFromContent(content string) []string {
    dec := json.NewDecoder(strings.NewReader(content))
    for {
        tok, err := dec.Token(); if err != nil { return []string{} }
        if key, ok := tok.(string); ok && key == "paths" { break }
    }
    if _, err := dec.Token(); err != nil { return []string{} }
    keys := make([]string, 0, 64)
    for dec.More() {
        t, err := dec.Token(); if err != nil { break }
        k, _ := t.(string)
        keys = append(keys, k)
        var raw json.RawMessage
        if err := dec.Decode(&raw); err != nil { break }
    }
    _, _ = dec.Token()
    return keys
}

// collectParameters 汇总 path+op 层的参数（header/path/query），并解析 $ref。
func collectParameters(j *gjson.Json, pathItem *gjson.Json, op *gjson.Json) (headers, pathsParams, queryParams []paramInfo) {
    arr := append(pathItem.Get("parameters").Array(), op.Get("parameters").Array()...)
    for _, v := range arr {
        pj := gjson.New(v)
        if ref := pj.Get("$ref").String(); ref != "" { rp := getRefJson(j, ref); if rp == nil { continue }; pj = rp }
        name := pj.Get("name").String()
        in := pj.Get("in").String()
        req := pj.Get("required").Bool()
        required := "否"; if req { required = "是" }
        desc := pj.Get("description").String()
        typ := paramSchemaType(j, pj.GetJson("schema"))
        info := paramInfo{Name: name, Required: required, Type: typ, Desc: desc}
        switch in { case "header": headers = append(headers, info); case "path": pathsParams = append(pathsParams, info); case "query": queryParams = append(queryParams, info) }
    }
    return
}

// paramInfo 参数的展示结构
type paramInfo struct { Name, Required, Type, Desc string }

// paramSchemaType 解析参数 schema 的类型（含数组 items 与 $ref）。
func paramSchemaType(j *gjson.Json, s *gjson.Json) string {
    if s == nil { return "" }
    if r := s.Get("$ref").String(); r != "" { return "object(" + componentNameFromRef(r) + ")" }
    t := s.Get("type").String()
    if t == "array" {
        it := s.GetJson("items"); if it == nil { return "array" }
        if r := it.Get("$ref").String(); r != "" { return "array(object(" + componentNameFromRef(r) + "))" }
        itt := it.Get("type").String()
        if it.Get("format").String() != "" { return "array(" + it.Get("format").String() + ")" }
        if itt != "" { return "array(" + itt + ")" }
        return "array"
    }
    f := s.Get("format").String(); if f != "" { return t }
    return t
}

func renderParamInfoTableHTML(list []paramInfo) string {
    var b strings.Builder
    b.WriteString("<table><thead><tr><th>参数名</th><th>必选</th><th>类型</th><th>说明</th></tr></thead><tbody>")
    for _, it := range list { b.WriteString("<tr><td>" + htmlEscape(it.Name) + "</td><td>" + it.Required + "</td><td>" + htmlEscape(it.Type) + "</td><td>" + htmlEscape(it.Desc) + "</td></tr>") }
    b.WriteString("</tbody></table>")
    return b.String()
}
func renderParamInfoTableMarkdown(list []paramInfo) string {
    var b strings.Builder
    b.WriteString("| 参数名 | 必选 | 类型 | 说明 |\n|---|---|---|---|\n")
    for _, it := range list { b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", it.Name, it.Required, it.Type, it.Desc)) }
    return b.String()
}

func renderParamTableHTML(j *gjson.Json, ref string) string {
    fields := flattenSchemaFields(j, ref, "")
    var b strings.Builder
    b.WriteString("<table><thead><tr><th>参数名</th><th>必选</th><th>类型</th><th>说明</th></tr></thead><tbody>")
    for _, f := range fields { req := "否"; if f.Required { req = "是" }; b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + req + "</td><td>" + htmlEscape(f.Type) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>") }
    b.WriteString("</tbody></table>")
    return b.String()
}
func renderParamTableMarkdown(j *gjson.Json, ref string) string {
    fields := flattenSchemaFields(j, ref, "")
    var b strings.Builder
    b.WriteString("| 参数名 | 必选 | 类型 | 说明 |\n|---|---|---|---|\n")
    for _, f := range fields { req := "否"; if f.Required { req = "是" }; b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", f.Path, req, f.Type, f.Desc)) }
    return b.String()
}

func renderParamTableHTMLFromJson(j *gjson.Json, sj *gjson.Json) string {
    fields := flattenSchemaFieldsFromJson(j, sj, "")
    var b strings.Builder
    b.WriteString("<table><thead><tr><th>参数名</th><th>必选</th><th>类型</th><th>说明</th></tr></thead><tbody>")
    for _, f := range fields { req := "否"; if f.Required { req = "是" }; b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + req + "</td><td>" + htmlEscape(f.Type) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>") }
    b.WriteString("</tbody></table>")
    return b.String()
}
func renderParamTableMarkdownFromJson(j *gjson.Json, sj *gjson.Json) string {
    fields := flattenSchemaFieldsFromJson(j, sj, "")
    var b strings.Builder
    b.WriteString("| 参数名 | 必选 | 类型 | 说明 |\n|---|---|---|---|\n")
    for _, f := range fields { req := "否"; if f.Required { req = "是" }; b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", f.Path, req, f.Type, f.Desc)) }
    return b.String()
}

func renderResponseParamFlatTableHTMLFromJson(j *gjson.Json, sj *gjson.Json) string {
    target := sj
    if !sj.Get("properties.data").IsNil() {
        d := sj.GetJson("properties.data")
        if r := d.Get("$ref").String(); r != "" { target = getSchema(j, componentNameFromRef(r)) } else { target = d }
    }
    prefix := ""; if target != sj { prefix = "data" }
    var fields []FieldInfo
    if target.Get("type").String() == "array" {
        it := target.GetJson("items")
        if it != nil {
            if r := it.Get("$ref").String(); r != "" {
                sub := getRefJson(j, r)
                if sub != nil {
                    for k, p := range sub.GetJsonMap("properties") {
                        t := p.Get("type").String()
                        if rr := p.Get("$ref").String(); rr != "" { t = "object(" + componentNameFromRef(rr) + ")" }
                        if t == "array" { it2 := p.GetJson("items"); if it2 != nil { if r2 := it2.Get("$ref").String(); r2 != "" { t = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { t = "array(" + it2.Get("type").String() + ")" } else { t = "array" } } else { t = "array" } }
                        fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
                    }
                }
            } else if it.Get("type").String() == "object" {
                props := it.GetJsonMap("properties")
                keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
                for _, k := range keys {
                    p := props[k]
                    t := p.Get("type").String()
                    if rr := p.Get("$ref").String(); rr != "" { t = "object(" + componentNameFromRef(rr) + ")" }
                    if t == "array" { it2 := p.GetJson("items"); if it2 != nil { if r2 := it2.Get("$ref").String(); r2 != "" { t = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { t = "array(" + it2.Get("type").String() + ")" } else { t = "array" } } else { t = "array" } } else if t == "" { t = "object" }
                    fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
                }
            } else {
                tt := it.Get("type").String(); if tt != "" { fields = append(fields, FieldInfo{Path: prefix + "[]", Type: "array(" + tt + ")", Desc: it.Get("description").String()}) }
            }
        }
    } else {
        props := target.GetJsonMap("properties")
        keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
        for _, k := range keys {
            p := props[k]
            cur := k; if prefix != "" { cur = prefix + "." + k }
            t := p.Get("type").String()
            if r := p.Get("$ref").String(); r != "" {
                fields = append(fields, FieldInfo{Path: cur, Type: "object(" + componentNameFromRef(r) + ")", Desc: p.Get("description").String()})
                sub := getRefJson(j, r)
                if sub != nil {
                    for sk, sp := range sub.GetJsonMap("properties") {
                        st := sp.Get("type").String()
                        if rr := sp.Get("$ref").String(); rr != "" { st = "object(" + componentNameFromRef(rr) + ")" }
                        if st == "array" { it2 := sp.GetJson("items"); if it2 != nil { if r2 := it2.Get("$ref").String(); r2 != "" { st = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { st = "array(" + it2.Get("type").String() + ")" } else { st = "array" } } else { st = "array" } }
                        fields = append(fields, FieldInfo{Path: cur + "." + sk, Type: st, Desc: sp.Get("description").String()})
                    }
                }
                continue
            }
            if t == "array" {
                it := p.GetJson("items")
                if it != nil {
                    if r2 := it.Get("$ref").String(); r2 != "" {
                        fields = append(fields, FieldInfo{Path: cur + "[]", Type: "array(object(" + componentNameFromRef(r2) + "))", Desc: p.Get("description").String()})
                        sub := getRefJson(j, r2)
                        if sub != nil {
                            for sk, sp := range sub.GetJsonMap("properties") {
                                st := sp.Get("type").String()
                                if rr := sp.Get("$ref").String(); rr != "" { st = "object(" + componentNameFromRef(rr) + ")" }
                                if st == "array" { it2 := sp.GetJson("items"); if it2 != nil { if r3 := it2.Get("$ref").String(); r3 != "" { st = "array(object(" + componentNameFromRef(r3) + "))" } else if it2.Get("type").String() != "" { st = "array(" + it2.Get("type").String() + ")" } else { st = "array" } } else { st = "array" } }
                                fields = append(fields, FieldInfo{Path: cur + "[]." + sk, Type: st, Desc: sp.Get("description").String()})
                            }
                        }
                        continue
                    }
                    if it.Get("type").String() == "object" {
                        props2 := it.GetJsonMap("properties")
                        k2 := make([]string, 0, len(props2)); for kk := range props2 { k2 = append(k2, kk) }; sortStrings(k2)
                        for _, kk := range k2 {
                            sp := props2[kk]
                            st := sp.Get("type").String()
                            if rr := sp.Get("$ref").String(); rr != "" { st = "object(" + componentNameFromRef(rr) + ")" }
                            if st == "array" { it3 := sp.GetJson("items"); if it3 != nil { if r3 := it3.Get("$ref").String(); r3 != "" { st = "array(object(" + componentNameFromRef(r3) + "))" } else if it3.Get("type").String() != "" { st = "array(" + it3.Get("type").String() + ")" } else { st = "array" } } else { st = "array" } }
                            fields = append(fields, FieldInfo{Path: cur + "[]." + kk, Type: st, Desc: sp.Get("description").String()})
                        }
                        continue
                    }
                    t = "array(" + it.Get("type").String() + ")"
                } else { t = "array" }
            } else if t == "" { t = "object" }
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
                    tt := it.Get("type").String(); desc := it.Get("description").String()
                    if tt != "" { b.WriteString("<tr><td>" + htmlEscape(prefix+"[]") + "</td><td>" + htmlEscape("array("+tt+")") + "</td><td>" + htmlEscape(desc) + "</td></tr>") } else { b.WriteString("<tr><td colspan=3>无字段</td></tr>") }
                } else {
                    keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
                    for _, k := range keys {
                        p := props[k]
                        t := p.Get("type").String()
                        if r := p.Get("$ref").String(); r != "" { t = "object(" + componentNameFromRef(r) + ")" } else if t == "array" {
                            it2 := p.GetJson("items")
                            if it2 != nil {
                                if r2 := it2.Get("$ref").String(); r2 != "" { t = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { t = "array(" + it2.Get("type").String() + ")" } else { t = "array" }
                            } else { t = "array" }
                        } else if t == "" { t = "object" }
                        desc := p.Get("description").String(); path := prefix + "[]." + k
                        b.WriteString("<tr><td>" + htmlEscape(path) + "</td><td>" + htmlEscape(t) + "</td><td>" + htmlEscape(desc) + "</td></tr>")
                    }
                }
            } else { b.WriteString("<tr><td colspan=3>无字段</td></tr>") }
        } else {
            props := target.GetJsonMap("properties")
            if len(props) == 0 { b.WriteString("<tr><td colspan=3>无字段</td></tr>") } else {
                keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
                for _, k := range keys {
                    p := props[k]
                    t := p.Get("type").String()
                    if r := p.Get("$ref").String(); r != "" { t = "object(" + componentNameFromRef(r) + ")" } else if t == "array" {
                        it2 := p.GetJson("items")
                        if it2 != nil {
                            if r2 := it2.Get("$ref").String(); r2 != "" { t = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { t = "array(" + it2.Get("type").String() + ")" } else { t = "array" }
                        } else { t = "array" }
                    } else if t == "" { t = "object" }
                    desc := p.Get("description").String(); path := k; if prefix != "" { path = prefix + "." + k }
                    b.WriteString("<tr><td>" + htmlEscape(path) + "</td><td>" + htmlEscape(t) + "</td><td>" + htmlEscape(desc) + "</td></tr>")
                }
            }
        }
    }
    for _, f := range fields { b.WriteString("<tr><td>" + htmlEscape(f.Path) + "</td><td>" + htmlEscape(f.Type) + "</td><td>" + htmlEscape(f.Desc) + "</td></tr>") }
    b.WriteString("</tbody></table>")
    return b.String()
}
func renderResponseParamFlatTableHTMLFromRef(j *gjson.Json, ref string) string {
    sj := getSchema(j, componentNameFromRef(ref)); if sj == nil { return "" }
    return renderResponseParamFlatTableHTMLFromJson(j, sj)
}
func renderResponseParamFlatTableMarkdownFromJson(j *gjson.Json, sj *gjson.Json) string {
    target := sj
    if !sj.Get("properties.data").IsNil() { d := sj.GetJson("properties.data"); if r := d.Get("$ref").String(); r != "" { target = getSchema(j, componentNameFromRef(r)) } else { target = d } }
    prefix := ""; if target != sj { prefix = "data" }
    var fields []FieldInfo
    if target.Get("type").String() == "array" {
        it := target.GetJson("items")
        if it != nil {
            if r := it.Get("$ref").String(); r != "" {
                sub := getRefJson(j, r)
                if sub != nil {
                    for k, p := range sub.GetJsonMap("properties") {
                        t := p.Get("type").String()
                        if rr := p.Get("$ref").String(); rr != "" { t = "object(" + componentNameFromRef(rr) + ")" }
                        if t == "array" { it2 := p.GetJson("items"); if it2 != nil { if r2 := it2.Get("$ref").String(); r2 != "" { t = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { t = "array(" + it2.Get("type").String() + ")" } else { t = "array" } } else { t = "array" } }
                        fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
                    }
                }
            } else if it.Get("type").String() == "object" {
                props := it.GetJsonMap("properties")
                keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
                for _, k := range keys {
                    p := props[k]
                    t := p.Get("type").String()
                    if rr := p.Get("$ref").String(); rr != "" { t = "object(" + componentNameFromRef(rr) + ")" }
                    if t == "array" { it2 := p.GetJson("items"); if it2 != nil { if r2 := it2.Get("$ref").String(); r2 != "" { t = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { t = "array(" + it2.Get("type").String() + ")" } else { t = "array" } } else { t = "array" } } else if t == "" { t = "object" }
                    fields = append(fields, FieldInfo{Path: prefix + "[]." + k, Type: t, Desc: p.Get("description").String()})
                }
            } else { tt := it.Get("type").String(); if tt != "" { fields = append(fields, FieldInfo{Path: prefix + "[]", Type: "array(" + tt + ")", Desc: it.Get("description").String()}) } }
        }
    } else {
        props := target.GetJsonMap("properties")
        keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
        for _, k := range keys {
            p := props[k]
            cur := k; if prefix != "" { cur = prefix + "." + k }
            t := p.Get("type").String()
            if r := p.Get("$ref").String(); r != "" {
                fields = append(fields, FieldInfo{Path: cur, Type: "object(" + componentNameFromRef(r) + ")", Desc: p.Get("description").String()})
                sub := getRefJson(j, r)
                if sub != nil {
                    for sk, sp := range sub.GetJsonMap("properties") {
                        st := sp.Get("type").String()
                        if rr := sp.Get("$ref").String(); rr != "" { st = "object(" + componentNameFromRef(rr) + ")" }
                        if st == "array" { it2 := sp.GetJson("items"); if it2 != nil { if r2 := it2.Get("$ref").String(); r2 != "" { st = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { st = "array(" + it2.Get("type").String() + ")" } else { st = "array" } } else { st = "array" } }
                        fields = append(fields, FieldInfo{Path: cur + "." + sk, Type: st, Desc: sp.Get("description").String()})
                    }
                }
                continue
            }
            if t == "array" {
                it := p.GetJson("items")
                if it != nil {
                    if r2 := it.Get("$ref").String(); r2 != "" {
                        fields = append(fields, FieldInfo{Path: cur + "[]", Type: "array(object(" + componentNameFromRef(r2) + "))", Desc: p.Get("description").String()})
                        sub := getRefJson(j, r2)
                        if sub != nil {
                            for sk, sp := range sub.GetJsonMap("properties") {
                                st := sp.Get("type").String()
                                if rr := sp.Get("$ref").String(); rr != "" { st = "object(" + componentNameFromRef(rr) + ")" }
                                if st == "array" { it2 := sp.GetJson("items"); if it2 != nil { if r3 := it2.Get("$ref").String(); r3 != "" { st = "array(object(" + componentNameFromRef(r3) + "))" } else if it2.Get("type").String() != "" { st = "array(" + it2.Get("type").String() + ")" } else { st = "array" } } else { st = "array" } }
                                fields = append(fields, FieldInfo{Path: cur + "[]." + sk, Type: st, Desc: sp.Get("description").String()})
                            }
                        }
                        continue
                    }
                    if it.Get("type").String() == "object" {
                        props2 := it.GetJsonMap("properties")
                        k2 := make([]string, 0, len(props2)); for kk := range props2 { k2 = append(k2, kk) }; sortStrings(k2)
                        for _, kk := range k2 {
                            sp := props2[kk]
                            st := sp.Get("type").String()
                            if rr := sp.Get("$ref").String(); rr != "" { st = "object(" + componentNameFromRef(rr) + ")" }
                            if st == "array" { it3 := sp.GetJson("items"); if it3 != nil { if r3 := it3.Get("$ref").String(); r3 != "" { st = "array(object(" + componentNameFromRef(r3) + "))" } else if it3.Get("type").String() != "" { st = "array(" + it3.Get("type").String() + ")" } else { st = "array" } } else { st = "array" } }
                            fields = append(fields, FieldInfo{Path: cur + "[]." + kk, Type: st, Desc: sp.Get("description").String()})
                        }
                        continue
                    }
                    t = "array(" + it.Get("type").String() + ")"
                } else { t = "array" }
            } else if t == "" { t = "object" }
            fields = append(fields, FieldInfo{Path: cur, Type: t, Desc: p.Get("description").String()})
        }
    }
    var b strings.Builder
    b.WriteString("| 字段 | 类型 | 说明 |\n|---|---|---|\n")
    if len(fields) == 0 {
        if target.Get("type").String() == "array" {
            it := target.GetJson("items")
            if it != nil {
                props := it.GetJsonMap("properties")
                if len(props) == 0 {
                    tt := it.Get("type").String(); desc := it.Get("description").String()
                    if tt != "" { b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", prefix+"[]", "array("+tt+")", desc)) } else { b.WriteString("| 无字段 |  |  |\n") }
                } else {
                    keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
                    for _, k := range keys {
                        p := props[k]
                        t := p.Get("type").String()
                        if r := p.Get("$ref").String(); r != "" { t = "object(" + componentNameFromRef(r) + ")" } else if t == "array" {
                            it2 := p.GetJson("items")
                            if it2 != nil {
                                if r2 := it2.Get("$ref").String(); r2 != "" { t = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { t = "array(" + it2.Get("type").String() + ")" } else { t = "array" }
                            } else { t = "array" }
                        } else if t == "" { t = "object" }
                        desc := p.Get("description").String(); path := prefix + "[]." + k
                        b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", path, t, desc))
                    }
                }
            } else { b.WriteString("| 无字段 |  |  |\n") }
        } else {
            props := target.GetJsonMap("properties")
            if len(props) == 0 { b.WriteString("| 无字段 |  |  |\n") } else {
                keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
                for _, k := range keys {
                    p := props[k]
                    t := p.Get("type").String()
                    if r := p.Get("$ref").String(); r != "" { t = "object(" + componentNameFromRef(r) + ")" } else if t == "array" {
                        it2 := p.GetJson("items")
                        if it2 != nil {
                            if r2 := it2.Get("$ref").String(); r2 != "" { t = "array(object(" + componentNameFromRef(r2) + "))" } else if it2.Get("type").String() != "" { t = "array(" + it2.Get("type").String() + ")" } else { t = "array" }
                        } else { t = "array" }
                    } else if t == "" { t = "object" }
                    desc := p.Get("description").String(); path := k; if prefix != "" { path = prefix + "." + k }
                    b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", path, t, desc))
                }
            }
        }
    }
    for _, f := range fields { b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", f.Path, f.Type, f.Desc)) }
    return b.String()
}
func renderResponseParamFlatTableMarkdownFromRef(j *gjson.Json, ref string) string {
    sj := getSchema(j, componentNameFromRef(ref)); if sj == nil { return "" }
    return renderResponseParamFlatTableMarkdownFromJson(j, sj)
}

// FieldInfo 用于参数/返回说明的扁平表结构。
type FieldInfo struct { Path string; Required bool; Type string; Desc string }

// flattenSchemaFields 基于 $ref 展开对象的所有子字段到扁平行（请求参数）。
func flattenSchemaFields(j *gjson.Json, ref string, prefix string) []FieldInfo {
    name := componentNameFromRef(ref)
    sj := getSchema(j, name); if sj == nil { return nil }
    fields := make([]FieldInfo, 0, 16)
    props := sj.GetJsonMap("properties")
    requiredSet := setFromArray(sj.Get("required").Array())
    keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
    for _, k := range keys {
        pj := props[k]
        req := requiredSet[k]
        typ := pj.Get("type").String()
        desc := pj.Get("description").String()
        ref2 := pj.Get("$ref").String()
        curPath := k; if prefix != "" { curPath = prefix + "." + k }
        if typ == "array" {
            it := pj.GetJson("items")
            if it != nil {
                if r := it.Get("$ref").String(); r != "" {
                    fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(object(" + componentNameFromRef(r) + "))", Desc: desc})
                    sub := flattenSchemaFields(j, r, curPath+"[]")
                    fields = append(fields, sub...)
                    continue
                }
                fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(" + it.Get("type").String() + ")", Desc: desc})
                continue
            }
            fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array", Desc: desc})
            continue
        }
        if ref2 != "" {
            fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: "object(" + componentNameFromRef(ref2) + ")", Desc: desc})
            sub := flattenSchemaFields(j, ref2, curPath)
            fields = append(fields, sub...)
            continue
        }
        if typ == "object" {
            fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: "object", Desc: desc})
            nestedProps := pj.GetJsonMap("properties")
            if len(nestedProps) > 0 {
                nestedReq := setFromArray(pj.Get("required").Array())
                nkeys := make([]string, 0, len(nestedProps)); for nk := range nestedProps { nkeys = append(nkeys, nk) }; sortStrings(nkeys)
                for _, nk := range nkeys {
                    np := nestedProps[nk]
                    nreq := nestedReq[nk]
                    ntyp := np.Get("type").String()
                    ndesc := np.Get("description").String()
                    nref := np.Get("$ref").String()
                    npath := curPath + "." + nk
                    if ntyp == "array" {
                        it := np.GetJson("items")
                        if it != nil {
                            if r := it.Get("$ref").String(); r != "" {
                                fields = append(fields, FieldInfo{Path: npath + "[]", Required: nreq, Type: "array(object(" + componentNameFromRef(r) + "))", Desc: ndesc})
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
                        fields = append(fields, FieldInfo{Path: npath, Required: nreq, Type: "object(" + componentNameFromRef(nref) + ")", Desc: ndesc})
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

// flattenSchemaFieldsFromJson 从内联 schema 展开到扁平行（请求/返回参数），兼容 allOf/oneOf/anyOf、数组 items、$ref。
func flattenSchemaFieldsFromJson(j *gjson.Json, sj *gjson.Json, prefix string) []FieldInfo {
    if sj == nil { return nil }
    fields := make([]FieldInfo, 0, 16)
    props := sj.GetJsonMap("properties")
    if len(props) == 0 {
        for _, arrName := range []string{"allOf", "oneOf", "anyOf"} {
            for _, it := range sj.Get(arrName).Array() {
                elem := gjson.New(it)
                if r := elem.Get("$ref").String(); r != "" {
                    sub := getRefJson(j, r)
                    if sub != nil { for k, v := range sub.GetJsonMap("properties") { props[k] = v } }
                } else {
                    for k, v := range elem.GetJsonMap("properties") { props[k] = v }
                }
            }
        }
    }
    requiredSet := setFromArray(sj.Get("required").Array())
    keys := make([]string, 0, len(props)); for k := range props { keys = append(keys, k) }; sortStrings(keys)
    for _, k := range keys {
        pj := props[k]
        req := requiredSet[k]
        typ := pj.Get("type").String()
        desc := pj.Get("description").String()
        ref2 := pj.Get("$ref").String()
        curPath := k; if prefix != "" { curPath = prefix + "." + k }
        if typ == "array" {
            it := pj.GetJson("items")
            if it != nil {
                if r := it.Get("$ref").String(); r != "" {
                    fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(object(" + componentNameFromRef(r) + "))", Desc: desc})
                    sub := getRefJson(j, r); if sub != nil { fields = append(fields, flattenSchemaFieldsFromJson(j, sub, curPath+"[]")...) }
                    continue
                }
                fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array(" + it.Get("type").String() + ")", Desc: desc})
                continue
            }
            fields = append(fields, FieldInfo{Path: curPath + "[]", Required: req, Type: "array", Desc: desc})
            continue
        }
        if ref2 != "" {
            fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: "object(" + componentNameFromRef(ref2) + ")", Desc: desc})
            sub := getRefJson(j, ref2); if sub != nil { fields = append(fields, flattenSchemaFieldsFromJson(j, sub, curPath)...) }
            continue
        }
        if typ == "object" {
            fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: "object", Desc: desc})
            fields = append(fields, flattenSchemaFieldsFromJson(j, pj, curPath)...) // 递归内联对象
            continue
        }
        fields = append(fields, FieldInfo{Path: curPath, Required: req, Type: typ, Desc: desc})
    }
    return fields
}

// presentMethods 返回 path 项中有效的 HTTP 方法列表（小写，排序）。
func presentMethods(pj *gjson.Json) []string {
    allowed := map[string]struct{}{"get": {}, "post": {}, "put": {}, "delete": {}, "patch": {}, "options": {}, "head": {}}
    mp := pj.Map()
    res := make([]string, 0, len(mp))
    for k := range mp { if _, ok := allowed[strings.ToLower(k)]; ok { res = append(res, strings.ToLower(k)) } }
    sortStrings(res)
    return res
}

// getRequestSchema 提取请求体的 schema（优先 $ref），返回 schema、内容类型与 $ref。
func getRequestSchema(j *gjson.Json, op *gjson.Json) (schema *gjson.Json, contentType, schemaRef string) {
    mp := op.GetJsonMap("requestBody.content")
    for ct, v := range mp {
        contentType = ct
        schemaRef = v.Get("schema.$ref").String()
        if schemaRef != "" { return getRefJson(j, schemaRef), contentType, schemaRef }
        s := v.GetJson("schema"); if s != nil { return s, contentType, "" }
    }
    return nil, "", ""
}

// getResponseSchema 提取响应体的 schema（优先选 200），支持 $ref 与内联。
func getResponseSchema(j *gjson.Json, op *gjson.Json) (schema *gjson.Json, contentType, schemaRef string) {
    resps := op.GetJsonMap("responses")
    codes := make([]string, 0, len(resps)); for c := range resps { codes = append(codes, c) }; sortStrings(codes)
    pick := ""; for _, c := range codes { if c == "200" { pick = c; break } }
    if pick == "" && len(codes) > 0 { pick = codes[0] }
    if pick == "" { return nil, "", "" }
    mp := resps[pick].GetJsonMap("content")
    for ct, v := range mp {
        contentType = ct
        schemaRef = v.Get("schema.$ref").String()
        if schemaRef != "" { return getRefJson(j, schemaRef), contentType, schemaRef }
        s := v.GetJson("schema"); if s != nil { return s, contentType, "" }
    }
    return nil, "", ""
}

// exampleJSON 基于 $ref 生成 JSON 示例文本。
func exampleJSON(j *gjson.Json, ref string) string { ex := exampleValue(j, ref); bs, _ := json.MarshalIndent(ex, "", "    "); return string(bs) }
// exampleJSONFromSchema 基于内联 schema 生成 JSON 示例文本。
func exampleJSONFromSchema(j *gjson.Json, s *gjson.Json) string { ex := exampleValueFromSchema(j, s); bs, _ := json.MarshalIndent(ex, "", "    "); return string(bs) }

// exampleValue 生成示例值（$ref）。
func exampleValue(j *gjson.Json, ref string) interface{} {
    name := componentNameFromRef(ref)
    sj := getSchema(j, name); if sj == nil { return nil }
    if sj.Get("type").String() == "object" || len(sj.GetJsonMap("properties")) > 0 {
        props := sj.GetJsonMap("properties")
        m := map[string]interface{}{}
        for k, pj := range props {
            if rr := pj.Get("$ref").String(); rr != "" { m[k] = exampleValue(j, rr); continue }
            typ := pj.Get("type").String()
            switch typ {
            case "string": m[k] = "string"
            case "integer": m[k] = 0
            case "number": m[k] = 0
            case "boolean": m[k] = false
            default: m[k] = nil
            }
        }
        return m
    }
    switch sj.Get("type").String() { case "string": return "string"; case "integer": return 0; case "number": return 0; case "boolean": return false }
    return nil
}

// exampleValueFromSchema 生成示例值（内联 schema）。
func exampleValueFromSchema(j *gjson.Json, sj *gjson.Json) interface{} {
    if sj == nil { return nil }
    if rr := sj.Get("$ref").String(); rr != "" { sub := getRefJson(j, rr); return exampleValueFromSchema(j, sub) }
    if sj.Get("type").String() == "array" {
        it := sj.GetJson("items")
        if it != nil {
            if r2 := it.Get("$ref").String(); r2 != "" { sub := getRefJson(j, r2); return []interface{}{exampleValueFromSchema(j, sub)} } else if it.Get("type").String() == "object" { return []interface{}{exampleValueFromSchema(j, it)} } else { return []interface{}{} }
        }
        return []interface{}{}
    }
    if sj.Get("type").String() == "object" || len(sj.GetJsonMap("properties")) > 0 {
        props := sj.GetJsonMap("properties"); m := map[string]interface{}{}
        for k, pj := range props {
            if rr := pj.Get("$ref").String(); rr != "" { sub := getRefJson(j, rr); m[k] = exampleValueFromSchema(j, sub); continue }
            typ := pj.Get("type").String()
            switch typ {
            case "string": m[k] = "string"
            case "integer": m[k] = 0
            case "number": m[k] = 0
            case "boolean": m[k] = false
            case "object": m[k] = exampleValueFromSchema(j, pj)
            case "array":
                it := pj.GetJson("items")
                if it != nil {
                    if r2 := it.Get("$ref").String(); r2 != "" { sub := getRefJson(j, r2); m[k] = []interface{}{exampleValueFromSchema(j, sub)} } else if it.Get("type").String() == "object" { m[k] = []interface{}{exampleValueFromSchema(j, it)} } else { m[k] = []interface{}{} }
                } else { m[k] = []interface{}{} }
            default: m[k] = nil
            }
        }
        return m
    }
    switch sj.Get("type").String() { case "string": return "string"; case "integer": return 0; case "number": return 0; case "boolean": return false }
    return nil
}

// getRefJson 解析 components 下的 $ref（schemas/parameters）。
func getRefJson(j *gjson.Json, ref string) *gjson.Json {
    if strings.HasPrefix(ref, "#/components/schemas/") { if j == nil { return nil }; return getSchema(j, componentNameFromRef(ref)) }
    if strings.HasPrefix(ref, "#/components/parameters/") { if j == nil { return nil }; pm := j.GetJsonMap("components.parameters"); return pm[componentNameFromRef(ref)] }
    return nil
}

// getSchema 获取 components.schemas 下指定名称的 schema。
func getSchema(j *gjson.Json, name string) *gjson.Json { if j == nil { return nil }; sm := j.GetJsonMap("components.schemas"); return sm[name] }

// htmlEscape 安全转义 HTML。
func htmlEscape(s string) string { s = strings.ReplaceAll(s, "&", "&amp;"); s = strings.ReplaceAll(s, "<", "&lt;"); s = strings.ReplaceAll(s, ">", "&gt;"); s = strings.ReplaceAll(s, "\"", "&quot;"); return s }
// setFromArray 将字符串数组构建为集合。
func setFromArray(arr []interface{}) map[string]bool { m := make(map[string]bool); for _, v := range arr { if s, ok := v.(string); ok { m[s] = true } }; return m }
// jsonArrayStrings 提取字符串数组。
func jsonArrayStrings(arr []interface{}) []string { res := make([]string, 0, len(arr)); for _, v := range arr { if s, ok := v.(string); ok { res = append(res, s) } }; return res }
// sortStrings 原地执行稳定的升序排序。
func sortStrings(a []string) { if len(a) < 2 { return }; for i := 1; i < len(a); i++ { v := a[i]; j := i - 1; for j >= 0 && a[j] > v { a[j+1] = a[j]; j-- }; a[j+1] = v } }
// anchorID 生成 endpoint 的锚点 ID。
func anchorID(method, path string) string { s := method + "-" + path; s = strings.ReplaceAll(s, "/", "-"); s = strings.ReplaceAll(s, "{", ""); s = strings.ReplaceAll(s, "}", ""); s = strings.ReplaceAll(s, " ", "-"); return s }
// slugify 将分组标题转为锚点友好的短串。
func slugify(s string) string { s = strings.TrimSpace(s); s = strings.ToLower(s); r := strings.NewReplacer(" ", "-", "/", "-", ".", "-", "(", "", ")", "", "[", "", "]", ""); s = r.Replace(s); return s }
// splitTagParts 将 tags[0] 按“主/次”分组，例如 “用户模块/用户管理”。
func splitTagParts(tags []string) (string, string) { if len(tags) == 0 { return "未分组", "默认" }; s := tags[0]; i := strings.Index(s, "/"); if i < 0 { return s, "默认" }; pre := s[:i]; suf := s[i+1:]; if suf == "" { suf = "默认" }; return pre, suf }
// componentNameFromRef 从 $ref 中提取末尾组件名。
func componentNameFromRef(ref string) string { i := strings.LastIndex(ref, "/"); if i < 0 { return "" }; return ref[i+1:] }

