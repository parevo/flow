package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"
)

// NotifyNode sends an HTTP webhook / notification to a target URL.
// It supports Go template rendering in both the URL and the body,
// using the step's input JSON as template data.
//
// Config:
//   - "url"     (string, required) : destination URL (can contain {{.field}} templates)
//   - "method"  (string, optional) : HTTP method, defaults to POST
//   - "headers" (map, optional)    : extra headers e.g. {"Authorization": "Bearer xyz"}
//   - "body"    (string, optional) : Go template for the request body
//                                    If not set, the raw input JSON is forwarded.
//
// Output: raw HTTP response body from the webhook endpoint
type NotifyNode struct {
	client *http.Client
}

func NewNotifyNode() *NotifyNode {
	return &NotifyNode{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (n *NotifyNode) Execute(_ context.Context, config map[string]interface{}, input string) (string, error) {
	urlTmpl, _ := config["url"].(string)
	method, _ := config["method"].(string)
	if method == "" {
		method = "POST"
	}

	// Parse input as template data
	var inMap map[string]interface{}
	_ = json.Unmarshal([]byte(input), &inMap)
	if inMap == nil {
		inMap = map[string]interface{}{}
	}

	// Render URL template
	renderedURL, err := renderTemplate("notify_url", urlTmpl, inMap)
	if err != nil {
		return "", fmt.Errorf("notify: url template error: %w", err)
	}

	// Render body template (or fall through to raw input)
	requestBody := input
	if bodyTmpl, ok := config["body"].(string); ok && bodyTmpl != "" {
		requestBody, err = renderTemplate("notify_body", bodyTmpl, inMap)
		if err != nil {
			return "", fmt.Errorf("notify: body template error: %w", err)
		}
	}

	req, err := http.NewRequest(method, renderedURL, bytes.NewBufferString(requestBody))
	if err != nil {
		return "", fmt.Errorf("notify: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Optional custom headers
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("notify: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("notify: webhook returned HTTP %d", resp.StatusCode)
	}

	// Pass input downstream (notify is a side-effect node, not a data transformer)
	return input, nil
}

func (n *NotifyNode) Validate(config map[string]interface{}) error {
	if _, ok := config["url"].(string); !ok {
		return fmt.Errorf("notify: missing 'url' in config")
	}
	return nil
}

// renderTemplate is a shared helper for Go template rendering.
func renderTemplate(name, tmplStr string, data interface{}) (string, error) {
	t, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
