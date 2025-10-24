package initialization

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectAnalysis contains the results of analyzing a project
type ProjectAnalysis struct {
	ProjectName     string              `json:"project_name"`
	Languages       []LanguageInfo      `json:"languages"`
	Frameworks      []FrameworkInfo     `json:"frameworks"`
	Dependencies    []DependencyInfo    `json:"dependencies"`
	Structure       ProjectStructure    `json:"structure"`
	Statistics      CodeStatistics      `json:"statistics"`
	GitInfo         *GitInfo            `json:"git_info,omitempty"`
	Recommendations []Recommendation    `json:"recommendations"`
}

// LanguageInfo describes a detected programming language
type LanguageInfo struct {
	Name       string `json:"name"`
	FileCount  int    `json:"file_count"`
	Extensions []string `json:"extensions"`
	Primary    bool   `json:"primary"`
}

// FrameworkInfo describes a detected framework
type FrameworkInfo struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Version  string `json:"version,omitempty"`
}

// DependencyInfo describes project dependencies
type DependencyInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // go.mod, package.json, requirements.txt, etc.
	Count   int    `json:"count"`
}

// ProjectStructure describes the project's directory structure
type ProjectStructure struct {
	HasSrcDir    bool     `json:"has_src_dir"`
	HasTestsDir  bool     `json:"has_tests_dir"`
	HasDocsDir   bool     `json:"has_docs_dir"`
	ConfigFiles  []string `json:"config_files"`
	EntryPoints  []string `json:"entry_points"`
}

// CodeStatistics contains code metrics
type CodeStatistics struct {
	TotalFiles       int `json:"total_files"`
	TotalDirectories int `json:"total_directories"`
	TotalLines       int `json:"total_lines"`
	CodeFiles        int `json:"code_files"`
}

// GitInfo contains git repository information
type GitInfo struct {
	IsGitRepo     bool   `json:"is_git_repo"`
	CurrentBranch string `json:"current_branch,omitempty"`
	HasRemote     bool   `json:"has_remote"`
}

// Recommendation represents a suggested action or feature
type Recommendation struct {
	Type        string `json:"type"` // "lsp", "embedding", "tool", etc.
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"` // "high", "medium", "low"
	Installed   bool   `json:"installed,omitempty"` // For LSP servers
}

// Analyzer performs project analysis
type Analyzer struct {
	workingDir string
	detector   *Detector
}

// NewAnalyzer creates a new project analyzer
func NewAnalyzer(workingDir string, detector *Detector) *Analyzer {
	return &Analyzer{
		workingDir: workingDir,
		detector:   detector,
	}
}

// Analyze performs a complete project analysis
func (a *Analyzer) Analyze() (*ProjectAnalysis, error) {
	analysis := &ProjectAnalysis{
		ProjectName: filepath.Base(a.workingDir),
	}

	// Analyze files and detect languages
	fileInfo, err := a.scanFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to scan files: %w", err)
	}

	analysis.Languages = a.detectLanguages(fileInfo)
	analysis.Statistics = a.calculateStatistics(fileInfo)
	analysis.Structure = a.analyzeStructure(fileInfo)
	analysis.Frameworks = a.detectFrameworks(fileInfo)
	analysis.Dependencies = a.detectDependencies(fileInfo)
	analysis.GitInfo = a.analyzeGit()

	// Save analysis to cache
	if err := a.saveAnalysis(analysis); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to cache analysis: %v\n", err)
	}

	return analysis, nil
}

// LoadCachedAnalysis loads previously cached analysis if available
func (a *Analyzer) LoadCachedAnalysis() (*ProjectAnalysis, error) {
	analysisPath := a.detector.GetAnalysisPath()

	data, err := os.ReadFile(analysisPath)
	if err != nil {
		return nil, err
	}

	var analysis ProjectAnalysis
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, err
	}

	return &analysis, nil
}

