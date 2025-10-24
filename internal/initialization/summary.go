package initialization

import (
	"fmt"
	"strings"

	"github.com/jake/gocode/internal/theme"
)

// DisplaySummary shows a formatted summary of the project analysis
func DisplaySummary(analysis *ProjectAnalysis, recommendations []Recommendation) {
	fmt.Println()
	fmt.Println(theme.HeaderBold("🔍 Project Analysis Complete"))
	fmt.Println()

	// Project Overview
	displayProjectOverview(analysis)
	fmt.Println()

	// Languages
	if len(analysis.Languages) > 0 {
		displayLanguages(analysis.Languages)
		fmt.Println()
	}

	// Frameworks
	if len(analysis.Frameworks) > 0 {
		displayFrameworks(analysis.Frameworks)
		fmt.Println()
	}

	// Dependencies
	if len(analysis.Dependencies) > 0 {
		displayDependencies(analysis.Dependencies)
		fmt.Println()
	}

	// Recommendations
	if len(recommendations) > 0 {
		displayRecommendations(recommendations)
		fmt.Println()
	}

	// Help hint
	fmt.Println(theme.Dim("Type 'help' to see available commands, or just start asking questions!"))
	fmt.Println()
}

func displayProjectOverview(analysis *ProjectAnalysis) {
	// Project name and type
	fmt.Printf("%s %s", theme.HeaderBold("📊 Project:"), theme.Agent(analysis.ProjectName))

	// Primary languages
	primaryLangs := []string{}
	for _, lang := range analysis.Languages {
		if lang.Primary {
			primaryLangs = append(primaryLangs, lang.Name)
		}
	}
	if len(primaryLangs) > 0 {
		fmt.Printf(" (%s)", theme.Success(strings.Join(primaryLangs, " + ")))
	}
	fmt.Println()

	// Statistics
	fmt.Printf("   %s %s files", theme.Dim("-"), theme.Agent(fmt.Sprintf("%d", analysis.Statistics.TotalFiles)))

	if analysis.Statistics.CodeFiles > 0 {
		fmt.Printf(" (%s code files)", theme.Success(fmt.Sprintf("%d", analysis.Statistics.CodeFiles)))
	}
	fmt.Println()

	if analysis.Statistics.TotalLines > 0 {
		fmt.Printf("   %s %s lines of code\n", theme.Dim("-"), theme.Agent(formatNumber(analysis.Statistics.TotalLines)))
	}

	// Git info
	if analysis.GitInfo != nil && analysis.GitInfo.IsGitRepo {
		if analysis.GitInfo.CurrentBranch != "" {
			fmt.Printf("   %s Git branch: %s\n", theme.Dim("-"), theme.Agent(analysis.GitInfo.CurrentBranch))
		}
	}
}

func displayLanguages(languages []LanguageInfo) {
	fmt.Println(theme.HeaderBold("💻 Languages Detected:"))
	for _, lang := range languages {
		marker := " "
		if lang.Primary {
			marker = "★"
		}
		fmt.Printf("   %s %s (%s files)\n",
			theme.Success(marker),
			theme.Agent(lang.Name),
			theme.Dim(fmt.Sprintf("%d", lang.FileCount)))
	}
}

func displayFrameworks(frameworks []FrameworkInfo) {
	fmt.Println(theme.HeaderBold("🛠️  Frameworks Detected:"))
	for _, fw := range frameworks {
		fmt.Printf("   %s %s (%s)\n",
			theme.Success("•"),
			theme.Agent(fw.Name),
			theme.Dim(fw.Language))
	}
}

func displayDependencies(dependencies []DependencyInfo) {
	fmt.Println(theme.HeaderBold("📦 Dependencies:"))
	for _, dep := range dependencies {
		count := ""
		if dep.Count > 0 {
			count = fmt.Sprintf(" - %d packages", dep.Count)
		}
		fmt.Printf("   %s %s (%s)%s\n",
			theme.Success("•"),
			theme.Agent(dep.Name),
			theme.Dim(dep.Type),
			theme.Dim(count))
	}
}

func displayRecommendations(recommendations []Recommendation) {
	fmt.Println(theme.HeaderBold("💡 Recommended Enhancements:"))

	// Group by priority
	highPriority := []Recommendation{}
	mediumPriority := []Recommendation{}
	lowPriority := []Recommendation{}

	for _, rec := range recommendations {
		switch rec.Priority {
		case "high":
			highPriority = append(highPriority, rec)
		case "medium":
			mediumPriority = append(mediumPriority, rec)
		default:
			lowPriority = append(lowPriority, rec)
		}
	}

	// Display high priority first
	displayRecommendationGroup(highPriority)
	displayRecommendationGroup(mediumPriority)
	displayRecommendationGroup(lowPriority)
}

func displayRecommendationGroup(recommendations []Recommendation) {
	for _, rec := range recommendations {
		icon := getRecommendationIcon(rec.Type, rec.Installed)
		fmt.Printf("   %s %s\n", icon, theme.Agent(rec.Title))
		fmt.Printf("      %s\n", theme.Dim(rec.Description))
	}
}

func getRecommendationIcon(recType string, installed bool) string {
	if recType == "lsp" {
		if installed {
			return theme.Success("✅")
		}
		return theme.Warning("⚠️")
	}
	return theme.Success("💡")
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}

// DisplayInitPrompt shows the initialization prompt to the user
func DisplayInitPrompt(projectName string) bool {
	fmt.Println()
	fmt.Println(theme.HeaderBold("🚀 Welcome to GoCode!"))
	fmt.Println()
	fmt.Printf("%s This appears to be the first time running gocode in %s.\n",
		theme.Agent("👋"),
		theme.Success(projectName))
	fmt.Println()
	fmt.Println(theme.Dim("Would you like to initialize this project?"))
	fmt.Println(theme.Dim("This will:"))
	fmt.Println(theme.Dim("  • Analyze project structure and detect languages/frameworks"))
	fmt.Println(theme.Dim("  • Recommend appropriate tools and features"))
	fmt.Println(theme.Dim("  • Create .gocode/ directory for caching (add to .gitignore)"))
	fmt.Println()
	fmt.Printf("%s ", theme.Agent("Initialize project? [Y/n]:"))

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))

	// Default to yes if empty or "y"
	return response == "" || response == "y" || response == "yes"
}

// DisplaySkipMessage shows a message when initialization is skipped
func DisplaySkipMessage() {
	fmt.Println()
	fmt.Println(theme.Dim("Skipping initialization. You can run this later by deleting .gocode/ directory."))
	fmt.Println()
}

// DisplayInitProgress shows progress during initialization
func DisplayInitProgress(message string) {
	fmt.Printf("%s %s\n", theme.Agent("⏳"), theme.Dim(message))
}

// DisplayInitError shows an error during initialization
func DisplayInitError(err error) {
	fmt.Println()
	fmt.Printf("%s Failed to initialize: %v\n", theme.Error("❌"), err)
	fmt.Println(theme.Dim("Continuing without initialization..."))
	fmt.Println()
}
