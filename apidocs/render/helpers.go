package render

import (
	"github.com/megatrZlp/go-apidocs/apidocs/tools"

	"github.com/gogf/gf/v2/encoding/gjson"
)

// getRefJson 解析 components 下的 $ref（支持 schemas/parameters 两类引用）。
func getRefJson(j *gjson.Json, ref string) *gjson.Json {
	return tools.GetRefJSON(j, ref)
}

// getSchema 获取 components.schemas 下指定名称的 schema。
func getSchema(j *gjson.Json, name string) *gjson.Json {
	return tools.GetSchema(j, name)
}

// htmlEscape 安全转义 HTML（避免表格与 code 中出现未转义的特殊字符）。
func htmlEscape(s string) string {
	return tools.HTMLEscape(s)
}

// setFromArray 将字符串数组构建为集合。
func setFromArray(arr []interface{}) map[string]bool {
	return tools.SetFromArray(arr)
}

// jsonArrayStrings 提取字符串数组。
func jsonArrayStrings(arr []interface{}) []string {
	return tools.JSONArrayStrings(arr)
}

// sortStrings 原地执行稳定的升序排序。
func sortStrings(a []string) {
	tools.SortStrings(a)
}

// anchorID 生成 endpoint 的锚点 ID。
func anchorID(method, path string) string {
	return tools.AnchorID(method, path)
}

// slugify 将分组标题转为锚点友好的短串。
func slugify(s string) string {
	return tools.Slugify(s)
}

// splitTagParts 将 tags[0] 按“主/次”分组，例如 “用户模块/用户管理”。
func splitTagParts(tags []string) (string, string) {
	return tools.SplitTagParts(tags)
}

// componentNameFromRef 从 $ref 中提取末尾组件名。
func componentNameFromRef(ref string) string {
	return tools.ComponentNameFromRef(ref)
}

// stripComponentTypeDecorations 规范化类型显示，移除如 object(Component) 与 array(object(Component)) 的组件名装饰，
// 使表格与 Markdown 输出中的类型更简洁（只保留 object/array(object) 等）。
func stripComponentTypeDecorations(s string) string {
	return tools.StripComponentTypeDecorations(s)
}
