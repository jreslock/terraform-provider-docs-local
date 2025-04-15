package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
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

// Updated cloneProvider to detect the default branch dynamically or use a user-specified branch
func cloneProvider(providerName string, provider Provider, targetDir string, branch string) error {
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

	// Detect the default branch if not specified
	if branch == "" {
		repo, err := git.Init(memory.NewStorage(), nil)
		if err != nil {
			return fmt.Errorf("error initializing in-memory repository: %v", err)
		}

		remote, err := repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{repoURL},
		})
		if err != nil {
			return fmt.Errorf("error creating remote: %v", err)
		}

		refs, err := remote.List(&git.ListOptions{})
		if err != nil {
			return fmt.Errorf("error listing remote references: %v", err)
		}

		for _, ref := range refs {
			if ref.Name().IsBranch() {
				branch = ref.Name().Short()
				break
			}
		}

		if branch == "" {
			branch = "main" // Fallback to "main" if no branch is detected
		}
	}

	// Clone the repository
	_, err := git.PlainClone(providerDir, false, &git.CloneOptions{
		URL:           repoURL,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
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
	var configFile string
	var providerName string
	var branch string

	rootCmd := &cobra.Command{
		Use:   "terraform-provider-docs-local",
		Short: "A tool for managing Terraform provider documentation",
	}

	cloneOneCmd := &cobra.Command{
		Use:   "clone-one",
		Short: "Clone a specific provider",
		Run: func(cmd *cobra.Command, args []string) {
			if providerName == "" {
				log.Fatal("Error: Provider name is required for clone-one command")
			}

			config, err := loadConfig(configFile)
			if err != nil {
				log.Fatalf("Error loading config: %v", err)
			}

			provider, exists := config.Providers[providerName]
			if !exists {
				log.Fatalf("Error: Provider '%s' not found in configuration", providerName)
			}

			if err := cloneProvider(providerName, provider, config.TargetDir, branch); err != nil {
				log.Fatalf("Error: %v", err)
			}

			if err := generateIndex(config); err != nil {
				log.Fatalf("Error generating index: %v", err)
			}
		},
	}

	cloneAllCmd := &cobra.Command{
		Use:   "clone-all",
		Short: "Clone all providers in the configuration",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := loadConfig(configFile)
			if err != nil {
				log.Fatalf("Error loading config: %v", err)
			}

			if len(config.Providers) == 0 {
				log.Fatal("Error: No providers found in configuration")
			}

			// Create target directory if it doesn't exist
			if err := os.MkdirAll(config.TargetDir, 0755); err != nil {
				log.Fatalf("Error creating target directory: %v", err)
			}

			for name, provider := range config.Providers {
				if err := cloneProvider(name, provider, config.TargetDir, branch); err != nil {
					log.Printf("Error cloning provider '%s': %v", name, err)
					// Continue with other providers even if one fails
					continue
				}
			}

			if err := generateIndex(config); err != nil {
				log.Fatalf("Error generating index: %v", err)
			}
		},
	}

	cloneOneCmd.Flags().StringVarP(&providerName, "provider", "p", "", "Provider name for clone-one command")
	cloneOneCmd.Flags().StringVarP(&branch, "branch", "b", "", "Branch to clone (default: detect from remote)")
	cloneAllCmd.Flags().StringVarP(&branch, "branch", "b", "", "Branch to clone for all providers (default: detect from remote)")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "providers.yaml", "Path to configuration file")

	rootCmd.AddCommand(cloneOneCmd)
	rootCmd.AddCommand(cloneAllCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
