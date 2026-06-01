package admintemplates

import (
	"fmt"
	"text/template"
)

// validateTemplateBody returns an error if `body` doesn't parse as
// Go text/template. This is the fastest way to catch typos like
// `{{.Share` (missing closing braces) before a broken template
// reaches the renderer at send time.
func validateTemplateBody(body string) error {
	if _, err := template.New("validate").Parse(body); err != nil {
		return fmt.Errorf("template parse failed: %w", err)
	}
	return nil
}
