# Dependency Review

**Date:** 2026-01-21
**Status:** New

## Summary

Comprehensive review of third-party package usage and dependency management. The project maintains a minimal, well-justified dependency set with no security vulnerabilities detected. All dependencies use permissive licences compatible with the project's MPL-2.0 licence.

## Security Assessment

### Vulnerability Scan

```
govulncheck ./...
No vulnerabilities found.
```

**Status:** All clear. No known CVEs in any direct or transitive dependencies.

## Direct Dependencies

| Package | Version | Licence | Maintenance | Usage |
|---------|---------|---------|-------------|-------|
| cuelang.org/go | v0.15.1 | Apache-2.0 | Active (cue-lang org) | 43 occurrences, 26 files |
| github.com/fatih/color | v1.18.0 | MIT | Active | 2 occurrences, 2 files |
| github.com/spf13/cobra | v1.10.2 | Apache-2.0 | Active (widely adopted) | 22 occurrences, 22 files |
| golang.org/x/term | v0.36.0 | BSD-3-Clause | Active (Go team) | 7 occurrences, 7 files |

### Dependency Analysis

#### 1. cuelang.org/go v0.15.1

**Purpose:** Core CUE language implementation for schema validation, configuration loading, and registry interaction.

**Justification:** Essential - the entire project is built on CUE as stated in AGENTS.md. Cannot be replaced with stdlib.

**Transitives introduced:**
- `cuelabs.dev/go/oci/ociregistry` - OCI registry client for CUE Central Registry
- `github.com/cockroachdb/apd/v3` - Arbitrary precision decimals (CUE number handling)
- `github.com/emicklei/proto` - Protocol buffer parsing
- `github.com/google/uuid` - UUID generation
- `github.com/mitchellh/go-wordwrap` - Word wrapping
- `github.com/opencontainers/go-digest` - OCI content digests
- `github.com/opencontainers/image-spec` - OCI image specification
- `github.com/pelletier/go-toml/v2` - TOML parsing
- `github.com/protocolbuffers/txtpbfmt` - Text protobuf formatting
- `github.com/rogpeppe/go-internal` - Internal testing utilities
- `go.yaml.in/yaml/v3` - YAML parsing
- `golang.org/x/net` - Extended networking
- `golang.org/x/oauth2` - OAuth2 client (registry auth)
- `golang.org/x/sync` - Synchronisation primitives
- `golang.org/x/text` - Text processing
- `google.golang.org/protobuf` - Protocol buffer runtime

**Assessment:** Acceptable transitive tree. CUE requires these for its multi-format support (JSON, YAML, TOML, Protobuf) and registry operations.

#### 2. github.com/fatih/color v1.18.0

**Purpose:** ANSI colour output for terminal display.

**Justification:** Used in `internal/cli/output.go` and `internal/cli/root.go` for coloured CLI output. While stdlib could produce ANSI codes directly, this library handles cross-platform concerns (particularly Windows) and provides a clean API.

**Transitives introduced:**
- `github.com/mattn/go-colorable` - Windows console colour support
- `github.com/mattn/go-isatty` - TTY detection

**Assessment:** Minimal footprint. Standard choice for Go CLI applications.

**Alternatives considered:**
- Raw ANSI codes: Less maintainable, Windows issues
- `github.com/muesli/termenv`: More features but larger dependency tree
- `github.com/charmbracelet/lipgloss`: Over-engineered for simple colour needs

#### 3. github.com/spf13/cobra v1.10.2

**Purpose:** CLI framework for command structure, flags, and help generation.

**Justification:** Industry-standard CLI framework used by Kubernetes, Hugo, GitHub CLI. Provides command composition, flag parsing, shell completion, and help generation. Stdlib `flag` package lacks subcommand support and completion generation.

**Transitives introduced:**
- `github.com/inconshreveable/mousetrap` - Windows process detection
- `github.com/spf13/pflag` - POSIX-compatible flag parsing

**Assessment:** Minimal transitive tree. De facto standard for Go CLI applications.

**Alternatives considered:**
- `github.com/urfave/cli`: Viable alternative, similar feature set
- stdlib `flag`: Insufficient for complex CLI with subcommands
- `github.com/alecthomas/kong`: Less community adoption

#### 4. golang.org/x/term v0.36.0

**Purpose:** Terminal handling functions including password input.

**Justification:** Used for secure password prompting and terminal state management. Part of the official Go extended libraries, maintained by the Go team.

**Transitives introduced:**
- `golang.org/x/sys` - System call wrappers (also a shared dependency)

**Assessment:** Minimal overhead. Official Go extended library.

## Transitive Dependencies

