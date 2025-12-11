package config

import "github.com/grantcarthew/start/test/testdata/schemas"

// Invalid: command must not be empty
agents: "bad": schemas.#Agent & {
	command: ""
}
