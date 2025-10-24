package initialization

import (
	"os/exec"
)

// FeatureDetector generates recommendations based on project analysis
type FeatureDetector struct {
	analysis *ProjectAnalysis
}

// NewFeatureDetector creates a new feature detector
func NewFeatureDetector(analysis *ProjectAnalysis) *FeatureDetector {
	return &FeatureDetector{
		analysis: analysis,
	}
}

// GenerateRecommendations creates a list of feature recommendations
func (fd *FeatureDetector) GenerateRecommendations() []Recommendation {
	var recommendations []Recommendation

	// LSP recommendations based on detected languages
	lspRecs := fd.generateLSPRecommendations()
	recommendations = append(recommendations, lspRecs...)

	// Retrieval/embeddings recommendations for larger codebases
	if fd.analysis.Statistics.CodeFiles > 100 {
		recommendations = append(recommendations, Recommendation{
			Type:        "retrieval",
			Title:       "Enable Semantic Search",
			Description: "Large codebase detected. Enable retrieval and embeddings for better context discovery.",
			Priority:    "high",
		})
	}

	// Checkpoint recommendations for complex projects
	if len(fd.analysis.Frameworks) > 2 || fd.analysis.Statistics.TotalLines > 10000 {
		recommendations = append(recommendations, Recommendation{
			Type:        "checkpoint",
			Title:       "Enable Session Checkpointing",
			Description: "Complex project detected. Checkpointing helps resume long-running conversations.",
			Priority:    "medium",
		})
	}

	// Memory recommendations
	if fd.analysis.GitInfo != nil && fd.analysis.GitInfo.IsGitRepo {
		recommendations = append(recommendations, Recommendation{
			Type:        "memory",
			Title:       "Enable Long-term Memory",
			Description: "Track architectural decisions and patterns across sessions.",
			Priority:    "medium",
		})
	}

	return recommendations
}

// generateLSPRecommendations creates LSP server recommendations
func (fd *FeatureDetector) generateLSPRecommendations() []Recommendation {
	var recommendations []Recommendation

	lspServers := map[string]struct {
		language string
		command  string
		name     string
	}{
		"Go":                  {"go", "gopls", "gopls"},
		"Python":              {"python", "pylsp", "Python Language Server"},
		"TypeScript":          {"typescript", "typescript-language-server", "TypeScript Language Server"},
		"JavaScript":          {"javascript", "typescript-language-server", "TypeScript Language Server"},
		"Rust":                {"rust", "rust-analyzer", "rust-analyzer"},
		"Java":                {"java", "jdtls", "Eclipse JDT Language Server"},
		"C":                   {"c", "clangd", "clangd"},
		"C++":                 {"cpp", "clangd", "clangd"},
		"C#":                  {"csharp", "omnisharp", "OmniSharp"},
	}

	// Check each detected language
	for _, lang := range fd.analysis.Languages {
		if server, exists := lspServers[lang.Name]; exists {
			installed := fd.isCommandAvailable(server.command)

			var desc string
			if installed {
				desc = server.name + " is installed and ready to use. Enable LSP in config for code navigation."
			} else {
				desc = server.name + " not found. Install it for advanced code navigation and symbol search."
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
