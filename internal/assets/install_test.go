package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/start/internal/registry"
)

// TestAssetExists tests the AssetExists function.
func TestAssetExists(t *testing.T) {
	t.Parallel()

	// Create a temporary config directory
	configDir := t.TempDir()

	// Write a contexts.cue file with an existing asset
	contextsFile := filepath.Join(configDir, "contexts.cue")
	existingContent := `// start configuration
contexts: {
	"cwd/agents-md": {
		origin: "github.com/test/contexts/cwd/agents-md@v0.1.0"
		description: "Read AGENTS.md file"
		file: "AGENTS.md"
		required: true
		default: true
	}
}
`
	if err := os.WriteFile(contextsFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		name      string
		category  string
		assetName string
		want      bool
	}{
		{
			name:      "existing asset with quotes",
			category:  "contexts",
			assetName: "cwd/agents-md",
			want:      true,
		},
		{
			name:      "non-existent asset",
			category:  "contexts",
			assetName: "cwd/other",
			want:      false,
		},
		{
			name:      "different category file doesn't exist",
			category:  "roles",
			assetName: "cwd/agents-md",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AssetExists(configDir, tt.category, tt.assetName)
			if got != tt.want {
				t.Errorf("AssetExists() = %v, want %v", got, tt.want)
			}
		})
	}
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

