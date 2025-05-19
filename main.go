package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed template/**
var templateFS embed.FS

// === ANSI color helpers ===
func red(text string) string    { return "\033[31m" + text + "\033[0m" }
func green(text string) string  { return "\033[32m" + text + "\033[0m" }
func yellow(text string) string { return "\033[33m" + text + "\033[0m" }
func blue(text string) string   { return "\033[34m" + text + "\033[0m" }
func cyan(text string) string   { return "\033[36m" + text + "\033[0m" }

// === Utility ===
func runCommandInDir(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// === Steps ===

func createProjectDir(name string, force bool) {
	if _, err := os.Stat(name); err == nil && !force {
		fmt.Println(yellow(fmt.Sprintf("⚠️ Folder '%s' already exists. Use --force to overwrite.", name)))
		os.Exit(1)
	}
}

func copyTemplate(projectName string) error {
	return fs.WalkDir(templateFS, "template", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.TrimPrefix(path, "template")
		targetPath := filepath.Join(projectName, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		}

		content, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}

		updated := strings.ReplaceAll(string(content), "{{projectName}}", projectName)
		if err := os.WriteFile(targetPath, []byte(updated), 0644); err != nil {
			return err
		}

		fmt.Println(green("✔ " + targetPath))
		return nil
	})
}

func createEnvFile(projectName string) error {
	envContent := `
# Application environment
APP_ENV=development
PORT=80
DEBUG=true

# Database configuration
DB_HOST=localhost
DB_USER=salut
DB_NAME=dbname
DB_PASSWORD=salutpassword
DB_PORT=5432

# Google OAuth configuration
GOOGLE_CLIENT_ID=pattern.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GX-pattern
GOOGLE_REDIRECT_URL=http://localhost:80/auth/google/callback

# Admin configuration
ADMIN=admin@admin
ADMIN_PASSWORD=supersecurepassword101MM
`
	envPath := filepath.Join(projectName, ".env")
	return os.WriteFile(envPath, []byte(envContent), 0644)
}

func createMakefile(projectName string) error {
	makefileContent := `esbuild :
	esbuild --bundle --minify --outdir=./web/static/js/ --watch ./web/source/*.ts 
tailwind : 
	tailwindcss -i ./web/source/app.css -o ./web/static/css/style.css --watch --optimize
`
	makefilePath := filepath.Join(projectName, "Makefile")
	return os.WriteFile(makefilePath, []byte(makefileContent), 0644)
}

func initTools(projectName string) {
	steps := []struct {
		label   string
		command []string
	}{
		{"🦕 Running `deno install`...", []string{"deno", "install"}},
		{"🔧 Running `go mod init`...", []string{"go", "mod", "init", projectName}},
		{"📦 Running `go mod tidy`...", []string{"go", "mod", "tidy"}},
	}

	for _, step := range steps {
		fmt.Println(cyan(step.label))
		if err := runCommandInDir(projectName, step.command[0], step.command[1:]...); err != nil {
			fmt.Println(red(fmt.Sprintf("❌ %s failed", step.command[1])))
			os.Exit(1)
		}
	}
}

func main() {
	projectName := flag.String("name", "", "Name of the project to create")
	force := flag.Bool("force", false, "Force overwrite if the folder already exists")
	flag.Parse()

	if *projectName == "" {
		fmt.Println(red("❌ You must provide a project name using --name"))
		os.Exit(1)
	}

	fmt.Println(cyan(fmt.Sprintf("🚀 Creating project: %s", *projectName)))
	createProjectDir(*projectName, *force)

	if err := copyTemplate(*projectName); err != nil {
		fmt.Println(red(fmt.Sprintf("🔥 Error copying template: %v", err)))
		os.Exit(1)
	}

	if err := createEnvFile(*projectName); err != nil {
		fmt.Println(red(fmt.Sprintf("❌ Failed to create .env file: %v", err)))
		os.Exit(1)
	}
	fmt.Println(green("✔ .env file created"))

	if err := createMakefile(*projectName); err != nil {
		fmt.Println(red(fmt.Sprintf("❌ Failed to create Makefile: %v", err)))
		os.Exit(1)
	}
	fmt.Println(green("✔ Makefile created"))

	initTools(*projectName)

	fmt.Println(blue(fmt.Sprintf("🎉 Project '%s' created and ready!", *projectName)))
}
