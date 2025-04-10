package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"gopkg.in/yaml.v3"
)

type Provider struct {
	Repo        string `yaml:"repo"`
	Description string `yaml:"description"`
}

type Config struct {
	TargetDir string              `yaml:"target_dir"`
	Providers map[string]Provider `yaml:"providers"`
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Check if the YAML file is empty
	if len(data) == 0 {
		return nil, fmt.Errorf("error parsing config file: file is empty")
	}

	var config Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	// Enable strict decoding to catch unexpected fields
	decoder.KnownFields(true)

	// Decode the YAML file
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Validate the structure of the decoded YAML
	if config.Providers == nil && config.TargetDir == "" {
		return nil, fmt.Errorf("invalid configuration: missing required fields")
	}

	// Check for unexpected fields in the YAML
	if len(config.Providers) == 0 && config.TargetDir == "" {
		return nil, fmt.Errorf("invalid configuration: no valid fields found")
	}

	// Validate that each provider has a valid repo and description
	for name, provider := range config.Providers {
		if provider.Repo == "" {
			return nil, fmt.Errorf("provider '%s' is missing a 'repo' field", name)
		}
		if provider.Description == "" {
			return nil, fmt.Errorf("provider '%s' is missing a 'description' field", name)
		}
	}

	// Set default target directory if not specified
	if config.TargetDir == "" {
		config.TargetDir = "terraform-providers"
	}

	return &config, nil
}

func cloneProvider(providerName string, provider Provider, targetDir string) error {
	fmt.Printf("Processing %s (%s)...\n", providerName, provider.Repo)

	// Create provider directory
	providerDir := filepath.Join(targetDir, providerName)

	// Use the local path directly if it's a local repository (for tests)
	var repoURL string
	if filepath.IsAbs(provider.Repo) || strings.HasPrefix(provider.Repo, ".") {
		repoURL = provider.Repo
	} else {
		repoURL = fmt.Sprintf("https://github.com/%s.git", provider.Repo)
	}

	// Clone the repository
	_, err := git.PlainClone(providerDir, false, &git.CloneOptions{
		URL:           repoURL,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName("main"),
	})
	if err != nil {
		return fmt.Errorf("error cloning repository: %v", err)
	}

	// Find all docs directories
	docsPaths, err := findDocsDirectories(providerDir)
	if err != nil {
		return fmt.Errorf("error finding docs directories: %v", err)
	}

	if len(docsPaths) == 0 {
		return fmt.Errorf("no docs directories found in repository")
	}

	fmt.Printf("Successfully cloned %s\n", providerName)
	return nil
}

func updateProvider(providerName string, targetDir string) error {
	providerDir := filepath.Join(targetDir, providerName)
	if _, err := os.Stat(providerDir); os.IsNotExist(err) {
		return fmt.Errorf("provider '%s' has not been cloned yet", providerName)
	}

	fmt.Printf("Updating docs for %s...\n", providerName)

	// Open the repository
	repo, err := git.PlainOpen(providerDir)
	if err != nil {
		return fmt.Errorf("error opening repository: %v", err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("error getting worktree: %v", err)
	}

	// Fetch latest changes
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"+refs/heads/main:refs/remotes/origin/main"},
		Depth:      1,
		Force:      true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("error fetching changes: %v", err)
	}

	// Get the reference to origin/main
	ref, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", "main"), true)
	if err != nil {
		return fmt.Errorf("error getting remote reference: %v", err)
	}

	// Reset to match remote
	err = worktree.Reset(&git.ResetOptions{
		Commit: ref.Hash(),
		Mode:   git.HardReset,
	})
	if err != nil {
		return fmt.Errorf("error resetting to remote: %v", err)
	}

	// Find docs directories again to ensure we have the right ones
	docsPaths, err := findDocsDirectories(providerDir)
	if err != nil {
		return fmt.Errorf("error finding docs directories: %v", err)
	}

	// Clean up any files not in docs directories
	files, err := os.ReadDir(providerDir)
	if err != nil {
		return fmt.Errorf("error reading directory: %v", err)
	}

	for _, file := range files {
		if file.Name() != ".git" {
			isDocsDir := false
			for _, docsPath := range docsPaths {
				if strings.HasPrefix(file.Name(), strings.Split(docsPath, "/")[0]) {
					isDocsDir = true
					break
				}
			}
			if !isDocsDir {
				path := filepath.Join(providerDir, file.Name())
				if err := os.RemoveAll(path); err != nil {
					return fmt.Errorf("error removing file %s: %v", path, err)
				}
			}
		}
	}

	fmt.Printf("Successfully updated docs for %s\n", providerName)
	return nil
}

