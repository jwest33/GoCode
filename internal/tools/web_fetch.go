package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type WebFetchTool struct {
	client *http.Client
}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetches content from a specified URL and returns it. Converts HTML to markdown-like format."
}

func (t *WebFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "The URL to fetch content from",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Optional prompt describing what information to extract",
			},
		},
		"required": []string{"url"},
	}
}

type WebFetchArgs struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt,omitempty"`
}

func (t *WebFetchTool) Execute(ctx context.Context, args string) (string, error) {
	var fetchArgs WebFetchArgs
	if err := UnmarshalArgs(args, &fetchArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fetchArgs.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Coder-Agent/1.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)

	// Basic HTML to markdown conversion
	content = t.htmlToMarkdown(content)

	// Truncate if too large
	if len(content) > 50000 {
		content = content[:50000] + "\n\n... (content truncated)"
	}

	if fetchArgs.Prompt != "" {
		return fmt.Sprintf("Content from %s:\n\n%s\n\nPrompt: %s", fetchArgs.URL, content, fetchArgs.Prompt), nil
	}

	return fmt.Sprintf("Content from %s:\n\n%s", fetchArgs.URL, content), nil
}

func (t *WebFetchTool) htmlToMarkdown(html string) string {
	// Very basic HTML stripping - in production you'd use a proper library
	content := html

	// Remove script and style tags
	content = removeTag(content, "script")
	content = removeTag(content, "style")

	// Convert common tags
	content = strings.ReplaceAll(content, "<br>", "\n")
	content = strings.ReplaceAll(content, "<br/>", "\n")
	content = strings.ReplaceAll(content, "<br />", "\n")
	content = strings.ReplaceAll(content, "<p>", "\n\n")
	content = strings.ReplaceAll(content, "</p>", "\n")
	content = strings.ReplaceAll(content, "<h1>", "\n# ")
	content = strings.ReplaceAll(content, "<h2>", "\n## ")
	content = strings.ReplaceAll(content, "<h3>", "\n### ")

	// Remove remaining HTML tags
	for strings.Contains(content, "<") && strings.Contains(content, ">") {
		start := strings.Index(content, "<")
		end := strings.Index(content[start:], ">")
		if end == -1 {
			break
		}
		content = content[:start] + content[start+end+1:]
	}

	// Clean up whitespace
	lines := strings.Split(content, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

func removeTag(content, tag string) string {
	for {
		start := strings.Index(content, "<"+tag)
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], "</"+tag+">")
		if end == -1 {
			break
		}
		end += start + len("</"+tag+">")
		content = content[:start] + content[end:]
	}
	return content
}
