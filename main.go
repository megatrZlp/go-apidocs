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
		//TemplateDir: "apidocs/templates_custom",
	})
	s.SetPort(8000)
	s.Run()
}
