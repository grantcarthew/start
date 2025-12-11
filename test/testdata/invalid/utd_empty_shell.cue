package config

import "github.com/grantcarthew/start/test/testdata/schemas"

// Invalid: shell must not be empty if provided
roles: "bad": schemas.#Role & {
	prompt: "Test prompt"
	shell:  ""
}
