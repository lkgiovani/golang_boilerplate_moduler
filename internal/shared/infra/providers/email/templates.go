package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates *template.Template

func init() {
	templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))
}

// RenderTemplate executes the named HTML template with the given data and returns the rendered string.
func RenderTemplate(name string, data map[string]any) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name+".html", data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.String(), nil
}
