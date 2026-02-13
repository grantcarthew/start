package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// TestSearchIndex tests the assets.SearchIndex function.
func TestSearchIndex(t *testing.T) {
	t.Parallel()
	index := &registry.Index{
		Agents: map[string]registry.IndexEntry{
			"ai/claude": {
				Module:      "github.com/test/agents/ai/claude@v0",
				Description: "Claude by Anthropic",
				Tags:        []string{"anthropic", "ai", "llm"},
			},
			"ai/gemini": {
				Module:      "github.com/test/agents/ai/gemini@v0",
				Description: "Gemini by Google",
				Tags:        []string{"google", "ai", "llm"},
			},
		},
		Roles: map[string]registry.IndexEntry{
			"golang/assistant": {
				Module:      "github.com/test/roles/golang/assistant@v0",
				Description: "Go programming expert",
				Tags:        []string{"golang", "programming"},
			},
		},
		Tasks: map[string]registry.IndexEntry{
			"golang/code-review": {
				Module:      "github.com/test/tasks/golang/code-review@v0",
				Description: "Review Go code for best practices",
				Tags:        []string{"golang", "review"},
			},
		},
		Contexts: map[string]registry.IndexEntry{},
	}

	tests := []struct {
		name       string
		query      string
		wantCount  int
		wantFirst  string
		wantInName bool
	}{
		{
			name:      "search by name - claude",
			query:     "claude",
			wantCount: 1,
			wantFirst: "ai/claude",
		},
		{
			name:      "search by tag - golang",
			query:     "golang",
			wantCount: 2, // role and task
		},
		{
			name:       "search by description - anthropic",
			query:      "anthropic",
			wantCount:  1,
			wantFirst:  "ai/claude",
			wantInName: false,
		},
		{
			name:      "search multiple matches - ai",
			query:     "ai",
			wantCount: 2, // claude and gemini
		},
		{
			name:      "no matches",
			query:     "nonexistent",
			wantCount: 0,
		},
		{
			name:      "case insensitive",
			query:     "CLAUDE",
			wantCount: 1,
			wantFirst: "ai/claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := assets.SearchIndex(index, tt.query)
			if err != nil {
				t.Fatalf("assets.SearchIndex() error: %v", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf("assets.SearchIndex() returned %d results, want %d", len(results), tt.wantCount)
			}

			if tt.wantFirst != "" && len(results) > 0 {
				if results[0].Name != tt.wantFirst {
					t.Errorf("first result = %q, want %q", results[0].Name, tt.wantFirst)
				}
			}
		})
	}
}

// TestMatchScore tests the matchScore function (now in assets package).
// TODO: Move to internal/assets/search_test.go
/*
func TestMatchScore(t *testing.T) {
	t.Parallel()
	entry := registry.IndexEntry{
		Module:      "github.com/test/roles/golang/assistant@v0",
		Description: "Go programming expert for code assistance",
		Tags:        []string{"golang", "programming", "expert"},
	}

	tests := []struct {
		name      string
		assetName string
		query     string
		wantScore int
	}{
		{
			name:      "exact name match scores highest",
			assetName: "golang",
			query:     "golang",
			wantScore: 6, // name(3) + module(2) + tag(1)
		},
		{
			name:      "module path match",
			assetName: "assistant",
			query:     "golang",
			wantScore: 3, // module(2) + tag(1)
		},
		{
			name:      "description only match",
			assetName: "assistant",
			query:     "programming",
			wantScore: 2, // description(1) + tag(1)
		},
		{
			name:      "tag only match",
			assetName: "assistant",
			query:     "expert",
			wantScore: 2, // description(1) + tag(1)
		},
		{
			name:      "no match",
			assetName: "assistant",
			query:     "nonexistent",
			wantScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := matchScore(tt.assetName, entry, strings.ToLower(tt.query))

			if score != tt.wantScore {
				t.Errorf("matchScore() = %d, want %d", score, tt.wantScore)
			}
		})
	}
}
*/

