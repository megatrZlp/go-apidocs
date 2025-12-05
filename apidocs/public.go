package apidocs

import (
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/megatrZlp/go-apidocs/apidocs/render"
)

// HTML 生成完整的 API 文档 HTML 页面（含侧边导航、分组、锚点、示例与参数说明）。
// 参数：
// - spec: OpenAPI 规范的解析对象（gjson.Json）
// - raw: 规范的原始 JSON 文本，用于保持 paths 原始顺序（必须与 spec 一致）
// 返回：完整 HTML 页面字符串
func HTML(spec *gjson.Json, raw string) string { return render.GenerateHTML(spec, raw) }

// Markdown 生成与 HTML 结构一致的 Markdown 文档（用于导出）。
// 参数：
// - spec: OpenAPI 规范的解析对象（gjson.Json）
// - raw: 规范的原始 JSON 文本，用于保持 paths 原始顺序（必须与 spec 一致）
// 返回：完整 Markdown 文本
func Markdown(spec *gjson.Json, raw string) string { return render.GenerateMarkdown(spec, raw) }