// Find valid docs directories in a given path
func findDocsDirectories(root string) ([]string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "docs" {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}

			// Only accept docs at root level or under website/
			parentDir := filepath.Dir(relPath)
			if parentDir == "." || parentDir == "website" {
				paths = append(paths, relPath)
			}
		}
		return nil
	})
	return paths, err
}

func generateIndex(config *Config) error {
	if len(config.Providers) == 0 {
		return fmt.Errorf("no providers have been cloned yet") // Return an error if no providers exist
	}

	fmt.Println("Generating index.md...")

	var indexContent strings.Builder
	indexContent.WriteString(`# Terraform Provider Documentation

This directory contains documentation for various Terraform providers. Each provider's documentation is maintained in its own subdirectory.

## Available Providers

`)

	// Add entries for each provider
	for name, provider := range config.Providers {
		indexContent.WriteString(fmt.Sprintf("- [%s](%s/docs): %s\n", name, name, provider.Description))
	}

	// Write index.md in the target directory
	indexPath := filepath.Join(config.TargetDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(indexContent.String()), 0644); err != nil {
		return fmt.Errorf("error writing index file: %v", err)
	}

	return nil
}

func cleanProviders(targetDir string) error {
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Println("No providers directory found")
		return nil
	}

	fmt.Println("Removing all cloned providers...")
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("error removing providers directory: %v", err)
	}

	fmt.Println("Cleanup complete")
	return nil
}

func main() {
	configFile := flag.String("config", "providers.yaml", "Path to providers configuration file")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: terraform-provider-docs-local [command] [options]")
		fmt.Println("Commands:")
		fmt.Println("  clone-all    Clone all providers and set up sparse checkout")
		fmt.Println("  clone-one    Clone a specific provider")
		fmt.Println("  list         List all available providers")
		fmt.Println("  clean        Remove all cloned providers")
		fmt.Println("  update-all   Update docs for all providers")
		fmt.Println("  update-one   Update docs for a specific provider")
		fmt.Println("  index        Generate index.md file")
		fmt.Println("\nOptions:")
		fmt.Println("  -config <file>  Path to providers configuration file")
		fmt.Println("  -p <name>       Provider name for clone-one/update-one commands")
		return
	}

	config, err := loadConfig(*configFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	command := args[0]
	switch command {
	case "clone-all":
		for name, provider := range config.Providers {
			if err := cloneProvider(name, provider, config.TargetDir); err != nil {
				fmt.Printf("Error cloning %s: %v\n", name, err)
			}
		}
		fmt.Println("All providers have been processed!")
		if err := generateIndex(config); err != nil {
			fmt.Printf("Error generating index: %v\n", err)
		}

	case "clone-one":
		providerName := flag.String("p", "", "Provider name")
		flag.Parse()
		if *providerName == "" {
			fmt.Println("Error: Provider name is required for clone-one command")
			os.Exit(1)
		}
		provider, exists := config.Providers[*providerName]
		if !exists {
			fmt.Printf("Error: Provider '%s' not found in configuration\n", *providerName)
			os.Exit(1)
		}
		if err := cloneProvider(*providerName, provider, config.TargetDir); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if err := generateIndex(config); err != nil {
			fmt.Printf("Error generating index: %v\n", err)
		}

	case "list":
		fmt.Println("Available Terraform providers:")
		for name, provider := range config.Providers {
			fmt.Printf("  - %s (%s)\n", name, provider.Repo)
		}

	case "clean":
		if err := cleanProviders(config.TargetDir); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "update-all":
		for name := range config.Providers {
			if err := updateProvider(name, config.TargetDir); err != nil {
				fmt.Printf("Error updating %s: %v\n", name, err)
			}
		}
		fmt.Println("All provider docs have been updated!")
		if err := generateIndex(config); err != nil {
			fmt.Printf("Error generating index: %v\n", err)
		}

	case "update-one":
		providerName := flag.String("p", "", "Provider name")
		flag.Parse()
		if *providerName == "" {
			fmt.Println("Error: Provider name is required for update-one command")
			os.Exit(1)
		}
		if err := updateProvider(*providerName, config.TargetDir); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if err := generateIndex(config); err != nil {
			fmt.Printf("Error generating index: %v\n", err)
		}

	case "index":
		if err := generateIndex(config); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Error: Unknown command '%s'\n", command)
		os.Exit(1)
	}
}