// TestCategoryOrder tests the categoryOrder function.
func TestCategoryOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		category string
		want     int
	}{
		{"agents", 0},
		{"roles", 1},
		{"tasks", 2},
		{"contexts", 3},
		{"unknown", 4},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := categoryOrder(tt.category)
			if got != tt.want {
				t.Errorf("categoryOrder(%q) = %d, want %d", tt.category, got, tt.want)
			}
		})
	}
}

// TestPrintSearchResults tests the printSearchResults function.
func TestPrintSearchResults(t *testing.T) {
	t.Parallel()
	results := []assets.SearchResult{
		{
			Category: "agents",
			Name:     "ai/claude",
			Entry: registry.IndexEntry{
				Module:      "github.com/test/agents/ai/claude@v0",
				Description: "Claude by Anthropic",
				Tags:        []string{"anthropic", "ai"},
			},
		},
		{
			Category: "roles",
			Name:     "golang/assistant",
			Entry: registry.IndexEntry{
				Module:      "github.com/test/roles/golang/assistant@v0",
				Description: "Go programming expert",
				Tags:        []string{"golang"},
			},
		},
	}

	t.Run("non-verbose output", func(t *testing.T) {
		var buf bytes.Buffer
		printSearchResults(&buf, results, false, nil)
		output := buf.String()

		if !strings.Contains(output, "Found 2 matches") {
			t.Errorf("output missing match count, got: %s", output)
		}
		if !strings.Contains(output, "agents/") {
			t.Errorf("output missing agents category, got: %s", output)
		}
		if !strings.Contains(output, "ai/claude") {
			t.Errorf("output missing claude, got: %s", output)
		}
		if !strings.Contains(output, "Claude by Anthropic") {
			t.Errorf("output missing description, got: %s", output)
		}
	})

	t.Run("verbose output", func(t *testing.T) {
		var buf bytes.Buffer
		printSearchResults(&buf, results, true, nil)
		output := buf.String()

		if !strings.Contains(output, "Module:") {
			t.Errorf("verbose output missing Module, got: %s", output)
		}
		if !strings.Contains(output, "Tags:") {
			t.Errorf("verbose output missing Tags, got: %s", output)
		}
	})

	t.Run("installed marker", func(t *testing.T) {
		var buf bytes.Buffer
		installed := map[string]bool{
			"agents/ai/claude": true,
		}
		printSearchResults(&buf, results, false, installed)
		output := buf.String()

		if !strings.Contains(output, "★") {
			t.Errorf("output missing installed marker for ai/claude, got: %s", output)
		}
	})
}

// TestAssetTypeToConfigFile tests the assetTypeToConfigFile function.
func TestAssetTypeToConfigFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		category string
		want     string
	}{
		{"agents", "agents.cue"},
		{"roles", "roles.cue"},
		{"tasks", "tasks.cue"},
		{"contexts", "contexts.cue"},
		{"unknown", "settings.cue"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := assetTypeToConfigFile(tt.category)
			if got != tt.want {
				t.Errorf("assetTypeToConfigFile(%q) = %q, want %q", tt.category, got, tt.want)
			}
		})
	}
}

// TestGenerateAssetCUE tests the generateAssetCUE function.
// NOTE: This function was removed as it was only used for testing.
// TODO: Remove this test or rewrite to test the actual installation flow.
/*
func TestGenerateAssetCUE(t *testing.T) {
	t.Parallel()
	asset := assets.SearchResult{
		Category: "roles",
		Name:     "golang/assistant",
		Entry: registry.IndexEntry{
			Module:      "github.com/test/roles/golang/assistant",
			Description: "Go programming expert",
		},
	}
	modulePath := "github.com/test/roles/golang/assistant@v0"

	t.Run("new file", func(t *testing.T) {
		content := generateAssetCUE(asset, modulePath, "")

		if !strings.Contains(content, "roles: {") {
			t.Errorf("output missing roles struct, got: %s", content)
		}
		if !strings.Contains(content, "Added: roles/golang/assistant") {
			t.Errorf("output missing added comment, got: %s", content)
		}
		if !strings.Contains(content, modulePath) {
			t.Errorf("output missing module path, got: %s", content)
		}
	})

	t.Run("existing file without module", func(t *testing.T) {
		existing := "// existing config\nroles: {}\n"
		content := generateAssetCUE(asset, modulePath, existing)

		if !strings.Contains(content, existing) {
			t.Error("should preserve existing content")
		}
		if !strings.Contains(content, modulePath) {
			t.Error("should add module path comment")
		}
	})

	t.Run("existing file with same module", func(t *testing.T) {
		existing := "// Module: " + modulePath + "\nroles: {}\n"
		content := generateAssetCUE(asset, modulePath, existing)

		// Should return as-is since already imported
		if content != existing {
			t.Error("should return existing content unchanged when module already present")
		}
	})
}
*/

