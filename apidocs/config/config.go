package config

import "github.com/megatrZlp/go-apidocs/apidocs/render"

// Config 用于自定义路由和预处理钩子。
// - RouteDocs: 文档页面路由（默认 /docs）
// - RouteMarkdown: Markdown 导出路由（默认 /docs.md）
// - Preprocess: 在注册后允许外部对 Server 进行预处理（可选）
type Config struct {
	// RouteDocs 文档页面路由（默认 /docs）
	RouteDocs string
	// RouteMarkdown Markdown 导出路由（默认 /docs.md）
	RouteMarkdown string
	// Preprocess 在注册完成后允许调用方对 Server 进行预处理（例如替换默认规范）
	Preprocess func(spec interface{}) interface{}
	// Domain+Port+Path 组合用于根据请求查询参数拼接远程 OpenAPI 源地址。
	// 示例：当请求为 /docs?uuid=123&env=prod，且 Domain=api.example.com, Port=80, Path=/openapi
	// 将拼接 http://api.example.com/openapi?uuid=123&env=prod 并拉取规范。
	Domain      string
	Port        int
	Path        string
	Customize   map[string]CustomizeReqAndRes
	TemplateDir string
}

type CustomizeReqAndRes struct {
	Headers  map[string]string
	Request  []string
	Response []string
}

// WithDefaults 返回带默认值的配置副本。
func (c *Config) WithDefaults() Config {
	d := *c
	if d.RouteDocs == "" {
		d.RouteDocs = "/docs"
	}
	if d.RouteMarkdown == "" {
		d.RouteMarkdown = "/docs.md"
	}
	d.Customize = d.customize()
	return d
}

// customize 返回“按接口路径”配置的定制规则集合，用于渲染时注入 Header、以及对请求/返回参数与示例进行白名单过滤。
// 过滤规则说明：
// - Request/Response 未设置（nil）时，不做过滤，完整展示；设置为非空切片时，按白名单过滤
// - Response 过滤仅作用于 data 内部的字段，顶层 code/message/data 永远保留
// - 白名单支持两类写法：
//  1. 叶子名：例如 "completeCode"，会保留 data 下所有名为 completeCode 的最底层基础类型字段
//  2. 完整路径：支持数组写法（.items[] 或 [] 等价），例如
//     "data.departments.items[].address"、"data.professions.items[].code"
//
// - Request 采用与 Response 相同的白名单规则（按 data 下叶子或完整路径过滤）；未设置则不过滤
// - Header 注入：Headers 的值格式为 "type#required#desc" 或 "type#desc"，其中 required/optional（或 必选/可选）会被解析为“是/否”，type 与 desc 分别写入类型与说明
func (c *Config) customize() map[string]CustomizeReqAndRes {
	customize := make(map[string]CustomizeReqAndRes)
	customize["/v1/record/record/*"] = CustomizeReqAndRes{
		Headers: map[string]string{
			"accessToken": "string#required#登录获取的accessToken",
		},
		Request:  nil,
		Response: nil,
	}
	return customize
}

// ToRenderConfig 映射到渲染配置结构
func (c Config) ToRenderConfig() render.RenderConfig {
	m := make(map[string]render.CustomizeReqAndRes, len(c.Customize))
	for k, v := range c.Customize {
		m[k] = render.CustomizeReqAndRes{Headers: v.Headers, Request: v.Request, Response: v.Response}
	}
	return render.RenderConfig{RouteMarkdown: c.RouteMarkdown, Customize: m, TemplateDir: c.TemplateDir}
}
