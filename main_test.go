package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	content := []byte(`
target_dir: test-providers
providers:
  aws:
    repo: hashicorp/terraform-provider-aws
    description: AWS Provider
  azurerm:
    repo: hashicorp/terraform-provider-azurerm
    description: Azure Provider
`)
	tmpfile, err := os.CreateTemp("", "providers.*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test loading valid config
	config, err := loadConfig(tmpfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "test-providers", config.TargetDir)
	assert.Len(t, config.Providers, 2)
	assert.Equal(t, "hashicorp/terraform-provider-aws", config.Providers["aws"].Repo)
	assert.Equal(t, "AWS Provider", config.Providers["aws"].Description)

	// Test loading non-existent file
	_, err = loadConfig("nonexistent.yaml")
	assert.Error(t, err)

	// Test default target directory
	content = []byte(`
providers:
  aws:
    repo: hashicorp/terraform-provider-aws
    description: AWS Provider
`)
	if err := os.WriteFile(tmpfile.Name(), content, 0644); err != nil {
		t.Fatal(err)
	}

	config, err = loadConfig(tmpfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "terraform-providers", config.TargetDir)
}

func TestFindDocsDirectories(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "test-repo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

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
	defer os.RemoveAll(tmpDir)

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
	defer os.RemoveAll(tmpDir)

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
	// Create a temporary directory for the test repository
	tmpDir, err := os.MkdirTemp("", "test-repo-*")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize a git repository
	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	// Create docs directory and a test file
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(docsDir, "test.md")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = wt.Add("docs")
	if err != nil {
		t.Fatal(err)
	}

	_, err = wt.Commit("Initial commit", &git.CommitOptions{})
	if err != nil {
		t.Fatal(err)
	}

	return tmpDir, &mockRepo{
		Repository: repo,
		worktree:   wt,
	}
}

func TestCloneProvider(t *testing.T) {
	// Create a temporary directory for the target
	targetDir, err := os.MkdirTemp("", "test-target-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(targetDir)

	// Set up a mock repository
	repoDir, _ := setupMockRepo(t)
	defer os.RemoveAll(repoDir)

	provider := Provider{
		Repo:        repoDir,
		Description: "Test Provider",
	}

	// Test cloning provider
	err = cloneProvider("test", provider, targetDir)
	assert.NoError(t, err)

	// Verify the docs directory was cloned
	docsPath := filepath.Join(targetDir, "test", "docs")
	_, err = os.Stat(docsPath)
	assert.NoError(t, err)

	// Test cloning with invalid repository
	provider.Repo = "nonexistent"
	err = cloneProvider("invalid", provider, targetDir)
	assert.Error(t, err)
}

func TestUpdateProvider(t *testing.T) {
	// Create a temporary directory for the target
	targetDir, err := os.MkdirTemp("", "test-target-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(targetDir)

	// Set up a mock repository
	repoDir, mockRepo := setupMockRepo(t)
	defer os.RemoveAll(repoDir)

	provider := Provider{
		Repo:        repoDir,
		Description: "Test Provider",
	}

	// First clone the provider
	err = cloneProvider("test", provider, targetDir)
	assert.NoError(t, err)

	// Make a change in the source repository
	testFile := filepath.Join(repoDir, "docs", "new.md")
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = mockRepo.worktree.Add("docs/new.md")
	if err != nil {
		t.Fatal(err)
	}

	_, err = mockRepo.worktree.Commit("Update", &git.CommitOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Test updating provider
	err = updateProvider("test", targetDir)
	assert.NoError(t, err)

	// Verify the new file exists
	updatedFile := filepath.Join(targetDir, "test", "docs", "new.md")
	_, err = os.Stat(updatedFile)
	assert.NoError(t, err)

	// Test updating non-existent provider
	err = updateProvider("nonexistent", targetDir)
	assert.Error(t, err)
}