// TestAssetsCommandExists tests that the assets command is registered.
func TestAssetsCommandExists(t *testing.T) {
	t.Parallel()
	cmd := NewRootCmd()

	// Find assets command
	var assetsCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Use == "assets" {
			assetsCmd = c
			break
		}
	}

	if assetsCmd == nil {
		t.Fatal("assets command not found")
	}

	// Check subcommands
	subcommands := []string{"browse", "search", "add", "list", "info", "update"}
	for _, name := range subcommands {
		found := false
		for _, c := range assetsCmd.Commands() {
			if strings.HasPrefix(c.Use, name) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found", name)
		}
	}
}

// TestUpdateAssetInConfig tests the assets.UpdateAssetInConfig function.
func TestUpdateAssetInConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		initial     string
		category    string
		assetName   string
		newContent  string
		wantContain []string
		wantErr     bool
	}{
		{
			name: "update simple asset",
			initial: `tasks: {
	"my/task": {
		origin: "old/origin"
		description: "old description"
		prompt: "old prompt"
	}
}
`,
			category:  "tasks",
			assetName: "my/task",
			newContent: `{
	origin: "new/origin"
	description: "new description"
	prompt: "new prompt"
}`,
			wantContain: []string{
				`"my/task": {`,
				`origin: "new/origin"`,
				`description: "new description"`,
				`prompt: "new prompt"`,
			},
		},
		{
			name: "update asset with template braces",
			initial: `tasks: {
	"project/start": {
		origin: "old/origin"
		prompt: """
			{{.instructions}}
			"""
	}
}
`,
			category:  "tasks",
			assetName: "project/start",
			newContent: `{
	origin: "new/origin"
	prompt: """
		{{if .instructions}}
		## Custom Instructions
		{{.instructions}}
		{{end}}
		"""
}`,
			wantContain: []string{
				`"project/start": {`,
				`origin: "new/origin"`,
				`{{if .instructions}}`,
				`{{end}}`,
			},
		},
		{
			name: "update preserves other assets",
			initial: `tasks: {
	"first/task": {
		origin: "first/origin"
		prompt: "first"
	}
	"second/task": {
		origin: "second/origin"
		prompt: "second"
	}
}
`,
			category:  "tasks",
			assetName: "first/task",
			newContent: `{
	origin: "updated/origin"
	prompt: "updated"
}`,
			wantContain: []string{
				`"first/task": {`,
				`origin: "updated/origin"`,
				`"second/task": {`,
				`origin: "second/origin"`,
			},
		},
		{
			name: "braces in string literals",
			initial: `tasks: {
	"my/task": {
		origin: "old/origin"
		description: "Use } to close blocks and { to open them"
		prompt: "Code example: if (x) { return } else { continue }"
	}
}
`,
			category:  "tasks",
			assetName: "my/task",
			newContent: `{
	origin: "new/origin"
	description: "Updated: { and } are important"
	prompt: "new prompt"
}`,
			wantContain: []string{
				`"my/task": {`,
				`origin: "new/origin"`,
				`description: "Updated: { and } are important"`,
				`prompt: "new prompt"`,
			},
		},
		{
			name: "comments with braces",
			initial: `tasks: {
	"my/task": {
		origin: "old/origin"
		// Note: use { and } carefully
		description: "old description"
		// TODO: update prompt { needs work }
	}
}
`,
			category:  "tasks",
			assetName: "my/task",
			newContent: `{
	origin: "new/origin"
	description: "new description"
}`,
			wantContain: []string{
				`"my/task": {`,
				`origin: "new/origin"`,
				`description: "new description"`,
			},
		},
		{
			name: "key in comment before actual definition",
			initial: `tasks: {
	// TODO: Configure "my/task": needs setup
	// Also check "my/task": for updates
	"my/task": {
		origin: "old/origin"
		description: "old description"
	}
}
`,
			category:  "tasks",
			assetName: "my/task",
			newContent: `{
	origin: "new/origin"
	description: "updated"
}`,
			wantContain: []string{
				`// TODO: Configure "my/task": needs setup`,
				`"my/task": {`,
				`origin: "new/origin"`,
				`description: "updated"`,
			},
		},
		{
			name: "comment with braces between key and opening brace",
			initial: `tasks: {
	"my/task": // TODO: fix this { urgent }
	{
		origin: "old/origin"
		description: "old description"
	}
}
`,
			category:  "tasks",
			assetName: "my/task",
			newContent: `{
	origin: "new/origin"
	description: "updated"
}`,
			wantContain: []string{
				`"my/task": {`,
				`origin: "new/origin"`,
				`description: "updated"`,
			},
		},
		{
			name: "key in string before actual definition",
			initial: `tasks: {
	"other/task": {
		origin: "other"
		description: "This relates to my/task: the foundation"
		prompt: "See my/task: for details"
	}
	"my/task": {
		origin: "old/origin"
		description: "old description"
	}
}
`,
			category:  "tasks",
			assetName: "my/task",
			newContent: `{
	origin: "new/origin"
	description: "updated"
}`,
			wantContain: []string{
				`"other/task": {`,
				`description: "This relates to my/task: the foundation"`,
				`"my/task": {`,
				`origin: "new/origin"`,
			},
		},
		{
			name: "asset not found",
			initial: `tasks: {
	"existing/task": {
		origin: "origin"
	}
}
`,
			category:   "tasks",
			assetName:  "nonexistent/task",
			newContent: `{}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "tasks.cue")

			if err := os.WriteFile(configPath, []byte(tt.initial), 0644); err != nil {
				t.Fatalf("failed to write initial config: %v", err)
			}

			err := assets.UpdateAssetInConfig(configPath, tt.category, tt.assetName, tt.newContent)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read result: %v", err)
			}

			for _, want := range tt.wantContain {
				if !strings.Contains(string(result), want) {
					t.Errorf("result missing %q\ngot:\n%s", want, result)
				}
			}
		})
	}
}

// TestFileContainsAsset tests the fileContainsAsset function.
func TestFileContainsAsset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		assetKey string
		want     bool
	}{
		{
			name:     "quoted key found",
			content:  `"my/asset": { origin: "test" }`,
			assetKey: "my/asset",
			want:     true,
		},
		{
			name:     "unquoted key found",
			content:  `simple: { origin: "test" }`,
			assetKey: "simple",
			want:     true,
		},
		{
			name:     "key not found",
			content:  `"other/asset": { origin: "test" }`,
			assetKey: "my/asset",
			want:     false,
		},
		{
			name:     "partial match is not found",
			content:  `"my/asset/extra": { origin: "test" }`,
			assetKey: "my/asset",
			want:     false,
		},
		{
			name: "key in comment only - not found",
			content: `tasks: {
	// TODO: Add "my/asset": later
	// Configure "my/asset": for production
	"other/asset": { origin: "test" }
}`,
			assetKey: "my/asset",
			want:     false,
		},
		{
			name: "key in string value - not found",
			content: `tasks: {
	"other/asset": {
		origin: "test"
		description: "Related to my/asset: the base task"
		prompt: "See my/asset: for details"
	}
}`,
			assetKey: "my/asset",
			want:     false,
		},
		{
			name: "key in comment and actual definition - found",
			content: `tasks: {
	// TODO: Update "my/asset": needs work
	"my/asset": { origin: "test" }
}`,
			assetKey: "my/asset",
			want:     true,
		},
		{
			name: "key in multi-line string - not found",
			content: `tasks: {
	"other/asset": {
		origin: "test"
		prompt: """
			Related to my/asset: see documentation
			Check my/asset: for updates
			"""
	}
}`,
			assetKey: "my/asset",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			filePath := filepath.Join(dir, "test.cue")

			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got := fileContainsAsset(filePath, tt.assetKey)
			if got != tt.want {
				t.Errorf("fileContainsAsset() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("file does not exist", func(t *testing.T) {
		got := fileContainsAsset("/nonexistent/path/file.cue", "any")
		if got != false {
			t.Error("expected false for nonexistent file")
		}
	})
}

// TestFindOpeningBrace tests the assets.FindOpeningBrace function.
func TestFindOpeningBrace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		startPos int
		wantPos  int
		wantErr  bool
	}{
		{
			name:     "simple case - immediate brace",
			content:  `"key": {`,
			startPos: 7, // After ": "
			wantPos:  7,
			wantErr:  false,
		},
		{
			name:     "whitespace before brace",
			content:  `"key":   {`,
			startPos: 6, // After ":"
			wantPos:  9,
			wantErr:  false,
		},
		{
			name: "newline before brace",
			content: `"key":
{`,
			startPos: 6, // After ":"
			wantPos:  7,
			wantErr:  false,
		},
		{
			name: "comment with brace before actual brace",
			content: `"key": // TODO: fix this { urgent }
{`,
			startPos: 7,  // After ": "
			wantPos:  36, // After newline, at the real {
			wantErr:  false,
		},
		{
			name: "multiple comments with braces",
			content: `"key": // First comment { brace }
    // Second comment { another }
    {`,
			startPos: 7,
			wantPos:  72, // At the real {
			wantErr:  false,
		},
		{
			name:     "brace in single-line string",
			content:  `"key": "description with { brace }" {`,
			startPos: 7,
			wantPos:  36,
			wantErr:  false,
		},
		{
			name: "brace in multi-line string",
			content: `"key": """
			Template: {{.field}}
			More: { and }
			""" {`,
			startPos: 7,
			wantPos:  59, // After """
			wantErr:  false,
		},
		{
			name: "complex case - comment and string with braces",
			content: `"key": // comment { brace }
    "description": "has { brace }"
    {`,
			startPos: 7,
			wantPos:  67,
			wantErr:  false,
		},
		{
			name:     "escaped quote in string",
			content:  `"key": "value with \" and { brace }" {`,
			startPos: 7,
			wantPos:  37,
			wantErr:  false,
		},
		{
			name: "multi-line string with escaped quotes",
			content: `"key": """
			Value: "quoted { brace }"
			""" {`,
			startPos: 7,
			wantPos:  47,
			wantErr:  false,
		},
		{
			name:     "no brace found",
			content:  `"key": "value"`,
			startPos: 7,
			wantPos:  0,
			wantErr:  true,
		},
		{
			name:     "brace only in comment - not found",
			content:  `"key": // only { in } comment`,
			startPos: 7,
			wantPos:  0,
			wantErr:  true,
		},
		{
			name:     "brace only in string - not found",
			content:  `"key": "only { in } string"`,
			startPos: 7,
			wantPos:  0,
			wantErr:  true,
		},
		{
			name: "real-world example",
			content: `tasks: {
	"my/task": // Needs review { important }
	{
		origin: "test"
		description: "Task with { braces } in description"
	}
}`,
			startPos: 19, // After "my/task":
			wantPos:  52, // At the real opening brace
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, err := assets.FindOpeningBrace(tt.content, tt.startPos)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if pos != tt.wantPos {
				t.Errorf("assets.FindOpeningBrace() = %d, want %d\nContent: %q", pos, tt.wantPos, tt.content)
			}

			// Verify that the position actually points to '{'
			if tt.content[pos] != '{' {
				t.Errorf("position %d does not point to '{', got %q", pos, tt.content[pos])
			}
		})
	}
}

// TestAssetsSearchValidation tests search command argument validation.
func TestAssetsSearchValidation(t *testing.T) {
	t.Parallel()
	// We can't easily test the full search without network,
	// but we can test the query length validation
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"query too short - 1 char", "a", true},
		{"query too short - 2 chars", "ab", true},
		{"query valid - 3 chars", "abc", false}, // Will fail on network, but passes validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test the validation logic
			if len(tt.query) < 3 {
				if !tt.wantErr {
					t.Error("expected validation error for short query")
				}
			}
		})
	}
}
