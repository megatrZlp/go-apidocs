package render

import (
	"html/template"
	"os"
	"path/filepath"

	tpl "github.com/megatrZlp/go-apidocs/apidocs/templates"
)

func loadTemplateContent(dir string, name string) (string, error) {
	// 优先读取用户配置的模板目录（绝对或相对路径均可）
	if dir != "" {
		p := filepath.Join(dir, name)
		if b, err := os.ReadFile(p); err == nil {
			return string(b), nil
		}
	}
	// 回退顺序：
	// 1) 项目中的 apidocs/templates
	// 2) 当前工作目录下的 templates（便于快速试验）
	fallbacks := []string{
		filepath.Join("apidocs", "templates", name),
		filepath.Join("templates", name),
	}
	for _, p := range fallbacks {
		if b, err := os.ReadFile(p); err == nil {
			return string(b), nil
		}
	}
	if s, err := tpl.Read(name); err == nil {
		return s, nil
	}
	return "", os.ErrNotExist
}

func buildLayoutTemplate(dir string) (*template.Template, error) {
	// 依次加载样式、脚本与布局模板，后续动态片段可选加载
	style, err := loadTemplateContent(dir, "style.tmpl")
	if err != nil {
		return nil, err
	}
	script, err := loadTemplateContent(dir, "script.tmpl")
	if err != nil {
		return nil, err
	}
	layout, err := loadTemplateContent(dir, "layout.tmpl")
	if err != nil {
		return nil, err
	}
	// 端点模板非必须，允许为空以便仅输出最简布局
	endpoint, err := loadTemplateContent(dir, "endpoint.tmpl")
	if err != nil {
		endpoint = ""
	}
	nav, err := loadTemplateContent(dir, "nav.tmpl")
	if err != nil {
		nav = ""
	}
	headings, err := loadTemplateContent(dir, "headings.tmpl")
	if err != nil {
		headings = ""
	}
	groupHeading, err := loadTemplateContent(dir, "group_heading.tmpl")
	if err != nil {
		groupHeading = ""
	}
	subHeading, err := loadTemplateContent(dir, "sub_heading.tmpl")
	if err != nil {
		subHeading = ""
	}
	mainHeader, err := loadTemplateContent(dir, "main_header.tmpl")
	if err != nil {
		mainHeader = ""
	}
	// 合并所有模板片段到同一个模板实例中
	t := template.New("layout")
	if _, err = t.Parse(style); err != nil {
		return nil, err
	}
	if _, err = t.Parse(script); err != nil {
		return nil, err
	}
	if _, err = t.Parse(layout); err != nil {
		return nil, err
	}
	if endpoint != "" {
		if _, err = t.Parse(endpoint); err != nil {
			return nil, err
		}
	}
	if nav != "" {
		if _, err = t.Parse(nav); err != nil {
			return nil, err
		}
	}
	if headings != "" {
		if _, err = t.Parse(headings); err != nil {
			return nil, err
		}
	}
	if mainHeader != "" {
		if _, err = t.Parse(mainHeader); err != nil {
			return nil, err
		}
	}
	if groupHeading != "" {
		if _, err = t.Parse(groupHeading); err != nil {
			return nil, err
		}
	}
	if subHeading != "" {
		if _, err = t.Parse(subHeading); err != nil {
			return nil, err
		}
	}
	return t, nil
}

type pageData struct {
	Title    string
	NavHTML  template.HTML
	MainHTML template.HTML
}

type EndpointData struct {
	Anchor          string
	MethodUpper     string
	Summary         string
	Path            string
	ContentType     string
	HeadersHTML     template.HTML
	PathParamsHTML  template.HTML
	QueryParamsHTML template.HTML
	ReqExample      template.HTML
	ReqTableHTML    template.HTML
	ResExample      template.HTML
	ResTableHTML    template.HTML
}

type NavItemVM struct {
	Summary string
	Anchor  string
}

type NavGroupVM struct {
	Name     string
	Id       string
	Items    []NavItemVM
	Children []*NavGroupVM
}

type NavData struct {
	Groups []*NavGroupVM
}

type MainGroupVM struct {
	PreName  string
	PreId    string
	Children []*NavGroupVM
}

type HeadingsData struct {
	MainGroups []*MainGroupVM
}

type MainHeaderData struct {
	Title   string
	MdRoute string
}

type GroupHeadingData struct {
	Id   string
	Name string
}

type SubHeadingData struct {
	Id   string
	Name string
}
