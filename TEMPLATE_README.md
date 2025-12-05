# 模板改造与自定义指南

本项目的前端页面已完全模板化，支持通过配置项 `TemplateDir` 指定外部模板目录覆盖内置模板。

## 模板文件列表

必需或可选的模板文件（文件名固定）：

- layout.tmpl（必需）：整页骨架，负责注入导航与正文
- style.tmpl（必需）：页面样式
- script.tmpl（必需）：交互脚本
- main_header.tmpl（可选）：正文顶部标题与“导出 Markdown”按钮
- nav.tmpl（可选）：侧边导航（递归渲染分组与子分组）
- group_heading.tmpl（可选）：分组 `h1` 标题
- sub_heading.tmpl（可选）：子分组 `h2` 标题
- endpoint.tmpl（可选）：接口详情区块（URL、方法、参数、示例等）

说明：未提供的模板文件会自动回退到内置模板，不会影响页面渲染。

## 模板数据结构与可用变量

所有模板均使用 Go `html/template` 引擎渲染。以下为每个模板的 `.Data` 结构与字段说明：

### layout.tmpl

- `.Title`：字符串，页面标题
- `.NavHTML`：HTML，侧边导航的已渲染片段
- `.MainHTML`：HTML，正文的已渲染片段

示例：

```html
{{define "layout"}}
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>{{template "style" .}}</style>
</head>
<body>
  <div class="layout">
    <aside>{{.NavHTML}}</aside>
    <main>{{.MainHTML}}</main>
  </div>
  <script>{{template "script" .}}</script>
</body>
</html>
{{end}}
```

### main_header.tmpl

- `.Title`：字符串，正文主标题
- `.MdRoute`：字符串，导出 Markdown 的路由

示例：

```html
{{define "main_header"}}
<div style="display:flex;align-items:center;justify-content:space-between"><h1>{{.Title}}</h1></div>
<a class="export-fixed" id="exportMd" href="{{.MdRoute}}" title="导出为 Markdown">导出为 Markdown</a>
{{end}}
```

### nav.tmpl（递归）

数据来源：`NavData`

- `.Groups`：`[]*NavGroupVM`
- `NavGroupVM.Name`：分组名称
- `NavGroupVM.Id`：分组锚点 id（如 `group-xxx` 或 `group-xxx-yyy`）
- `NavGroupVM.Items`：`[]NavItemVM`，其中 `NavItemVM.Summary` 为接口摘要，`NavItemVM.Anchor` 为接口锚点 id
- `NavGroupVM.Children`：`[]*NavGroupVM`，递归的子分组

示例（已内置）：

```html
{{define "nav"}}
<div class="nav">
  <div class="nav-top" style="display:flex;gap:8px;align-items:center">
    <button id="expandAll">全部展开</button>
    <button id="collapseAll">全部收起</button>
  </div>
  {{range .Groups}}
    <details class="grp" open>
      <summary><a href="#{{.Id}}">{{.Name}}</a></summary>
      {{range .Children}}
        {{template "navSubGroup" .}}
      {{end}}
    </details>
  {{end}}
</div>
{{end}}

{{define "navSubGroup"}}
<details class="subgrp" open>
  <summary><a href="#{{.Id}}">{{.Name}}</a></summary>
  {{range .Items}}
    <div class="item"><a class="item-link" href="#{{.Anchor}}">{{.Summary}}</a></div>
  {{end}}
  {{range .Children}}
    {{template "navSubGroup" .}}
  {{end}}
</details>
{{end}}
```

### group_heading.tmpl

- `.Id`：字符串，分组锚点 id（如 `group-xxx`）
- `.Name`：字符串，分组名称

示例：

```html
{{define "group_heading"}}
<h1 id="{{.Id}}">{{.Name}}</h1>
{{end}}
```

### sub_heading.tmpl

- `.Id`：字符串，子分组锚点 id（如 `group-xxx-yyy`）
- `.Name`：字符串，子分组名称

示例：

```html
{{define "sub_heading"}}
<h2 id="{{.Id}}">{{.Name}}</h2>
{{end}}
```

### endpoint.tmpl（接口详情）

数据来源：`EndpointData`

- `.Anchor`：接口块锚点 id
- `.MethodUpper`：HTTP 方法（大写）
- `.Summary`：接口摘要（已 HTML 转义）
- `.Path`：请求 URL（已 HTML 转义）
- `.ContentType`：内容类型（已 HTML 转义）
- `.HeadersHTML`：Header 参数表（HTML 片段）
- `.PathParamsHTML`：路径参数表（HTML 片段）
- `.QueryParamsHTML`：Query 参数表（HTML 片段）
- `.ReqExample`：请求示例（已转义的 `<pre><code>` 内容）
- `.ReqTableHTML`：请求参数表（HTML 片段）
- `.ResExample`：返回示例（已转义的 `<pre><code>` 内容）
- `.ResTableHTML`：返回参数说明（HTML 片段）

