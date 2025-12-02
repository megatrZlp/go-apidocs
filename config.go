package apidocs

// Config 用于自定义路由和预处理钩子。
// - RouteDocs: 文档页面路由（默认 /docs）
// - RouteMarkdown: Markdown 导出路由（默认 /docs.md）
// - Preprocess: 在注册后允许外部对 Server 进行预处理（可选）
type Config struct {
    RouteDocs     string
    RouteMarkdown string
    Preprocess    func(spec *Server) *Server
}

// withDefaults 返回带默认值的配置副本。
func (c *Config) withDefaults() Config {
    d := *c
    if d.RouteDocs == "" { d.RouteDocs = "/docs" }
    if d.RouteMarkdown == "" { d.RouteMarkdown = "/docs.md" }
    return d
}

