package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const defaultManifest = `name = "%s"
version = "0.1.0"
`

const defaultMain = `fn main() -> MeasuredResult {
    let q = qreg(2);
    h(q[0]);
    cx(q[0], q[1]);
    return measure(q);
}
`

const readmeTemplate = `# %s

A Qasmith quantum computing project.

## Getting Started

Build the project:

` + "```bash" + `
qasmith compile
` + "```" + `

Run the quantum simulation:

` + "```bash" + `
qasmith run
` + "```" + `

## Project Structure

- ` + "`src/main.smith`" + ` - Entry point
- ` + "`qasmith.toml`" + ` - Project manifest
`

func runInit(args []string, out, stderr io.Writer) int {
	if len(args) == 0 || len(args) > 2 {
		fmt.Fprintln(stderr, "usage: qasmith init <project-name> [--template <template>]")
		return 2
	}
	
	projectName := args[0]
	template := "default"
	
	// Parse optional flags
	for i := 1; i < len(args); i++ {
		if args[i] == "--template" && i+1 < len(args) {
			template = args[i+1]
			i++
		} else {
			fmt.Fprintf(stderr, "unknown option: %s\n", args[i])
			return 2
		}
	}
	
	// Validate project name
	if !isValidProjectName(projectName) {
		fmt.Fprintln(stderr, "invalid project name: must start with a letter and contain only letters, numbers, hyphens, and underscores")
		return 1
	}
	
	// Create project directory
	projectDir := filepath.Join(".", projectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		fmt.Fprintf(stderr, "failed to create project directory: %v\n", err)
		return 1
	}
	
	// Create src directory
	srcDir := filepath.Join(projectDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		fmt.Fprintf(stderr, "failed to create src directory: %v\n", err)
		return 1
	}
	
	// Create manifest file
	manifestPath := filepath.Join(projectDir, "qasmith.toml")
	if _, err := os.Stat(manifestPath); err == nil {
		fmt.Fprintf(stderr, "manifest already exists: %s\n", manifestPath)
		return 1
	}
	
	manifestContent := fmt.Sprintf(defaultManifest, projectName)
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		fmt.Fprintf(stderr, "failed to write manifest: %v\n", err)
		return 1
	}
	
	// Create main.smith based on template
	mainPath := filepath.Join(srcDir, "main.smith")
	mainContent := getTemplateContent(template)
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		fmt.Fprintf(stderr, "failed to write main.smith: %v\n", err)
		return 1
	}
	
	// Create README.md
	readmePath := filepath.Join(projectDir, "README.md")
	readmeContent := fmt.Sprintf(readmeTemplate, projectName)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		fmt.Fprintf(stderr, "failed to write README.md: %v\n", err)
		return 1
	}
	
	// Create .gitignore
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	gitignoreContent := `# Qasmith build artifacts
*.qasm
qasmith.lock

# Editor directories
.vscode/
.idea/

# OS files
.DS_Store
Thumbs.db
`
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		fmt.Fprintf(stderr, "failed to write .gitignore: %v\n", err)
		return 1
	}
	
	fmt.Fprintf(out, "Created Qasmith project %q in %s\n", projectName, projectDir)
	fmt.Fprintf(out, "\nNext steps:\n")
	fmt.Fprintf(out, "  cd %s\n", projectName)
	fmt.Fprintf(out, "  qasmith run\n")
	
	return 0
}

func isValidProjectName(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Must start with a letter
	if !isLetter(name[0]) {
		return false
	}
	// Can contain letters, numbers, hyphens, and underscores
	for _, c := range name {
		if !isLetter(byte(c)) && !isDigit(byte(c)) && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func getTemplateContent(template string) string {
	switch strings.ToLower(template) {
	case "bell":
		return `// Bell state preparation
fn main() -> MeasuredResult {
    let q = qreg(2);
    h(q[0]);
    cx(q[0], q[1]);
    return measure(q);
}
`
	case "ghz":
		return `// GHZ state preparation
fn main() -> MeasuredResult {
    let q = qreg(3);
    h(q[0]);
    cx(q[0], q[1]);
    cx(q[0], q[2]);
    return measure(q);
}
`
	case "superposition":
		return `// Single qubit superposition
fn main() -> MeasuredResult {
    let q = qubit();
    h(q);
    return measure(q);
}
`
	default:
		return defaultMain
	}
}
