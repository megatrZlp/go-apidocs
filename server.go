package apidocs

import "github.com/gogf/gf/v2/net/ghttp"

// RegisterWithConfig 按照给定配置注册文档页面与导出路由。
// - s: GoFrame 服务器
// - defaultContent: 默认 OpenAPI 文本（未提供 src 参数时使用）
// - cfg: 路由与预处理配置
// 返回：Server 结构体，持有默认规范与原始文本
func RegisterWithConfig(s *ghttp.Server, defaultContent string, cfg Config) *Server {
    c := cfg.withDefaults()
    srv := Register(s, defaultContent)
    if c.Preprocess != nil { srv = c.Preprocess(srv) }
    // 文档页面
    s.BindHandler("GET:"+c.RouteDocs, func(r *ghttp.Request) {
        spec := srv.spec
        raw := srv.raw
        // src 参数支持本地/远程地址，且对 Windows 路径和片段（#Lx-y）做归一化
        if src := r.Get("src").String(); src != "" {
            if j2, content, err := loadSpecFromSource(src); err == nil && j2 != nil { spec = j2; raw = content }
        }
        r.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
        r.Response.Write(generateHTML(spec, raw))
    })
    // Markdown 导出
    s.BindHandler("GET:"+c.RouteMarkdown, func(r *ghttp.Request) {
        spec := srv.spec
        raw := srv.raw
        if src := r.Get("src").String(); src != "" {
            if j2, content, err := loadSpecFromSource(src); err == nil && j2 != nil { spec = j2; raw = content }
        }
        md := generateMarkdown(spec, raw)
        r.Response.Header().Set("Content-Type", "text/markdown; charset=utf-8")
        r.Response.Header().Set("Content-Disposition", "attachment; filename=api-docs.md")
        r.Response.Write(md)
    })
    return srv
}

