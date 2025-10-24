package prompts

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/jake/gocode/internal/config"
)

// PromptManager handles template-based prompt rendering
type PromptManager struct {
	templates *template.Template
}

// NewPromptManager creates a new prompt manager with embedded templates
func NewPromptManager() (*PromptManager, error) {
	// Parse all embedded templates
	tmpl, err := template.New("prompts").Funcs(templateFuncs()).ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt templates: %w", err)
	}

	return &PromptManager{
		templates: tmpl,
	}, nil
}

// SystemPromptData contains data for rendering the system prompt
type SystemPromptData struct {
	ContextWindow  int
	ModelName      string
	EnabledTools   []ToolInfo
	Features       FeatureFlags
	ProjectContext *ProjectContext
}

// ProjectContext contains project-specific information for the prompt
type ProjectContext struct {
	ProjectName      string
	PrimaryLanguages string
	TotalFiles       int
	CodeFiles        int
	TotalLines       int
	Frameworks       string
	GitBranch        string
	TechStack        string
	Structure        string
}

// ToolInfo describes a tool for the prompt
type ToolInfo struct {
	Name        string
	Description string
	Category    string
}

// FeatureFlags indicates which advanced features are enabled
type FeatureFlags struct {
	LSP         bool
	Retrieval   bool
	Checkpoint  bool
	Memory      bool
	Telemetry   bool
	Embeddings  bool
}

// RenderSystem renders the main system prompt
func (pm *PromptManager) RenderSystem(cfg *config.Config, tools []ToolInfo) (string, error) {
	return pm.RenderSystemWithProject(cfg, tools, nil)
}

// RenderSystemWithProject renders the system prompt with optional project context
func (pm *PromptManager) RenderSystemWithProject(cfg *config.Config, tools []ToolInfo, projectContext *ProjectContext) (string, error) {
	data := SystemPromptData{
		ContextWindow:  cfg.LLM.ContextWindow,
		ModelName:      cfg.LLM.Model,
		EnabledTools:   tools,
		ProjectContext: projectContext,
		Features: FeatureFlags{
			LSP:        cfg.LSP.Enabled,
			Retrieval:  cfg.Retrieval.Enabled,
			Checkpoint: cfg.Checkpoint.Enabled,
			Memory:     cfg.Memory.Enabled,
			Telemetry:  cfg.Telemetry.Enabled,
			Embeddings: cfg.Embeddings.Enabled,
		},
	}

	// Use enhanced template if project context is provided
	templateName := "system.tmpl"
	if projectContext != nil {
		templateName = "system_with_project.tmpl"
	}

	var buf bytes.Buffer
	if err := pm.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to render system prompt: %w", err)
	}

	return buf.String(), nil
}

// ContextInjectionData contains data for rendering context injection messages
type ContextInjectionData struct {
	Contexts []string
	Query    string
}

// RenderContextInjection renders a context injection message
func (pm *PromptManager) RenderContextInjection(contexts []string, query string) (string, error) {
	data := ContextInjectionData{
		Contexts: contexts,
		Query:    query,
	}

	var buf bytes.Buffer
	if err := pm.templates.ExecuteTemplate(&buf, "context_injection.tmpl", data); err != nil {
		return "", fmt.Errorf("failed to render context injection: %w", err)
	}

	return buf.String(), nil
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"formatNumber": func(n int) string {
			if n >= 1000 {
				return fmt.Sprintf("%d,%03d", n/1000, n%1000)
			}
			return fmt.Sprintf("%d", n)
		},
	}
}
