# github.com/megatrZlp/go-apidocs 使用说明

## 概览
`github.com/megatrZlp/go-apidocs` 基于 OpenAPI 规范生成可交互的 HTML/Markdown 文档，支持：
- 菜单分组递归展开，锚点联动与高亮
- 接口请求/返回示例生成
- 按接口路径应用定制：Header 注入、请求/返回参数白名单过滤
- 导出 Markdown（保持 paths 原始顺序）

## 快速开始
1. 将你的 OpenAPI JSON/YAML 内容读取为字符串（示例使用 `api.json`）：
```go
raw := gfile.GetContents("./api.json")
```
2. 注册路由：
```go
srv := ghttp.NewServer()
cfg := apidocs.Config{ Domain: "", Port: 0, Path: "", Customize: nil }
apidocs.RegisterWithConfig(srv, raw, cfg)
srv.SetPort(8000)
srv.Run()
```
3. 打开浏览器：
- 文档页面：`http://localhost:8000/docs`
- 导出 Markdown：`http://localhost:8000/docs.md`

## 数据源选择
- `src` 查询参数优先：`/docs?src=file:///D:/path/to/api.json` 或 `/docs?src=http://host/openapi`
- 转发查询参数到远程源：当 `Config.Domain/Port/Path` 设置后，`/docs?uuid=...&env=...` 会拼接到 `http://Domain:Port/Path` 拉取规范

## 自定义规则（Config.Customize）
在 `Config.Customize[path]` 下配置：
- `Headers map[string]string`: 注入 Header 行，值支持两种格式：
  - `type#required#desc`（或 `type#desc`）
  - `required/optional` 或 `必选/可选` 会被解析为“是/否”，`type` 与 `desc` 分别写入类型与说明
- `Request []string`: 请求体字段白名单
- `Response []string`: 返回体字段白名单

### 过滤规则
- 未设置（nil）时不过滤；设置为非空切片时按白名单过滤
- Response 过滤仅作用于 `data` 内部字段，顶层 `code/message/data` 永远保留；Request 采用相同规则
- 白名单写法：
  1) 叶子名：如 `completeCode`，保留 `data` 下所有同名叶子基础类型
  2) 完整路径：支持数组写法（`.items[]` 与 `[]` 等价），如 `data.departments.items[].address`

示例：
```go
cfg.Customize = map[string]apidocs.CustomizeReqAndRes{
  "/common/common/area/all": { Response: []string{"code", "completeCode"} },
  "/user/user/profile":     { Response: []string{"data.departments.items[].address", "data.professions.items[].code"} },
}
```

## 菜单与锚点
- 菜单分组按 `tags[0]` 的 "主/次/…" 拆分为递归树，保留 `paths` 原始顺序
- 点击菜单高亮锁定当前项；滚动联动优先匹配可见 `.endpoint`，无可见接口时回退到分组标题
- 标题设置 `scroll-margin-top`，滚动定位更准确

## 样式与交互
- 侧边菜单顶部固定“全部展开/收起”按钮
- 子分组与条目按层级缩进展示；字体大小可在 `apidocs.go` 的 CSS 写入处调整

## 常见问题
- “锚点偏移”：由标题外边距引起，已通过 `scroll-margin-top` 缓解
- “菜单与内容顺序不一致”：菜单与正文均按 `paths` 原始顺序渲染；若看到不一致，请确认 `tags` 的分组字符串是否一致
- “示例与表格不一致”：当配置了白名单，示例与表格都会按相同规则过滤 `data` 内部叶子

## 目录结构
- `apidocs/helpers.go`: 通用工具与锚点/分组辅助
- `apidocs/source.go`: OpenAPI 加载与路径顺序提取
- `apidocs/examples.go`: 请求/返回示例生成与白名单过滤
- `apidocs/nav.go`: 菜单树结构与标题渲染
- `apidocs/render.go`: 参数/返回表格渲染与字段展开
- `apidocs/apidocs.go`: 页面生成主流程与交互脚本

## 许可
内部项目示例，按需修改与使用。

