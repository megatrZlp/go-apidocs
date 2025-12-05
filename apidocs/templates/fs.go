package templates

import (
	"embed"
)

//go:embed *.tmpl
var Embedded embed.FS

func Read(name string) (string, error) {
	b, err := Embedded.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