// fileInfo represents information about scanned files
type fileInfo struct {
	path      string
	ext       string
	isDir     bool
	size      int64
	lines     int
}

// scanFiles recursively scans the project directory
func (a *Analyzer) scanFiles() ([]fileInfo, error) {
	var files []fileInfo

	// Directories to skip
	skipDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		".gocode":      true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
		"dist":         true,
		"build":        true,
		"target":       true,
		".next":        true,
		".nuxt":        true,
	}

	err := filepath.Walk(a.workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue scanning
		}

		// Skip hidden files and directories (except .go, .gitignore, etc.)
		name := filepath.Base(path)
		if strings.HasPrefix(name, ".") && info.IsDir() && name != "." {
			if skipDirs[name] {
				return filepath.SkipDir
			}
		}

		// Skip known large directories
		if info.IsDir() && skipDirs[name] {
			return filepath.SkipDir
		}

		// Get relative path
		relPath, _ := filepath.Rel(a.workingDir, path)
		if relPath == "." {
			return nil
		}

		ext := filepath.Ext(path)
		fi := fileInfo{
			path:  relPath,
			ext:   ext,
			isDir: info.IsDir(),
			size:  info.Size(),
		}

		// Count lines for code files
		if !info.IsDir() && a.isCodeFile(ext) {
			fi.lines = a.countLines(path)
		}

		files = append(files, fi)
		return nil
	})

	return files, err
}

// detectLanguages identifies programming languages in the project
func (a *Analyzer) detectLanguages(files []fileInfo) []LanguageInfo {
	langMap := map[string]*LanguageInfo{
		"go": {
			Name:       "Go",
			Extensions: []string{".go"},
		},
		"python": {
			Name:       "Python",
			Extensions: []string{".py"},
		},
		"javascript": {
			Name:       "JavaScript",
			Extensions: []string{".js", ".mjs", ".cjs"},
		},
		"typescript": {
			Name:       "TypeScript",
			Extensions: []string{".ts", ".tsx"},
		},
		"rust": {
			Name:       "Rust",
			Extensions: []string{".rs"},
		},
		"java": {
			Name:       "Java",
			Extensions: []string{".java"},
		},
		"c": {
			Name:       "C",
			Extensions: []string{".c", ".h"},
		},
		"cpp": {
			Name:       "C++",
			Extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".h++"},
		},
		"csharp": {
			Name:       "C#",
			Extensions: []string{".cs"},
		},
		"ruby": {
			Name:       "Ruby",
			Extensions: []string{".rb"},
		},
		"php": {
			Name:       "PHP",
			Extensions: []string{".php"},
		},
	}

	// Count files per language
	counts := make(map[string]int)
	for _, file := range files {
		if file.isDir {
			continue
		}
		for key, info := range langMap {
			for _, ext := range info.Extensions {
				if file.ext == ext {
					counts[key]++
					break
				}
			}
		}
	}

	// Build result
	var languages []LanguageInfo
	maxCount := 0
	primaryLang := ""

	for key, count := range counts {
		if count > 0 {
			info := *langMap[key]
			info.FileCount = count
			languages = append(languages, info)

			if count > maxCount {
				maxCount = count
				primaryLang = key
			}
		}
	}

	// Mark primary language
	for i := range languages {
		if langMap[primaryLang].Name == languages[i].Name {
			languages[i].Primary = true
		}
	}

	return languages
}

