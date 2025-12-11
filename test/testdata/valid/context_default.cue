package config

import "github.com/grantcarthew/start/test/testdata/schemas"

// Default context - included in plain `start`, not with --context
contexts: "project": schemas.#Context & {
	default:     true
	description: "Current project documentation"
	file:        "./PROJECT.md"
	prompt:      "Project context:\n{{.file_contents}}"
}