// TestGetAssetKey tests the getAssetKey function.
func TestGetAssetKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{"simple", "simple"},
		{"cwd/agents-md", "cwd/agents-md"},
		{"golang/code-review", "golang/code-review"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAssetKey(tt.name)
			if got != tt.want {
				t.Errorf("getAssetKey(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestFindAssetKey tests the FindAssetKey function.
func TestFindAssetKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		assetKey  string
		wantFound bool
		wantPos   int
	}{
		{
			name:      "quoted key",
			content:   `contexts: {\n\t"cwd/agents-md": {`,
			assetKey:  "cwd/agents-md",
			wantFound: true,
		},
		{
			name:      "unquoted key",
			content:   `contexts: {\n\tsimple: {`,
			assetKey:  "simple",
			wantFound: true,
		},
		{
			name: "key in comment ignored",
			content: "// \"cwd/agents-md\": comment\ncontexts: {\n",
			assetKey:  "cwd/agents-md",
			wantFound: false,
		},
		{
			name: "key in string ignored",
			content: "description: \"has cwd/agents-md in it\"\ncontexts: {\n",
			assetKey:  "cwd/agents-md",
			wantFound: false,
		},
		{
			name: "key in multi-line string ignored",
			content: "description: \"\"\"\n\tcwd/agents-md: is mentioned here\n\t\"\"\"\ncontexts: {\n",
			assetKey:  "cwd/agents-md",
			wantFound: false,
		},
		{
			name:      "key not found",
			content:   `contexts: {\n\t"other": {`,
			assetKey:  "cwd/agents-md",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := FindAssetKey(tt.content, tt.assetKey)
			found := err == nil

			if found != tt.wantFound {
				t.Errorf("FindAssetKey() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

// TestFindMatchingBrace tests the FindMatchingBrace function.
func TestFindMatchingBrace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		openBracePos int
		wantEndPos   int
		wantErr      bool
	}{
		{
			name:         "simple nested",
			content:      "{ { } }",
			openBracePos: 0,
			wantEndPos:   7,
			wantErr:      false,
		},
		{
			name:         "with strings",
			content:      `{ "key": "value { }" }`,
			openBracePos: 0,
			wantEndPos:   22,
			wantErr:      false,
		},
		{
			name:         "with comments",
			content:      "{ // comment { \n }",
			openBracePos: 0,
			wantEndPos:   18,
			wantErr:      false,
		},
		{
			name:         "unmatched",
			content:      "{ { }",
			openBracePos: 0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPos, err := FindMatchingBrace(tt.content, tt.openBracePos)

			if tt.wantErr {
				if err == nil {
					t.Error("FindMatchingBrace() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("FindMatchingBrace() unexpected error: %v", err)
				return
			}

			if gotPos != tt.wantEndPos {
				t.Errorf("FindMatchingBrace() = %d, want %d", gotPos, tt.wantEndPos)
			}
		})
	}
}

// TestFindOpeningBrace tests the FindOpeningBrace function.
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
			name:     "immediate brace",
			content:  "{ }",
			startPos: 0,
			wantPos:  0,
			wantErr:  false,
		},
		{
			name:     "skip whitespace",
			content:  "   { }",
			startPos: 0,
			wantPos:  3,
			wantErr:  false,
		},
		{
			name:     "skip string",
			content:  `"{ }" {`,
			startPos: 0,
			wantPos:  6,
			wantErr:  false,
		},
		{
			name:     "skip comment",
			content:  "// { \n {",
			startPos: 0,
			wantPos:  7,
			wantErr:  false,
		},
		{
			name:     "not found",
			content:  "no brace here",
			startPos: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPos, err := FindOpeningBrace(tt.content, tt.startPos)

			if tt.wantErr {
				if err == nil {
					t.Error("FindOpeningBrace() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("FindOpeningBrace() unexpected error: %v", err)
				return
			}

			if gotPos != tt.wantPos {
				t.Errorf("FindOpeningBrace() = %d, want %d", gotPos, tt.wantPos)
			}
		})
	}
}

// TestFindAssetKey_EmptyKey documents Bug: empty assetKey matches any colon.
// When assetKey is "", unquotedKey becomes ":", matching any colon in normal state.
func TestFindAssetKey_EmptyKey(t *testing.T) {
	t.Parallel()

	content := `contexts: {
	"existing": {
		origin: "test"
	}
}`

	_, _, err := FindAssetKey(content, "")
	if err == nil {
		t.Error("FindAssetKey() with empty key should return error, but matched a colon")
	}
}

// TestFindAssetKey_EmptyContent tests FindAssetKey with empty content.
func TestFindAssetKey_EmptyContent(t *testing.T) {
	t.Parallel()

	_, _, err := FindAssetKey("", "my/asset")
	if err == nil {
		t.Error("FindAssetKey() with empty content should return error")
	}
}

// TestFindMatchingBrace_MultiLineString tests FindMatchingBrace with multi-line
// strings containing triple quotes.
func TestFindMatchingBrace_MultiLineString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		openBracePos int
		wantEndPos   int
		wantErr      bool
	}{
		{
			name: "multi-line string with braces inside",
			content: "{\n\tprompt: \"\"\"\n\t\tif (x) { return }\n\t\t\"\"\"\n}",
			openBracePos: 0,
			wantErr:      false,
		},
		{
			name: "multi-line string with nested triple quotes pattern",
			content: "{\n\tprompt: \"\"\"\n\t\tuse {{.field}} here\n\t\t\"\"\"\n}",
			openBracePos: 0,
			wantErr:      false,
		},
		{
			name: "multi-line string at end of content",
			content: "{\n\tprompt: \"\"\"\n\t\thello\n\t\t\"\"\"\n}",
			openBracePos: 0,
			wantErr:      false,
		},
		{
			name:         "unterminated multi-line string",
			content:      "{\n\tprompt: \"\"\"\n\t\thello\n",
			openBracePos: 0,
			wantErr:      true,
		},
		{
			name: "braces in both comments and multi-line strings",
			content: "{\n\t// comment with { brace }\n\tprompt: \"\"\"\n\t\t{ and } in string\n\t\t\"\"\"\n}",
			openBracePos: 0,
			wantErr:      false,
		},
		{
			name: "escaped quotes in single-line string before brace",
			content: `{ "key": "value with \" and { brace }" }`,
			openBracePos: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endPos, err := FindMatchingBrace(tt.content, tt.openBracePos)
			if tt.wantErr {
				if err == nil {
					t.Error("FindMatchingBrace() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("FindMatchingBrace() unexpected error: %v", err)
				return
			}
			if tt.wantEndPos != 0 && endPos != tt.wantEndPos {
				t.Errorf("FindMatchingBrace() = %d, want %d", endPos, tt.wantEndPos)
			}
			// Verify the content up to endPos has balanced braces
			// (the closing brace should be at endPos-1)
			if tt.content[endPos-1] != '}' {
				t.Errorf("position before endPos (%d) is %q, want '}'", endPos-1, tt.content[endPos-1])
			}
		})
	}
}

// TestFindOpeningBrace_MultiLineString tests FindOpeningBrace with multi-line strings.
func TestFindOpeningBrace_MultiLineString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		startPos int
		wantErr  bool
	}{
		{
			name: "brace after multi-line string",
			content: "\"\"\"\n\t{ not this }\n\t\"\"\" {",
			startPos: 0,
			wantErr:  false,
		},
		{
			name: "only braces inside multi-line string",
			content: "\"\"\"\n\t{ not this }\n\t\"\"\"",
			startPos: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, err := FindOpeningBrace(tt.content, tt.startPos)
			if tt.wantErr {
				if err == nil {
					t.Error("FindOpeningBrace() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("FindOpeningBrace() unexpected error: %v", err)
				return
			}
			if tt.content[pos] != '{' {
				t.Errorf("position %d is %q, want '{'", pos, tt.content[pos])
			}
		})
	}
}

// TestWriteAssetToConfig_NewCategory tests adding an asset with a category
// that doesn't exist yet in an existing file.
func TestWriteAssetToConfig_NewCategory(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "tasks.cue")

	existingContent := `// start configuration
contexts: {
	"existing": {
		origin: "test"
	}
}
`
	if err := os.WriteFile(configPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	asset := SearchResult{
		Category: "tasks",
		Name:     "new/task",
		Entry:    registry.IndexEntry{Module: "github.com/test/tasks/new/task@v0"},
	}
	assetContent := `{
	origin: "github.com/test/tasks/new/task@v0.1.0"
	description: "A new task"
}`

	err := writeAssetToConfig(configPath, asset, assetContent, asset.Entry.Module)
	if err != nil {
		t.Fatalf("writeAssetToConfig() error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	result := string(data)
	// Should preserve the existing contexts block
	if !strings.Contains(result, "contexts:") {
		t.Error("result missing existing contexts block")
	}
	if !strings.Contains(result, `"existing":`) {
		t.Error("result missing existing asset")
	}
	// Should have the new tasks block
	if !strings.Contains(result, "tasks: {") {
		t.Error("result missing new tasks category")
	}
	if !strings.Contains(result, `"new/task":`) {
		t.Error("result missing new task asset")
	}
}

// TestUpdateAssetInConfig tests the UpdateAssetInConfig function.
func TestUpdateAssetInConfig(t *testing.T) {
	t.Parallel()

	// Create a temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "contexts.cue")

	// Write initial config
	initialContent := `// start configuration
contexts: {
	"cwd/agents-md": {
		origin: "github.com/test/contexts/cwd/agents-md@v0.1.0"
		description: "Old description"
		file: "AGENTS.md"
	}
}
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	newContent := `{
	origin: "github.com/test/contexts/cwd/agents-md@v0.2.0"
	description: "New description"
	file: "AGENTS.md"
	required: true
}`

	// Update the asset
	err := UpdateAssetInConfig(configPath, "contexts", "cwd/agents-md", newContent)
	if err != nil {
		t.Fatalf("UpdateAssetInConfig() error: %v", err)
	}

	// Read back and verify
	updatedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}

	updatedContent := string(updatedData)

	// Verify new content is present
	if !strings.Contains(updatedContent, "v0.2.0") {
		t.Error("Updated config missing new version")
	}
	if !strings.Contains(updatedContent, "New description") {
		t.Error("Updated config missing new description")
	}
	if !strings.Contains(updatedContent, "required: true") {
		t.Error("Updated config missing new field")
	}

	// Verify old content is gone
	if strings.Contains(updatedContent, "v0.1.0") {
		t.Error("Updated config still contains old version")
	}
	if strings.Contains(updatedContent, "Old description") {
		t.Error("Updated config still contains old description")
	}
}

// TestWriteAssetToConfig tests the writeAssetToConfig function.
func TestWriteAssetToConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name            string
		existingFile    string
		existingContent string
		asset           SearchResult
		assetContent    string
		wantErr         bool
		wantContains    []string
	}{
		{
			name:         "new file",
			existingFile: "",
			asset: SearchResult{
				Category: "contexts",
				Name:     "cwd/agents-md",
				Entry: registry.IndexEntry{
					Module: "github.com/test/contexts/cwd/agents-md@v0",
				},
			},
			assetContent: `{
	origin: "github.com/test/contexts/cwd/agents-md@v0.1.0"
	description: "Test context"
	file: "AGENTS.md"
}`,
			wantErr: false,
			wantContains: []string{
				"contexts: {",
				`"cwd/agents-md":`,
				"origin:",
				"v0.1.0",
			},
		},
		{
			name:         "append to existing file",
			existingFile: "contexts.cue",
			existingContent: `// start configuration
contexts: {
	"other": {
		origin: "test"
	}
}
`,
			asset: SearchResult{
				Category: "contexts",
				Name:     "cwd/agents-md",
				Entry: registry.IndexEntry{
					Module: "github.com/test/contexts/cwd/agents-md@v0",
				},
			},
			assetContent: `{
	origin: "github.com/test/contexts/cwd/agents-md@v0.1.0"
	description: "Test context"
}`,
			wantErr: false,
			wantContains: []string{
				"contexts: {",
				`"other":`,
				`"cwd/agents-md":`,
				"v0.1.0",
			},
		},
		{
			name:         "duplicate asset",
			existingFile: "contexts.cue",
			existingContent: `contexts: {
	"cwd/agents-md": {
		origin: "test"
	}
}
`,
			asset: SearchResult{
				Category: "contexts",
				Name:     "cwd/agents-md",
			},
			assetContent: `{
	origin: "test"
}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			if tt.existingFile != "" {
				configPath = filepath.Join(tempDir, tt.name, tt.existingFile)
				if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				if err := os.WriteFile(configPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to write existing file: %v", err)
				}
			} else {
				configPath = filepath.Join(tempDir, tt.name, assetTypeToConfigFile(tt.asset.Category))
				if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
			}

			err := writeAssetToConfig(configPath, tt.asset, tt.assetContent, tt.asset.Entry.Module)

			if tt.wantErr {
				if err == nil {
					t.Error("writeAssetToConfig() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("writeAssetToConfig() unexpected error: %v", err)
				return
			}

			// Read back and verify
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read config file: %v", err)
			}

			content := string(data)
			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("writeAssetToConfig() result missing %q\nGot:\n%s", want, content)
				}
			}
		})
	}
}