// detectFrameworks identifies frameworks based on config files and structure
func (a *Analyzer) detectFrameworks(files []fileInfo) []FrameworkInfo {
	var frameworks []FrameworkInfo

	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file.path] = true
	}

	// Go frameworks
	if fileSet["go.mod"] {
		content, _ := os.ReadFile(filepath.Join(a.workingDir, "go.mod"))
		contentStr := string(content)

		if strings.Contains(contentStr, "gin-gonic/gin") {
			frameworks = append(frameworks, FrameworkInfo{Name: "Gin", Language: "Go"})
		}
		if strings.Contains(contentStr, "gorilla/mux") {
			frameworks = append(frameworks, FrameworkInfo{Name: "Gorilla Mux", Language: "Go"})
		}
		if strings.Contains(contentStr, "labstack/echo") {
			frameworks = append(frameworks, FrameworkInfo{Name: "Echo", Language: "Go"})
		}
	}

	// JavaScript/TypeScript frameworks
	if fileSet["package.json"] {
		content, _ := os.ReadFile(filepath.Join(a.workingDir, "package.json"))
		contentStr := string(content)

		if strings.Contains(contentStr, "\"react\"") {
			frameworks = append(frameworks, FrameworkInfo{Name: "React", Language: "TypeScript/JavaScript"})
		}
		if strings.Contains(contentStr, "\"vue\"") {
			frameworks = append(frameworks, FrameworkInfo{Name: "Vue", Language: "TypeScript/JavaScript"})
		}
		if strings.Contains(contentStr, "\"next\"") {
			frameworks = append(frameworks, FrameworkInfo{Name: "Next.js", Language: "TypeScript/JavaScript"})
		}
		if strings.Contains(contentStr, "\"express\"") {
			frameworks = append(frameworks, FrameworkInfo{Name: "Express", Language: "TypeScript/JavaScript"})
		}
		if strings.Contains(contentStr, "\"@nestjs\"") {
			frameworks = append(frameworks, FrameworkInfo{Name: "NestJS", Language: "TypeScript"})
		}
	}

	// Python frameworks
	if fileSet["requirements.txt"] || fileSet["pyproject.toml"] {
		reqFile := filepath.Join(a.workingDir, "requirements.txt")
		if !fileSet["requirements.txt"] {
			reqFile = filepath.Join(a.workingDir, "pyproject.toml")
		}

		content, _ := os.ReadFile(reqFile)
		contentStr := string(content)

		if strings.Contains(contentStr, "django") {
			frameworks = append(frameworks, FrameworkInfo{Name: "Django", Language: "Python"})
		}
		if strings.Contains(contentStr, "flask") {
			frameworks = append(frameworks, FrameworkInfo{Name: "Flask", Language: "Python"})
		}
		if strings.Contains(contentStr, "fastapi") {
			frameworks = append(frameworks, FrameworkInfo{Name: "FastAPI", Language: "Python"})
		}
	}

	return frameworks
}

// detectDependencies analyzes dependency files
func (a *Analyzer) detectDependencies(files []fileInfo) []DependencyInfo {
	var deps []DependencyInfo

	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file.path] = true
	}

	if fileSet["go.mod"] {
		count := a.countGoModDependencies()
		deps = append(deps, DependencyInfo{
			Name:  "Go Modules",
			Type:  "go.mod",
			Count: count,
		})
	}

	if fileSet["package.json"] {
		count := a.countPackageJSONDependencies()
		deps = append(deps, DependencyInfo{
			Name:  "npm/yarn",
			Type:  "package.json",
			Count: count,
		})
	}

	if fileSet["requirements.txt"] {
		count := a.countRequirementsTxt()
		deps = append(deps, DependencyInfo{
			Name:  "pip",
			Type:  "requirements.txt",
			Count: count,
		})
	}

	if fileSet["Cargo.toml"] {
		deps = append(deps, DependencyInfo{
			Name:  "Cargo",
			Type:  "Cargo.toml",
			Count: 0, // TODO: parse Cargo.toml
		})
	}

	return deps
}

