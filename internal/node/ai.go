package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AINode represents a node that calls AI APIs (OpenAI, Anthropic, Gemini)
// Config:
//   - "provider"      (string) : "openai" | "anthropic" | "gemini"
//   - "api_key"       (string) : API key for the provider
//   - "model"         (string) : Model name (e.g., "gpt-4", "claude-3-sonnet", "gemini-pro")
//   - "prompt"        (string) : Prompt template with {{.field}} syntax
//   - "system_prompt" (string) : Optional system message
//   - "temperature"   (float)  : Optional (0.0-2.0)
//   - "max_tokens"    (int)    : Optional max response tokens
//   - "result_key"    (string) : Key to store AI response (default: "ai_response")
type AINode struct{}

func (n *AINode) Execute(ctx context.Context, config map[string]interface{}, input string) (string, error) {
	provider, _ := config["provider"].(string)
	apiKey, _ := config["api_key"].(string)
	model, _ := config["model"].(string)
	promptTemplate, _ := config["prompt"].(string)
	systemPrompt, _ := config["system_prompt"].(string)
	resultKey, _ := config["result_key"].(string)

	if resultKey == "" {
		resultKey = "ai_response"
	}

	if provider == "" {
		return "", fmt.Errorf("ai_node: 'provider' is required")
	}
	if apiKey == "" {
		return "", fmt.Errorf("ai_node: 'api_key' is required")
	}
	if model == "" {
		return "", fmt.Errorf("ai_node: 'model' is required")
	}
	if promptTemplate == "" {
		return "", fmt.Errorf("ai_node: 'prompt' is required")
	}

	// Parse input JSON
	var inputMap map[string]interface{}
	if input != "" {
		_ = json.Unmarshal([]byte(input), &inputMap)
	}
	if inputMap == nil {
		inputMap = make(map[string]interface{})
	}

	// Render prompt template
	prompt, err := renderTemplate(promptTemplate, inputMap)
	if err != nil {
		return "", fmt.Errorf("ai_node: failed to render prompt: %w", err)
	}

	// Get optional parameters
	temperature := 0.7
	if t, ok := config["temperature"].(float64); ok {
		temperature = t
	}
	maxTokens := 1000
	if m, ok := config["max_tokens"].(float64); ok {
		maxTokens = int(m)
	}

	fmt.Printf("\n🤖 [AI NODE] Provider: %s | Model: %s\n", provider, model)
	fmt.Printf("   Prompt: %s\n", truncateString(prompt, 150))

	var response string
	switch provider {
	case "openai":
		response, err = n.callOpenAI(ctx, apiKey, model, systemPrompt, prompt, temperature, maxTokens)
	case "anthropic":
		response, err = n.callAnthropic(ctx, apiKey, model, systemPrompt, prompt, temperature, maxTokens)
	case "gemini":
		response, err = n.callGemini(ctx, apiKey, model, prompt, temperature, maxTokens)
	default:
		return "", fmt.Errorf("ai_node: unsupported provider '%s'", provider)
	}

	if err != nil {
		return "", fmt.Errorf("ai_node: %w", err)
	}

	fmt.Printf("   Response: %s\n", truncateString(response, 150))

	// Merge response into output
	outputMap := make(map[string]interface{})
	for k, v := range inputMap {
		outputMap[k] = v
	}
	outputMap[resultKey] = response
	delete(outputMap, "branch") // Clean branch tag

	out, _ := json.Marshal(outputMap)
	return string(out), nil
}

func (n *AINode) Validate(config map[string]interface{}) error {
	provider, ok := config["provider"].(string)
	if !ok {
		return fmt.Errorf("ai_node: 'provider' must be a string")
	}

	validProviders := map[string]bool{"openai": true, "anthropic": true, "gemini": true}
	if !validProviders[provider] {
		return fmt.Errorf("ai_node: invalid provider '%s' (must be: openai, anthropic, gemini)", provider)
	}

	if _, ok := config["api_key"].(string); !ok {
		return fmt.Errorf("ai_node: 'api_key' must be a string")
	}
	if _, ok := config["model"].(string); !ok {
		return fmt.Errorf("ai_node: 'model' must be a string")
	}
	if _, ok := config["prompt"].(string); !ok {
		return fmt.Errorf("ai_node: 'prompt' must be a string")
	}

	return nil
}

// ──────────────────────────────────────────────────────────────
// OpenAI API (ChatGPT)
// ──────────────────────────────────────────────────────────────

type openAIRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIMessage     `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (n *AINode) callOpenAI(ctx context.Context, apiKey, model, systemPrompt, prompt string, temp float64, maxTokens int) (string, error) {
	messages := []openAIMessage{}

	if systemPrompt != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, openAIMessage{Role: "user", Content: prompt})

	reqBody := openAIRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices in response")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// ──────────────────────────────────────────────────────────────
// Anthropic Claude API
// ──────────────────────────────────────────────────────────────

type anthropicRequest struct {
	Model       string              `json:"model"`
	Messages    []anthropicMessage  `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature,omitempty"`
	System      string              `json:"system,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (n *AINode) callAnthropic(ctx context.Context, apiKey, model, systemPrompt, prompt string, temp float64, maxTokens int) (string, error) {
	reqBody := anthropicRequest{
		Model:       model,
		Messages:    []anthropicMessage{{Role: "user", Content: prompt}},
		MaxTokens:   maxTokens,
		Temperature: temp,
		System:      systemPrompt,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anthropic api error (status %d): %s", resp.StatusCode, string(body))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return "", err
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("anthropic: no content in response")
	}

	return anthropicResp.Content[0].Text, nil
}

// ──────────────────────────────────────────────────────────────
// Google Gemini API
// ──────────────────────────────────────────────────────────────

type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig geminiGenerationConfig  `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (n *AINode) callGemini(ctx context.Context, apiKey, model, prompt string, temp float64, maxTokens int) (string, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature: temp,
			MaxTokens:   maxTokens,
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gemini api error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: no candidates in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}
