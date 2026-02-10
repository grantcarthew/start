//go:build integration

package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/start/internal/cli"
	"github.com/grantcarthew/start/internal/registry"
)

// TestIntegration_AssetsIndexCommand tests the index command with a real asset repo.
func TestIntegration_AssetsIndexCommand(t *testing.T) {
	// Create a mock asset repository structure
	tmpDir := t.TempDir()

	// Create category directories
	agentsDir := filepath.Join(tmpDir, "agents", "ai", "test-agent")
	rolesDir := filepath.Join(tmpDir, "roles", "golang", "assistant")

	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("creating agents dir: %v", err)
	}
	if err := os.MkdirAll(rolesDir, 0755); err != nil {
		t.Fatalf("creating roles dir: %v", err)
	}

	// Create cue.mod directories to indicate CUE modules
	if err := os.MkdirAll(filepath.Join(agentsDir, "cue.mod"), 0755); err != nil {
		t.Fatalf("creating agent cue.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rolesDir, "cue.mod"), 0755); err != nil {
		t.Fatalf("creating role cue.mod: %v", err)
	}

	// Write module.cue files
	agentModule := `module: "github.com/test/agents/ai/test-agent"
language: version: "v0.12.0"
`
	if err := os.WriteFile(filepath.Join(agentsDir, "cue.mod", "module.cue"), []byte(agentModule), 0644); err != nil {
		t.Fatalf("writing agent module.cue: %v", err)
	}

	roleModule := `module: "github.com/test/roles/golang/assistant"
language: version: "v0.12.0"
`
	if err := os.WriteFile(filepath.Join(rolesDir, "cue.mod", "module.cue"), []byte(roleModule), 0644); err != nil {
		t.Fatalf("writing role module.cue: %v", err)
	}

	// Write CUE files with asset definitions
	agentCUE := `package "test-agent"

agent: {
	bin: "test-agent"
	command: "{{.bin}} --model {{.model}}"
	description: "Test agent for integration tests"
	tags: ["test", "integration"]
}
`
	if err := os.WriteFile(filepath.Join(agentsDir, "agent.cue"), []byte(agentCUE), 0644); err != nil {
		t.Fatalf("writing agent.cue: %v", err)
	}

	roleCUE := `package assistant

role: {
	description: "Go programming assistant"
	prompt: "You are a Go expert."
	tags: ["golang", "programming"]
}
`
	if err := os.WriteFile(filepath.Join(rolesDir, "role.cue"), []byte(roleCUE), 0644); err != nil {
		t.Fatalf("writing role.cue: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Run the index command via Cobra
	buf := new(bytes.Buffer)
	cmd := cli.NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"assets", "index"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("assets index failed: %v", err)
	}

	output := buf.String()

	// Verify output mentions assets indexed
	if !strings.Contains(output, "Generated index") {
		t.Errorf("output should mention generated index, got: %s", output)
	}

	// Verify index file was created
	indexPath := filepath.Join(tmpDir, "index", "index.cue")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("index/index.cue should have been created")
	}

	// Read and verify index content
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}

	indexContent := string(indexData)

	if !strings.Contains(indexContent, "package index") {
		t.Error("index should have package declaration")
	}
	if !strings.Contains(indexContent, "agents:") {
		t.Error("index should have agents section")
	}
	if !strings.Contains(indexContent, "roles:") {
		t.Error("index should have roles section")
	}
	if !strings.Contains(indexContent, "ai/test-agent") {
		t.Error("index should contain ai/test-agent")
	}
	if !strings.Contains(indexContent, "golang/assistant") {
		t.Error("index should contain golang/assistant")
	}
}

// TestIntegration_AssetsIndexNotAssetRepo tests error when not in asset repo.
func TestIntegration_AssetsIndexNotAssetRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a non-asset directory
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	buf := new(bytes.Buffer)
	cmd := cli.NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"assets", "index"})

	err = cmd.Execute()
	if err == nil {
		t.Error("expected error when not in asset repo")
	}

	if !strings.Contains(err.Error(), "not an asset repository") {
		t.Errorf("error should mention 'not an asset repository', got: %v", err)
	}
}