// analyzeStructure analyzes project directory structure
func (a *Analyzer) analyzeStructure(files []fileInfo) ProjectStructure {
	structure := ProjectStructure{
		ConfigFiles: []string{},
		EntryPoints: []string{},
	}

	for _, file := range files {
		lowerPath := strings.ToLower(file.path)
		name := filepath.Base(file.path)

		// Check for common directories
		if file.isDir {
			if strings.Contains(lowerPath, "src") || strings.HasPrefix(lowerPath, "src/") {
				structure.HasSrcDir = true
			}
			if strings.Contains(lowerPath, "test") || strings.HasPrefix(lowerPath, "test") {
				structure.HasTestsDir = true
			}
			if strings.Contains(lowerPath, "doc") {
				structure.HasDocsDir = true
			}
		}

		// Detect config files
		configNames := []string{
			"config.yaml", "config.yml", "config.json", "config.toml",
			".env", ".env.example",
			"tsconfig.json", "webpack.config.js", "vite.config.ts",
			"go.mod", "package.json", "Cargo.toml", "pyproject.toml",
		}
		for _, cfg := range configNames {
			if name == cfg {
				structure.ConfigFiles = append(structure.ConfigFiles, file.path)
				break
			}
		}

		// Detect entry points
		if name == "main.go" || name == "main.py" || name == "index.js" ||
		   name == "index.ts" || name == "app.py" || name == "server.js" {
			structure.EntryPoints = append(structure.EntryPoints, file.path)
		}
	}

	return structure
}

// calculateStatistics computes code statistics
func (a *Analyzer) calculateStatistics(files []fileInfo) CodeStatistics {
	stats := CodeStatistics{}

	for _, file := range files {
		if file.isDir {
			stats.TotalDirectories++
		} else {
			stats.TotalFiles++
			if a.isCodeFile(file.ext) {
				stats.CodeFiles++
				stats.TotalLines += file.lines
			}
		}
	}

	return stats
}

// analyzeGit checks git repository information
func (a *Analyzer) analyzeGit() *GitInfo {
	gitDir := filepath.Join(a.workingDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return &GitInfo{IsGitRepo: false}
	}

	info := &GitInfo{IsGitRepo: true}

	// Try to read current branch
	headFile := filepath.Join(gitDir, "HEAD")
	if content, err := os.ReadFile(headFile); err == nil {
		head := strings.TrimSpace(string(content))
		if strings.HasPrefix(head, "ref: refs/heads/") {
			info.CurrentBranch = strings.TrimPrefix(head, "ref: refs/heads/")
		}
	}

	// Check for remote
	configFile := filepath.Join(gitDir, "config")
	if content, err := os.ReadFile(configFile); err == nil {
		info.HasRemote = strings.Contains(string(content), "[remote")
	}

	return info
}

// Helper functions

func (a *Analyzer) isCodeFile(ext string) bool {
	codeExts := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".rs": true, ".java": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".cs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true, ".scala": true,
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".html": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
		".sql": true, ".graphql": true, ".proto": true,
	}
	return codeExts[ext]
}

func (a *Analyzer) countLines(path string) int {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(content), "\n") + 1
}

func (a *Analyzer) countGoModDependencies() int {
	content, err := os.ReadFile(filepath.Join(a.workingDir, "go.mod"))
	if err != nil {
		return 0
	}
	return strings.Count(string(content), "\n\t") // Rough count of require lines
}

func (a *Analyzer) countPackageJSONDependencies() int {
	content, err := os.ReadFile(filepath.Join(a.workingDir, "package.json"))
	if err != nil {
		return 0
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return 0
	}

	count := 0
	if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
		count += len(deps)
	}
	if devDeps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		count += len(devDeps)
	}

	return count
}

func (a *Analyzer) countRequirementsTxt() int {
	content, err := os.ReadFile(filepath.Join(a.workingDir, "requirements.txt"))
	if err != nil {
		return 0
	}

	lines := strings.Split(string(content), "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}

	return count
}

func (a *Analyzer) saveAnalysis(analysis *ProjectAnalysis) error {
	if err := a.detector.EnsureStateDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return err
	}

	analysisPath := a.detector.GetAnalysisPath()
	return os.WriteFile(analysisPath, data, 0644)
}
