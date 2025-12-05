package apidocs

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/megatrZlp/go-apidocs/apidocs/config"
	"github.com/megatrZlp/go-apidocs/apidocs/render"
	"github.com/megatrZlp/go-apidocs/apidocs/source"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/net/ghttp"
)

// RegisterWithConfig 按照给定配置注册文档页面与导出路由。
// 参数：
// - s: GoFrame 服务器
// - defaultContent: 默认 OpenAPI 文本（未提供 src 参数时使用）
// - cfg: 路由与预处理配置（支持根据查询参数转发到远程源）
// 返回：Server 结构体，持有默认规范与原始文本。
// 说明：
// - 若配置了 Domain/Port/Path，则会将请求中的所有查询参数（排除 src）拼接到远程地址并拉取规范；
// - 若提供 src，则优先使用 src 指定的数据源；
// - 返回的 HTML/Markdown 页面始终保持 paths 原始顺序与锚点联动行为。
func RegisterWithConfig(s *ghttp.Server, defaultContent string, cfg config.Config) *Server {
	c := cfg.WithDefaults()
	var j *gjson.Json
	if defaultContent != "" {
		if jj, err := gjson.LoadContent([]byte(defaultContent)); err == nil {
			j = jj
		}
	}
	srv := &Server{spec: j, raw: defaultContent}
	if c.Preprocess != nil {
		v := c.Preprocess(srv)
		if vv, ok := v.(*Server); ok {
			srv = vv
		}
	}
	// 文档页面：GET /docs
	s.BindHandler("GET:"+c.RouteDocs, func(r *ghttp.Request) {
		spec := srv.spec
		raw := srv.raw
		// 转发查询参数到远程源：当配置了 Domain/Port/Path 时生效
		// 请求地址来源：Domain+Port+Path 组合；请求参数来源：r.GetMap()（排除 src）
		if c.Domain != "" && c.Path != "" {
			params := r.GetMap()
			base := "http://" + c.Domain
			if c.Port > 0 {
				base = base + fmt.Sprintf(":%d", c.Port)
			}
			p := c.Path
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			var b strings.Builder
			first := true
			for k, v := range params {
				if k == "src" {
					continue
				}
				if first {
					first = false
				} else {
					b.WriteByte('&')
				}
				b.WriteString(url.QueryEscape(k))
				b.WriteByte('=')
				b.WriteString(url.QueryEscape(fmt.Sprintf("%v", v)))
			}
			src := base + p
			if b.Len() > 0 {
				src = src + "?" + b.String()
			}
			if j2, content2, e := source.LoadSpecFromSource(src); e == nil && j2 != nil {
				spec = j2
				raw = content2
			}
		}
		// src 参数支持本地/远程地址，且对 Windows 路径和片段（#Lx-y）做归一化
		// 显式 src 优先于 Domain/Port/Path 转发逻辑
		if src := r.Get("src").String(); src != "" {
			if j2, content, err := source.LoadSpecFromSource(src); err == nil && j2 != nil {
				spec = j2
				raw = content
			}
		}
		srv.spec = spec
		srv.raw = raw
		r.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		r.Response.Header().Set("X-OpenAPI-Source", r.Get("src").String())
		r.Response.Header().Set("X-Paths-Count", fmt.Sprintf("%d", len(spec.GetJsonMap("paths"))))
		r.Response.Write(render.GenerateHTMLWithConfig(spec, raw, c.ToRenderConfig()))
	})
	// Markdown 导出：GET /docs.md
	s.BindHandler("GET:"+c.RouteMarkdown, func(r *ghttp.Request) {
		spec := srv.spec
		raw := srv.raw
		// 同文档页面逻辑：优先根据配置转发查询参数以拉取规范
		if c.Domain != "" && c.Path != "" {
			params := r.GetMap()
			base := "http://" + c.Domain
			if c.Port > 0 {
				base = base + fmt.Sprintf(":%d", c.Port)
			}
			p := c.Path
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			var b strings.Builder
			first := true
			for k, v := range params {
				if k == "src" {
					continue
				}
				if first {
					first = false
				} else {
					b.WriteByte('&')
				}
				b.WriteString(url.QueryEscape(k))
				b.WriteByte('=')
				b.WriteString(url.QueryEscape(fmt.Sprintf("%v", v)))
			}
			src := base + p
			if b.Len() > 0 {
				src = src + "?" + b.String()
			}
			if j2, content2, e := source.LoadSpecFromSource(src); e == nil && j2 != nil {
				spec = j2
				raw = content2
			}
		}
		// 显式 src 优先于转发逻辑，用于直接指定源
		if src := r.Get("src").String(); src != "" {
			if j2, content, err := source.LoadSpecFromSource(src); err == nil && j2 != nil {
				spec = j2
				raw = content
			}
		}
		srv.spec = spec
		srv.raw = raw
		r.Response.Header().Set("X-OpenAPI-Source", r.Get("src").String())
		r.Response.Header().Set("X-Paths-Count", fmt.Sprintf("%d", len(spec.GetJsonMap("paths"))))
		md := render.GenerateMarkdownWithConfig(spec, raw, c.ToRenderConfig())
		r.Response.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		r.Response.Header().Set("Content-Disposition", "attachment; filename=api-docs.md")
		r.Response.Write(md)
	})
	return srv
}
