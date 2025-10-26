package initialization

import (
	"os/exec"
)

// FeatureDetector generates recommendations based on project analysis
type FeatureDetector struct {
	analysis *ProjectAnalysis
	config   interface{} // Config interface to check enabled features
}

// Config interface for checking enabled features
type Config interface {
	IsMemoryEnabled() bool
	IsCheckpointEnabled() bool
	IsRetrievalEnabled() bool
	IsTelemetryEnabled() bool
	IsEvaluationEnabled() bool
	IsLSPEnabled() bool
}

// NewFeatureDetector creates a new feature detector
func NewFeatureDetector(analysis *ProjectAnalysis, cfg Config) *FeatureDetector {
	return &FeatureDetector{
		analysis: analysis,
		config:   cfg,
	}
}

// GenerateRecommendations creates a list of feature recommendations
func (fd *FeatureDetector) GenerateRecommendations() []Recommendation {
	var recommendations []Recommendation

	// LSP recommendations based on detected languages
	lspRecs := fd.generateLSPRecommendations()
	recommendations = append(recommendations, lspRecs...)

	// Retrieval/embeddings recommendations for larger codebases
	if fd.config != nil && !fd.config.(Config).IsRetrievalEnabled() {
		if fd.analysis.Statistics.CodeFiles > 100 {
			recommendations = append(recommendations, Recommendation{
				Type:        "retrieval",
				Title:       "Enable Semantic Search",
				Description: "Large codebase detected. Enable retrieval and embeddings for better context discovery.",
				Priority:    "high",
			})
		}
	}

	// Checkpoint recommendations for complex projects
	if fd.config != nil && !fd.config.(Config).IsCheckpointEnabled() {
		if len(fd.analysis.Frameworks) > 2 || fd.analysis.Statistics.TotalLines > 10000 {
			recommendations = append(recommendations, Recommendation{
				Type:        "checkpoint",
				Title:       "Enable Session Checkpointing",
				Description: "Complex project detected. Checkpointing helps resume long-running conversations.",
				Priority:    "medium",
			})
		}
	}

	// Memory recommendations
	if fd.config != nil && !fd.config.(Config).IsMemoryEnabled() {
		if fd.analysis.GitInfo != nil && fd.analysis.GitInfo.IsGitRepo {
			recommendations = append(recommendations, Recommendation{
				Type:        "memory",
				Title:       "Enable Long-term Memory",
				Description: "Track architectural decisions and patterns across sessions.",
				Priority:    "medium",
			})
		}
	}

	return recommendations
}

// generateLSPRecommendations creates LSP server recommendations
func (fd *FeatureDetector) generateLSPRecommendations() []Recommendation {
	var recommendations []Recommendation

	lspServers := map[string]struct {
		language    string
		command     string
		name        string
		installCmd  string
	}{
		"Go":         {"go", "gopls", "gopls", "go install golang.org/x/tools/gopls@latest"},
		"Python":     {"python", "pylsp", "Python Language Server", "pip install python-lsp-server"},
		"TypeScript": {"typescript", "typescript-language-server", "TypeScript Language Server", "npm install -g typescript-language-server"},
		"JavaScript": {"javascript", "typescript-language-server", "TypeScript Language Server", "npm install -g typescript-language-server"},
		"Rust":       {"rust", "rust-analyzer", "rust-analyzer", "rustup component add rust-analyzer"},
		"Java":       {"java", "jdtls", "Eclipse JDT Language Server", "See: https://github.com/eclipse/eclipse.jdt.ls"},
		"C":          {"c", "clangd", "clangd", "Install LLVM/Clang toolchain"},
		"C++":        {"cpp", "clangd", "clangd", "Install LLVM/Clang toolchain"},
		"C#":         {"csharp", "omnisharp", "OmniSharp", "See: https://github.com/OmniSharp/omnisharp-roslyn"},
	}

	// Check each detected language
	for _, lang := range fd.analysis.Languages {
		if server, exists := lspServers[lang.Name]; exists {
			installed := fd.isCommandAvailable(server.command)

			// Don't recommend if LSP is already enabled and server is installed
			if fd.config != nil && fd.config.(Config).IsLSPEnabled() && installed {
				continue // Skip this recommendation - already configured
			}

			var desc string
			var action string

			if installed {
				desc = server.name + " is installed and ready to use. Enable LSP in config for code navigation."
			} else {
				desc = server.name + " not found. Install it for advanced code navigation and symbol search."
				action = server.installCmd
			}

			priority := "medium"
			if lang.Primary {
				priority = "high"
			}

			recommendations = append(recommendations, Recommendation{
				Type:        "lsp",
				Title:       "LSP for " + lang.Name,
				Description: desc,
				Priority:    priority,
				Installed:   installed,
				Action:      action,
			})
		}
	}

	return recommendations
}

// isCommandAvailable checks if a command is available in PATH
func (fd *FeatureDetector) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