示例（已内置）：

```html
{{define "endpoint"}}
<div class="endpoint" id="{{.Anchor}}">
  <h2><span class="method">{{.MethodUpper}}</span> {{.Summary}}</h2>
  <h3 id="{{.Anchor}}-url">请求URL</h3>
  <pre><code>{{.Path}}</code></pre>
  <h3 id="{{.Anchor}}-method">请求方式</h3>
  <ul><li>{{.MethodUpper}} <em>Content-Type: {{.ContentType}}</em></li></ul>
  {{if .HeadersHTML}}<h3 id="{{.Anchor}}-headers">Header参数</h3>{{.HeadersHTML}}{{end}}
  {{if .PathParamsHTML}}<h3 id="{{.Anchor}}-path-params">路径参数</h3>{{.PathParamsHTML}}{{end}}
  {{if .QueryParamsHTML}}<h3 id="{{.Anchor}}-query-params">Query参数</h3>{{.QueryParamsHTML}}{{end}}
  {{if .ReqExample}}<h3 id="{{.Anchor}}-req-example">请求示例</h3><pre><code>{{.ReqExample}}</code></pre>{{end}}
  {{if .ReqTableHTML}}<h3 id="{{.Anchor}}-req">请求参数</h3>{{.ReqTableHTML}}{{end}}
  {{if .ResExample}}<h3 id="{{.Anchor}}-res-example">返回示例</h3><pre><code>{{.ResExample}}</code></pre>{{end}}
  {{if .ResTableHTML}}<h3 id="{{.Anchor}}-res-params">返回参数说明</h3>{{.ResTableHTML}}{{end}}
</div>
{{end}}
```

## 启用自定义模板

1. 在你的自定义目录中创建以上模板文件（至少 `layout.tmpl`、`style.tmpl`、`script.tmpl`）。
2. 在注册路由时设置：

```go
// main.go
import (
    "github.com/megatrZlp/go-apidocs/apidocs"
    "github.com/megatrZlp/go-apidocs/apidocs/config"
    "github.com/gogf/gf/v2/frame/g"
)
func main() {
    s := g.Server()
    apidocs.RegisterWithConfig(s, "", config.Config{
        Domain: "127.0.0.1",
        Port:   10014,
        Path:   "/server/swagger/api.json",
        TemplateDir: "你的模板目录",
    })
    s.SetPort(8000)
    s.Run()
}
```

3. 运行并访问：

- 页面：`http://localhost:8000/docs?src=file://<你的openapi.json>`
- 导出：`http://localhost:8000/docs.md?src=file://<你的openapi.json>`

## 渲染顺序与联动说明

- 正文渲染顺序：
  - `main_header` → 每个分组的 `group_heading`（遇到新分组时输出）→ 每个子分组的 `sub_heading`（首次遇到该子分组时输出）→ 接口 `endpoint`
- 菜单与标题锚点规则一致：分组 `group-<slug>`；子分组 `group-<slug>-<sub-slug>`；接口块为方法+路径规范化后的 `id`
- 页面脚本负责菜单联动高亮、展开/收起等交互；可在 `script.tmpl` 定制。

## 安全与转义

- 变量中 `*HTML` 类型字段为已生成的 HTML 片段，直接输出；其余字符串均已在代码中进行 HTML 转义后传入模板。
- 如需在模板中拼接未转义内容，请谨慎处理，避免 XSS 风险。

## 可扩展建议

- 如需新增模板文件（例如页脚、工具栏），可在 `apidocs/render/templates.go` 的 `buildLayoutTemplate` 中读取并 `Parse`，随后在 `layout.tmpl` 中通过 `{{template "你的模板名" .}}` 引用。
- 如需调整标题插入位置或分组粒度，可在 `apidocs/render/html.go` 中调整调用顺序或传入的数据结构（例如将接口渲染按子分组聚合）。

## 常用排错

- 页面空白或未按预期：检查 `TemplateDir` 指向是否正确，以及模板文件是否包含 `{{define "模板名"}} ... {{end}}`。
- 接口块未显示：确认 `endpoint.tmpl` 是否存在，或回退到内置模板；检查 `EndpointData` 字段是否被模板正确引用。
- 标题与内容顺序错误：确认未在模板中一次性整体渲染所有子分组标题；参照“渲染顺序与联动说明”。
