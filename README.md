# go-apidocs 使用说明

`github.com/megatrZlp/go-apidocs` 基于 OpenAPI 规范生成可交互的 HTML/Markdown 文档，保持 `paths` 原始顺序，并支持按接口路径定制 Header 与请求/返回的白名单过滤。

## 特性概览
- 侧边菜单按 `tags[0]` 的“主/次”递归分组，滚动联动与高亮
- 请求/返回示例自动生成，支持 `$ref` 与内联 schema
- 按接口路径定制：Header 注入、请求/返回字段白名单过滤（支持 `*` 通配）
- 一键导出 Markdown，顺序与 HTML 保持一致
- 模板可自定义（`TemplateDir`），前端完全模板化

## 安装
- 运行环境：`Go 1.23+`
- 依赖：`github.com/gogf/gf/v2 v2.9.6`
- 获取模块：
```
go get github.com/megatrZlp/go-apidocs@latest
```

## 快速开始（作为服务）
```go
package main

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
        //TemplateDir: "你的模板目录",
    })
    s.SetPort(8000)
    s.Run()
}
```
- 文档页面：`http://localhost:8000/docs`
- 导出 Markdown：`http://localhost:8000/docs.md`

## 快速开始（作为库）
可直接生成 HTML/Markdown 字符串用于嵌入你自己的页面或导出：
```go
import (
    "github.com/gogf/gf/v2/encoding/gjson"
    "github.com/megatrZlp/go-apidocs/apidocs"
    "github.com/gogf/gf/v2/os/gfile"
)

raw := gfile.GetContents("./api.json")
spec, _ := gjson.LoadContent([]byte(raw))
html := apidocs.HTML(spec, raw)
md := apidocs.Markdown(spec, raw)
```

## 路由与配置
`config.Config` 支持以下字段：
- `RouteDocs`：文档页面路由，默认 `/docs`
- `RouteMarkdown`：Markdown 导出路由，默认 `/docs.md`
- `Preprocess`：注册完成后对 `Server` 进行预处理的回调
- `Domain/Port/Path`：远程源拼接；把请求查询参数（排除 `src`）拼到 `http://Domain:Port/Path` 拉取规范
- `Customize`：按接口路径的定制规则集合
- `TemplateDir`：模板目录，覆盖内置模板

示例（自定义路由与预处理）：
```go
cfg := config.Config{
    RouteDocs:     "/api-docs",
    RouteMarkdown: "/api-docs.md",
    Preprocess: func(s interface{}) interface{} { return s },
}
```

## 数据源选择与转发
- 显式 `src` 优先：`/docs?src=file://D:/path/api.json` 或 `src=http://host/openapi`
- 当设置了 `Domain/Port/Path` 时，请求中的查询参数（排除 `src`）会拼接到远程地址并拉取规范
- 支持 Windows 路径归一化与片段移除（`#/Lx-y`）

## 按接口路径定制（`Config.Customize`）
在 `Customize[接口路径或通配]` 下配置：
- `Headers map[string]string`：注入 Header 参数，值格式：`type#required#desc` 或 `type#desc`
- `Request []string`：请求体字段白名单
- `Response []string`：返回体字段白名单

过滤规则：
- 未设置（nil）时不过滤；设置为非空切片时按白名单过滤
- Response 过滤仅作用于 `data` 内部字段，顶层 `code/message/data` 永远保留；Request 采用相同规则
- 白名单写法：
  - 叶子名：如 `completeCode`，保留 `data` 下所有同名最底层基础类型
  - 完整路径：支持数组写法（`.items[]` 与 `[]` 等价），如 `data.departments.items[].address`

示例：
```go
cfg := config.Config{}
cfg.Customize = map[string]config.CustomizeReqAndRes{
    "/v1/record/record/*": {
        Headers: map[string]string{ "accessToken": "string#required#登录获取的accessToken" },
    },
}
```

## 模板自定义（`TemplateDir`）
- 将自定义模板目录传给 `TemplateDir` 即可覆盖内置模板
- 必需文件：`layout.tmpl`、`style.tmpl`、`script.tmpl`
- 可选文件：`main_header.tmpl`、`nav.tmpl`、`group_heading.tmpl`、`sub_heading.tmpl`、`endpoint.tmpl`
- 详见 `TEMPLATE_README.md`

## 页面行为与交互
- 菜单与正文均按 `paths` 原始顺序渲染，点击高亮并滚动联动到最近可见接口块
- 标题设置 `scroll-margin-top`，滚动定位更准确
- 侧边菜单固定“全部展开/收起”按钮

## 导出 Markdown
- 路由：`/docs.md`（可通过 `RouteMarkdown` 修改）
- 响应头：`Content-Type: text/markdown`，`Content-Disposition: attachment; filename=api-docs.md`
- 内容顺序与 HTML 一致，便于离线阅览

## 目录结构
- `apidocs/config/`：路由与定制配置
- `apidocs/source/`：数据源加载与 `paths` 顺序提取
- `apidocs/render/`：页面/Markdown 渲染、示例与参数表
- `apidocs/templates/`：内置模板片段
- `apidocs/tools/`：通用工具（转义、分组、锚点等）
- `main.go`：示例服务入口

## 常见问题
- 锚点偏移：由标题外边距引起，已通过 `scroll-margin-top` 缓解
- 菜单与内容顺序不一致：确认 `tags` 的分组字符串是否一致；渲染严格按 `paths` 原始顺序
- 示例与表格不一致：使用了白名单时，示例与表格都会按相同规则过滤 `data` 内部叶子

## 许可
内部项目示例，按需修改与使用