// TestWriteAssetToConfig_BracesInStringValues documents Bug 1:
// writeAssetToConfig uses strings.LastIndex(existingContent, "}") to find
// the closing brace of the category, but this matches the last "}" in the
// entire file content. When a file has multiple top-level categories,
// the last "}" belongs to a DIFFERENT category, causing the new asset
// to be inserted into the wrong category block.
func TestWriteAssetToConfig_BracesInStringValues(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "contexts.cue")

	// File has two top-level categories. The code does:
	//   1. strings.Index(content, "contexts:") -> finds "contexts:" at line 2
	//   2. strings.LastIndex(content, "}") -> finds the LAST "}" which closes "settings:"
	// So the new asset gets inserted at the end of "settings:" instead of "contexts:".
	existingContent := `// start configuration
contexts: {
	"existing": {
		origin: "test"
		description: "An existing context"
	}
}

settings: {
	default_agent: "claude"
}
`
	if err := os.WriteFile(configPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	asset := SearchResult{
		Category: "contexts",
		Name:     "new-asset",
		Entry: registry.IndexEntry{
			Module: "github.com/test/contexts/new-asset@v0",
		},
	}
	assetContent := `{
	origin: "github.com/test/contexts/new-asset@v0.1.0"
	description: "New asset"
}`

	err := writeAssetToConfig(configPath, asset, assetContent, asset.Entry.Module)
	if err != nil {
		t.Fatalf("writeAssetToConfig() error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	result := string(data)

	// The result should contain both the existing and new assets
	if !strings.Contains(result, `"existing":`) {
		t.Error("result missing existing asset")
	}
	if !strings.Contains(result, `"new-asset":`) {
		t.Error("result missing new asset")
	}

	// BUG: The new asset should be inside the contexts: {} block, not
	// the settings: {} block. With strings.LastIndex, it inserts before
	// the last "}" which belongs to settings, not contexts.
	//
	// Verify the new asset appears BEFORE "settings:" - if it doesn't,
	// the bug has been triggered and the asset was put in the wrong block.
	settingsPos := strings.Index(result, "settings:")
	newAssetPos := strings.Index(result, `"new-asset":`)
	if settingsPos == -1 || newAssetPos == -1 {
		t.Fatal("cannot find settings or new-asset in result")
	}
	if newAssetPos > settingsPos {
		t.Errorf("BUG: new asset was inserted into settings block instead of contexts block\n"+
			"new-asset at pos %d, settings at pos %d\nResult:\n%s",
			newAssetPos, settingsPos, result)
	}
}
