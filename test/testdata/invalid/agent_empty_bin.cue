package config

import "github.com/grantcarthew/start/test/testdata/schemas"

// Invalid: bin must not be empty if provided
agents: "bad": schemas.#Agent & {
	command: "test"
	bin:     ""
}