| Package | Introduced By | Licence |
|---------|---------------|---------|
| cuelabs.dev/go/oci/ociregistry | cuelang.org/go | Apache-2.0 |
| github.com/cockroachdb/apd/v3 | cuelang.org/go | Apache-2.0 |
| github.com/emicklei/proto | cuelang.org/go | MIT |
| github.com/google/uuid | cuelang.org/go | BSD-3-Clause |
| github.com/inconshreveable/mousetrap | spf13/cobra | Apache-2.0 |
| github.com/mattn/go-colorable | fatih/color | MIT |
| github.com/mattn/go-isatty | fatih/color | MIT |
| github.com/mitchellh/go-wordwrap | cuelang.org/go | MIT |
| github.com/opencontainers/go-digest | cuelang.org/go | Apache-2.0 |
| github.com/opencontainers/image-spec | cuelang.org/go | Apache-2.0 |
| github.com/pelletier/go-toml/v2 | cuelang.org/go | MIT |
| github.com/protocolbuffers/txtpbfmt | cuelang.org/go | Apache-2.0 |
| github.com/rogpeppe/go-internal | cuelang.org/go | BSD-3-Clause |
| github.com/spf13/pflag | spf13/cobra | BSD-3-Clause |
| go.yaml.in/yaml/v3 | cuelang.org/go | MIT |
| golang.org/x/net | cuelang.org/go | BSD-3-Clause |
| golang.org/x/oauth2 | cuelang.org/go | BSD-3-Clause |
| golang.org/x/sync | cuelang.org/go | BSD-3-Clause |
| golang.org/x/sys | golang.org/x/term | BSD-3-Clause |
| golang.org/x/text | cuelang.org/go | BSD-3-Clause |
| google.golang.org/protobuf | cuelang.org/go | BSD-3-Clause |

**Total:** 4 direct + 21 indirect = 25 dependencies

## Licence Compatibility

**Project Licence:** MPL-2.0 (Mozilla Public License 2.0)

| Dependency Licence | Count | Compatible |
|--------------------|-------|------------|
| Apache-2.0 | 7 | Yes |
| MIT | 7 | Yes |
| BSD-3-Clause | 7 | Yes |

**Assessment:** All dependency licences are permissive and compatible with MPL-2.0. No copyleft licences (GPL, AGPL, LGPL) present that would create compatibility issues.

## go.mod Hygiene

### Verification

- **go mod tidy:** Clean - no changes required
- **go mod verify:** All checksums match
- **go.sum committed:** Yes
- **Replace directives:** None
- **Module path:** Correct (`github.com/grantcarthew/start`)
- **Go version:** 1.24.0 (current stable)

### Version Currency

| Package | Current | Assessment |
|---------|---------|------------|
| cuelang.org/go | v0.15.1 | Latest stable |
| github.com/fatih/color | v1.18.0 | Latest stable |
| github.com/spf13/cobra | v1.10.2 | Latest stable |
| golang.org/x/term | v0.36.0 | Latest stable |

**Assessment:** All direct dependencies are at their latest stable versions.

## Vendoring

**Status:** Not vendored (no `vendor/` directory)

**Configuration:** Default GOPROXY (`https://proxy.golang.org,direct`) with public checksum database (GOSUMDB).

**Assessment:** Appropriate for a public open-source project. Module proxy provides:
- Immutable module versions
- Checksum verification
- Availability guarantee

## Red Flags Check

| Risk Factor | Status |
|-------------|--------|
| Dependencies with no updates in 2+ years | None |
| Single-maintainer projects for critical functionality | None (all have organisations) |
| Dependencies pulling in dozens of transitives | None (CUE's 16 transitives are reasonable) |
| Known unfixed vulnerabilities | None |
| Problematic licences | None |
| Duplicating stdlib functionality | None |

## Recommendations

### No Action Required

The dependency management is well-maintained:

1. **Minimal dependencies:** Only 4 direct dependencies, all justified
2. **No vulnerabilities:** Clean govulncheck scan
3. **Compatible licences:** All permissive, MPL-2.0 compatible
4. **Current versions:** All dependencies at latest stable
5. **Clean go.mod:** No unused dependencies or replace directives

### Maintenance Suggestions

1. **Dependabot/Renovate:** Consider enabling automated dependency updates to stay current with security patches. Example `.github/dependabot.yml`:
   ```yaml
   version: 2
   updates:
     - package-ecosystem: "gomod"
       directory: "/"
       schedule:
         interval: "weekly"
   ```

2. **CI Integration:** The `gl` script already runs `govulncheck` on every lint check. Consider adding it to CI pipeline if not already present.

3. **Version Pinning:** Current approach of pinning to specific versions in go.mod is correct. Continue avoiding version ranges.

## Conclusion

The project demonstrates excellent dependency hygiene. The four direct dependencies are:
- **Essential:** Cannot be replaced by stdlib without significant effort
- **Minimal:** Smallest reasonable footprint for each function
- **Well-maintained:** All backed by organisations or Go team
- **Secure:** No known vulnerabilities
- **Compatible:** All licences work with MPL-2.0

No changes recommended at this time.
