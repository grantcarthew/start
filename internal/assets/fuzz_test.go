package assets

import (
	"testing"
)

// FuzzFindAssetKey tests FindAssetKey with arbitrary content and keys.
// The function uses a state machine to parse CUE syntax (strings, comments,
// multi-line strings) and should never panic regardless of input.
func FuzzFindAssetKey(f *testing.F) {
	// Seed corpus with representative CUE patterns
	seeds := []struct {
		content  string
		assetKey string
	}{
		{`"my/key": { origin: "test" }`, "my/key"},
		{`simple: { origin: "test" }`, "simple"},
		{"// comment with \"my/key\": ignored\n\"my/key\": {}", "my/key"},
		{"desc: \"has my/key: in it\"\n", "my/key"},
		{"prompt: \"\"\"\n\tmy/key: mentioned\n\t\"\"\"\n", "my/key"},
		{`{ "key": "value { }" }`, "key"},
		{"", "test"},
		{`contexts: {}`, ""},
		{`"escaped \" quote": {}`, "escaped"},
		{"// only a comment\n", "test"},
		{"\"\"\"\ntriple\nquote\n\"\"\"", "test"},
	}

	for _, s := range seeds {
		f.Add(s.content, s.assetKey)
	}

	f.Fuzz(func(t *testing.T, content, assetKey string) {
		// FindAssetKey should never panic
		keyStart, keyLen, err := FindAssetKey(content, assetKey)
		if err != nil {
			return // Not found is fine
		}

		// If found, verify the result is within bounds
		if keyStart < 0 || keyStart >= len(content) {
			t.Errorf("keyStart %d out of bounds for content length %d", keyStart, len(content))
		}
		if keyLen <= 0 {
			t.Errorf("keyLen %d should be positive", keyLen)
		}
		if keyStart+keyLen > len(content) {
			t.Errorf("keyStart(%d)+keyLen(%d) = %d exceeds content length %d",
				keyStart, keyLen, keyStart+keyLen, len(content))
		}
	})
}

// FuzzFindMatchingBrace tests FindMatchingBrace with arbitrary content.
// The function uses a state machine that handles strings, comments, and
// multi-line strings, and should never panic regardless of input.
func FuzzFindMatchingBrace(f *testing.F) {
	// Seed corpus with representative patterns
	seeds := []struct {
		content      string
		openBracePos int
	}{
		{"{ }", 0},
		{"{ { } }", 0},
		{`{ "key": "value { }" }`, 0},
		{"{ // comment { \n }", 0},
		{"{\n\tprompt: \"\"\"\n\t\t{ brace }\n\t\t\"\"\"\n}", 0},
		{`{ "escaped \" { quote" }`, 0},
		{"{ { }", 0}, // unmatched
		{"{\n}", 0},
		{`{ "a": { "b": { "c": {} } } }`, 0},
	}

	for _, s := range seeds {
		f.Add(s.content, s.openBracePos)
	}

	f.Fuzz(func(t *testing.T, content string, openBracePos int) {
		// Skip invalid starting positions
		if openBracePos < 0 || openBracePos >= len(content) {
			return
		}
		if content[openBracePos] != '{' {
			return
		}

		// FindMatchingBrace should never panic
		endPos, err := FindMatchingBrace(content, openBracePos)
		if err != nil {
			return // Unmatched is fine
		}

		// If found, verify the result is within bounds
		if endPos <= openBracePos {
			t.Errorf("endPos %d should be after openBracePos %d", endPos, openBracePos)
		}
		if endPos > len(content) {
			t.Errorf("endPos %d exceeds content length %d", endPos, len(content))
		}
		// The closing brace should be at endPos-1
		if content[endPos-1] != '}' {
			t.Errorf("position before endPos (%d) is %q, want '}'", endPos-1, content[endPos-1])
		}
	})
}

// FuzzFindOpeningBrace tests FindOpeningBrace with arbitrary content.
// The function parses CUE syntax to skip strings and comments, and
// should never panic regardless of input.
func FuzzFindOpeningBrace(f *testing.F) {
	// Seed corpus with representative patterns
	seeds := []struct {
		content  string
		startPos int
	}{
		{"{ }", 0},
		{"   { }", 0},
		{`"{ }" {`, 0},
		{"// { \n {", 0},
		{"no brace here", 0},
		{"\"\"\"\n{ not this }\n\"\"\" {", 0},
		{`"key": {`, 7},
		{"// comment { brace }\n{", 0},
		{`"escaped \" { " {`, 0},
	}

	for _, s := range seeds {
		f.Add(s.content, s.startPos)
	}

	f.Fuzz(func(t *testing.T, content string, startPos int) {
		// Skip invalid starting positions
		if startPos < 0 || startPos >= len(content) {
			return
		}

		// FindOpeningBrace should never panic
		pos, err := FindOpeningBrace(content, startPos)
		if err != nil {
			return // Not found is fine
		}

		// If found, verify the result is within bounds and points to '{'
		if pos < startPos {
			t.Errorf("pos %d should be >= startPos %d", pos, startPos)
		}
		if pos >= len(content) {
			t.Errorf("pos %d exceeds content length %d", pos, len(content))
		}
		if content[pos] != '{' {
			t.Errorf("position %d is %q, want '{'", pos, content[pos])
		}
	})
}
