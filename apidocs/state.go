package apidocs

import "github.com/gogf/gf/v2/encoding/gjson"

// Server 持有默认的 OpenAPI 解析对象与其原始文本，用于路由处理时回退。
// - spec: 解析后的 OpenAPI 结构
// - raw: 原始 JSON 文本（保持 paths 原始顺序）
type Server struct {
	spec *gjson.Json
	raw  string
}
