package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfigInvalidFile(t *testing.T) {
	// Test loading an invalid YAML file
	tmpfile, err := os.CreateTemp("", "invalid.*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Fatalf("failed to remove temporary file: %v", err)
		}
	}()

	invalidContent := []byte(`
target_dir: test-providers
providers:
  aws:
    repo: hashicorp/terraform-provider-aws
    description: AWS Provider
  azurerm:
    repo: hashicorp/terraform-provider-azurerm
    description: Azure Provider
invalid_field: true
`)
	if _, err := tmpfile.Write(invalidContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = loadConfig(tmpfile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing config file")
}

func TestFindDocsDirectoriesEmptyRepo(t *testing.T) {
	// Test with an empty repository
	tmpDir, err := os.MkdirTemp("", "empty-repo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	paths, err := findDocsDirectories(tmpDir)
	assert.NoError(t, err)
	assert.Empty(t, paths)
}

func TestGenerateIndexNoProviders(t *testing.T) {
	// Test generating index with no providers
	tmpDir, err := os.MkdirTemp("", "test-providers-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	config := &Config{
		TargetDir: tmpDir,
		Providers: map[string]Provider{},
	}

	err = generateIndex(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no providers have been cloned yet")
}

func TestCloneProviderInvalidURL(t *testing.T) {
	// Test cloning a provider with an invalid repository URL
	tmpDir, err := os.MkdirTemp("", "test-target-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	provider := Provider{
		Repo:        "invalid-url",
		Description: "Invalid Provider",
	}

	err = cloneProvider("invalid", provider, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error cloning repository")
}

func TestUpdateProviderNoChanges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-target-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	repoDir, _ := setupMockRepo(t)
	defer func() {
		if err := os.RemoveAll(repoDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	provider := Provider{
		Repo:        repoDir, // Use the local mock repository
		Description: "Test Provider",
	}

	// Clone the provider first
	err = cloneProvider("test", provider, tmpDir)
	assert.NoError(t, err)

	// Update the provider without making any changes
	err = updateProvider("test", tmpDir)
	assert.NoError(t, err)
}

func TestCleanProvidersEmptyDir(t *testing.T) {
	// Test cleaning an empty directory
	tmpDir, err := os.MkdirTemp("", "empty-providers-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	err = cleanProviders(tmpDir)
	assert.NoError(t, err)
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err))
}

func TestFindDocsDirectoriesNestedInvalid(t *testing.T) {
	// Test with deeply nested invalid docs directories
	tmpDir, err := os.MkdirTemp("", "nested-repo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	dirs := []string{
		"deep/nested/docs", // Invalid: too deeply nested
		"docs",             // Valid: root docs
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	paths, err := findDocsDirectories(tmpDir)
	assert.NoError(t, err)
	assert.Len(t, paths, 1)
	assert.Contains(t, paths, "docs")
}

func TestFindDocsDirectories(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "test-repo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	// Create test directory structure
	dirs := []string{
		"docs",              // Valid: root docs
		"website/docs",      // Valid: website docs
		"contributing/docs", // Invalid: contributing docs
		"examples/docs",     // Invalid: examples docs
		"internal/docs",     // Invalid: internal docs
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Test finding docs directories
	paths, err := findDocsDirectories(tmpDir)
	assert.NoError(t, err)
	assert.Len(t, paths, 2)
	assert.Contains(t, paths, "docs")
	assert.Contains(t, paths, "website/docs")
}

func TestGenerateIndex(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "test-providers-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	// Create test providers with docs
	providers := map[string]struct{}{
		"aws":     {},
		"azurerm": {},
	}

	for provider := range providers {
		providerDir := filepath.Join(tmpDir, provider)
		docsDir := filepath.Join(providerDir, "docs")
		if err := os.MkdirAll(docsDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create test config
	config := &Config{
		TargetDir: tmpDir,
		Providers: map[string]Provider{
			"aws": {
				Repo:        "hashicorp/terraform-provider-aws",
				Description: "AWS Provider",
			},
			"azurerm": {
				Repo:        "hashicorp/terraform-provider-azurerm",
				Description: "Azure Provider",
			},
		},
	}

	// Test generating index
	err = generateIndex(config)
	assert.NoError(t, err)

	// Verify index.md was created
	indexPath := filepath.Join(tmpDir, "index.md")
	_, err = os.Stat(indexPath)
	assert.NoError(t, err)

	// Test with non-existent target directory
	config.TargetDir = "nonexistent"
	err = generateIndex(config)
	assert.Error(t, err)
}

func TestCleanProviders(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-providers-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	// Create some test directories
	testDirs := []string{"aws", "azurerm"}
	for _, dir := range testDirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Test cleaning existing directory
	err = cleanProviders(tmpDir)
	assert.NoError(t, err)
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err))

	// Test cleaning non-existent directory
	err = cleanProviders("nonexistent")
	assert.NoError(t, err)
}

// Mock git repository for testing clone and update operations
type mockRepo struct {
	*git.Repository
	worktree *git.Worktree
}

func setupMockRepo(t *testing.T) (string, *mockRepo) {
	tmpDir, err := os.MkdirTemp("", "test-repo-*")
	if err != nil {
		t.Fatal(err)
	}

	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	// Create an initial commit
	filePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(filePath, []byte("# Test Repo"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = worktree.Add("README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Debugging: Log the repository state after the initial commit
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("failed to get repository head: %v", err)
	}
	t.Logf("Repository HEAD after initial commit: %s", head.Hash())

	// Add a docs directory to the mock repository
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "index.md"), []byte("# Docs"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = worktree.Add("docs/index.md")
	if err != nil {
		t.Fatal(err)
	}
	_, err = worktree.Commit("Add docs directory", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a main branch
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("main"),
		Create: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	return tmpDir, &mockRepo{Repository: repo, worktree: worktree}
}

func TestCloneProvider(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-target-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	repoDir, _ := setupMockRepo(t)
	defer func() {
		if err := os.RemoveAll(repoDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	// Simplify the test by removing commit logic and focusing on cloning
	provider := Provider{
		Repo:        repoDir, // Use the local mock repository
		Description: "Test Provider",
	}

	err = cloneProvider("test", provider, tmpDir)
	assert.NoError(t, err)

	// Verify the docs directory was cloned
	_, err = os.Stat(filepath.Join(tmpDir, "test", "docs"))
	assert.NoError(t, err)
}

func TestUpdateProvider(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-target-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	repoDir, mockRepo := setupMockRepo(t)
	defer func() {
		if err := os.RemoveAll(repoDir); err != nil {
			t.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	provider := Provider{
		Repo:        repoDir, // Use the local mock repository
		Description: "Test Provider",
	}

	// Clone the provider first
	err = cloneProvider("test", provider, tmpDir)
	assert.NoError(t, err)

	// Add a new file to the mock repository
	newFilePath := filepath.Join(repoDir, "docs", "new.md")
	if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newFilePath, []byte("# New Docs"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = mockRepo.worktree.Add("docs/new.md")
	if err != nil {
		t.Fatal(err)
	}
	_, err = mockRepo.worktree.Commit("Add new docs file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
		All: true, // Ensures all changes are committed, even if empty
	})
	if err != nil {
		t.Fatal(err)
	}

	// Update the provider
	err = updateProvider("test", tmpDir)
	assert.NoError(t, err)

	// Verify the new file exists
	_, err = os.Stat(filepath.Join(tmpDir, "test", "docs", "new.md"))
	assert.NoError(t, err)
}

func TestCloneOneCommand(t *testing.T) {
	// Set up a temporary config file
	tmpfile, err := os.CreateTemp("", "config.*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Fatalf("failed to remove temporary file: %v", err)
		}
	}()

	configContent := []byte(`
    target_dir: test-providers
    providers:
      aws:
        repo: hashicorp/terraform-provider-aws
        description: AWS Provider
`)
	if _, err := tmpfile.Write(configContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Set up the cobra command
	var configFile string
	var providerName string

	rootCmd := &cobra.Command{
		Use:   "terraform-provider-docs-local",
		Short: "A tool for managing Terraform provider documentation",
	}

	cloneOneCmd := &cobra.Command{
		Use:   "clone-one",
		Short: "Clone a specific provider",
		Run: func(cmd *cobra.Command, args []string) {
			assert.Equal(t, "aws", providerName)
			assert.Equal(t, tmpfile.Name(), configFile)
		},
	}

	cloneOneCmd.Flags().StringVarP(&providerName, "provider", "p", "", "Provider name for clone-one command")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to providers configuration file")
	rootCmd.AddCommand(cloneOneCmd)

	// Simulate command-line arguments
	rootCmd.SetArgs([]string{"clone-one", "-c", tmpfile.Name(), "-p", "aws"})

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
}
