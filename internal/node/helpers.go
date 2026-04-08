package node

import (
	"bytes"
	"text/template"
)

// renderTemplate renders a Go template string with the provided data.
// This is a shared helper used by multiple node types (AI, Notify, etc.)
func renderTemplate(tmpl string, data map[string]interface{}) (string, error) {
	t, err := template.New("template").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderTemplateNamed renders a Go template with a custom template name.
// Used by nodes that need more control over template naming (e.g., NotifyNode)
func renderTemplateNamed(name, tmpl string, data interface{}) (string, error) {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// truncateString truncates a string to maxLen characters and adds "..." if truncated.
// Used for logging and display purposes.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
