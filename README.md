# go-apidocs

一个可直接集成到 GoFrame 项目的 OpenAPI 文档页面/Markdown 导出库。

## 安装

```
go get github.com/yourname/go-apidocs
```

（首次发布后，将 `yourname` 替换为你的 GitHub 用户名）

## 快速开始

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/os/gfile"
    "github.com/yourname/go-apidocs"
)

func main() {
    raw := gfile.GetContents("api.json")
    s := g.Server()
    apidocs.Register(s, raw)
    s.SetPort(8000)
    s.Run()
}
```

访问：
- 页面：`http://localhost:8000/docs`
- 导出：`http://localhost:8000/docs.md`

支持动态切换文档源（本地/远程）：
- `http://localhost:8000/docs?src=d:/path/to/api1.json`
- `http://localhost:8000/docs.md?src=https://example.com/openapi.json`

## 自定义路由

```go
apidocs.RegisterWithConfig(s, raw, apidocs.Config{
    RouteDocs:     "/openapi",
    RouteMarkdown: "/openapi.md",
})
```

## 说明

- 保持 `paths` 原始顺序，确保菜单顺序与内容一致
- 递归解析请求/响应 schema，数组 `items` 与 `$ref` 会展开
- 响应的 `data` 字段（若存在）优先作为目标结构进行展开