// TestIntegration_AssetsListWithConfig tests listing assets from config.
func TestIntegration_AssetsListWithConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config directory
	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	// Write config with assets
	config := `
agents: {
	claude: {
		bin: "claude"
		command: "{{.bin}} --model {{.model}}"
		origin: "github.com/grantcarthew/start-assets/agents/ai/claude@v0.2.0"
	}
}

roles: {
	assistant: {
		prompt: "You are a helpful assistant."
		origin: "github.com/grantcarthew/start-assets/roles/assistant@v0.1.0"
	}
	reviewer: {
		prompt: "You are a code reviewer."
		origin: "github.com/grantcarthew/start-assets/roles/reviewer@v0.3.1"
	}
}

tasks: {
	review: {
		prompt: "Review this code."
		origin: "github.com/grantcarthew/start-assets/tasks/review@v0.1.0"
	}
}
`
	if err := os.WriteFile(filepath.Join(configDir, "settings.cue"), []byte(config), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Override HOME to isolate from global config
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	buf := new(bytes.Buffer)
	cmd := cli.NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"assets", "list"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("assets list failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Installed assets") {
		t.Errorf("output should mention installed assets, got: %s", output)
	}
	if !strings.Contains(output, "agents/") {
		t.Errorf("output should show agents category, got: %s", output)
	}
	if !strings.Contains(output, "claude") {
		t.Errorf("output should show claude agent, got: %s", output)
	}
	if !strings.Contains(output, "roles/") {
		t.Errorf("output should show roles category, got: %s", output)
	}
	if !strings.Contains(output, "assistant") {
		t.Errorf("output should show assistant role, got: %s", output)
	}
	if !strings.Contains(output, "v0.2.0") {
		t.Errorf("output should show claude version, got: %s", output)
	}
	if !strings.Contains(output, "v0.1.0") {
		t.Errorf("output should show assistant version, got: %s", output)
	}
}

// TestIntegration_AssetsListNoConfig tests listing when no config exists.
func TestIntegration_AssetsListNoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Override HOME to isolate
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	buf := new(bytes.Buffer)
	cmd := cli.NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"assets", "list"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("assets list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No configuration found") {
		t.Errorf("should report no config, got: %s", output)
	}
}

// TestIntegration_SearchIndex tests the search functionality.
func TestIntegration_SearchIndex(t *testing.T) {
	// This test uses the internal search function directly
	// since we can't easily mock the registry fetch

	index := &registry.Index{
		Agents: map[string]registry.IndexEntry{
			"ai/claude": {
				Module:      "github.com/test/agents/ai/claude@v0",
				Description: "Claude by Anthropic",
				Tags:        []string{"anthropic", "ai"},
			},
		},
		Roles: map[string]registry.IndexEntry{
			"golang/assistant": {
				Module:      "github.com/test/roles/golang/assistant@v0",
				Description: "Go programming expert",
				Tags:        []string{"golang", "programming"},
			},
			"golang/reviewer": {
				Module:      "github.com/test/roles/golang/reviewer@v0",
				Description: "Go code reviewer",
				Tags:        []string{"golang", "review"},
			},
		},
		Tasks:    map[string]registry.IndexEntry{},
		Contexts: map[string]registry.IndexEntry{},
	}

	// Test search for "golang" - should find both roles
	results := searchIndexEntries(index, "golang")
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'golang', got %d", len(results))
	}

	// Test search for "claude" - should find agent
	results = searchIndexEntries(index, "claude")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'claude', got %d", len(results))
	}
	if len(results) > 0 && results[0].Name != "ai/claude" {
		t.Errorf("expected ai/claude, got %s", results[0].Name)
	}

	// Test search for "programming" - should match description
	results = searchIndexEntries(index, "programming")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'programming', got %d", len(results))
	}
}

// searchResult mirrors cli.SearchResult for testing
type searchResult struct {
	Category string
	Name     string
	Entry    registry.IndexEntry
}

// searchIndexEntries is a copy of the search logic for integration testing
func searchIndexEntries(index *registry.Index, query string) []searchResult {
	var results []searchResult
	queryLower := strings.ToLower(query)

	// Search agents
	for name, entry := range index.Agents {
		if matchesQuery(name, entry, queryLower) {
			results = append(results, searchResult{Category: "agents", Name: name, Entry: entry})
		}
	}

	// Search roles
	for name, entry := range index.Roles {
		if matchesQuery(name, entry, queryLower) {
			results = append(results, searchResult{Category: "roles", Name: name, Entry: entry})
		}
	}

	// Search tasks
	for name, entry := range index.Tasks {
		if matchesQuery(name, entry, queryLower) {
			results = append(results, searchResult{Category: "tasks", Name: name, Entry: entry})
		}
	}

	// Search contexts
	for name, entry := range index.Contexts {
		if matchesQuery(name, entry, queryLower) {
			results = append(results, searchResult{Category: "contexts", Name: name, Entry: entry})
		}
	}

	return results
}

func matchesQuery(name string, entry registry.IndexEntry, queryLower string) bool {
	if strings.Contains(strings.ToLower(name), queryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Description), queryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Module), queryLower) {
		return true
	}
	for _, tag := range entry.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			return true
		}
	}
	return false
}

// TestIntegration_AssetsCommandHelp tests that help works for assets commands.
func TestIntegration_AssetsCommandHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "assets help",
			args: []string{"assets", "--help"},
			want: []string{"Manage assets", "browse", "search", "add", "list", "info", "update", "index"},
		},
		{
			name: "assets search help",
			args: []string{"assets", "search", "--help"},
			want: []string{"Search", "query", "3 characters"},
		},
		{
			name: "assets add help",
			args: []string{"assets", "add", "--help"},
			want: []string{"Install", "--local"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := cli.NewRootCmd()
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			_ = cmd.Execute() // Help returns nil

			output := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("help output missing %q, got: %s", want, output)
				}
			}
		})
	}
}
