# Publishing CUE Modules to the Central Registry

This guide documents the process of publishing CUE modules to the CUE Central Registry (registry.cue.works), based on our experience publishing the schemas module.

## Prerequisites

### 1. Install CUE

```bash
brew install cue
```

Verify installation:

```bash
cue version
```

### 2. Authenticate with Registry

```bash
cue login
```

This will prompt you to visit a URL and enter a device code. The login credentials are stored in `~/.config/cue/logins.json`.

### 3. GitHub Repository

Your module must be in a GitHub repository. The CUE Central Registry verifies ownership by checking that the repository exists at the module path location.

**Important:** The module path must match the GitHub repository structure:

- ✅ Good: Module at `github.com/user/repo/path@v0` with repo at `github.com/user/repo`
- ❌ Bad: Module at `github.com/user/module-name@v0` without repo at `github.com/user/module-name`

## Module Setup

### 1. Initialize Module

From your module directory:

```bash
cue mod init github.com/user/repo/path@v0 --source=git
```

This creates:

- `cue.mod/` directory
- `cue.mod/module.cue` file

### 2. Verify module.cue

The `module.cue` file must include:

```cue
module: "github.com/user/repo/path@v0"
language: {
    version: "v0.15.1"  // or your CUE version
}
source: {
    kind: "git"
}
```

**Critical:** The `source: {kind: "git"}` field is required for publishing. If missing, you'll get an error.

### 3. Package Structure

Your module should contain CUE files with a package declaration:

```cue
package schemas

#MyDefinition: {
    field: string
}
```

The package name doesn't have to match the module path basename, but it's conventional when they align (allows shorter imports).

## Publishing Workflow

### 1. Ensure Clean VCS State

The repository must be clean before publishing:

```bash
git status
# Should show: "working tree clean"
```

Commit any pending changes:

```bash
git add .
git commit -m "feat: prepare for publishing"
```

### 2. Create Annotated Tag

Use annotated tags (not lightweight tags) for releases:

```bash
git tag -a path/v0.0.1 -m "Release v0.0.1 - initial release"
```

**Tag naming convention:**

- For modules in subdirectories: `subdirectory/v0.0.1`
- For root modules: `v0.0.1`
- Always use semantic versioning: `vMAJOR.MINOR.PATCH`

Example from our schemas:

```bash
git tag -a schemas/v0.0.1 -m "Release schemas v0.0.1 - initial schema definitions"
```

### 3. Push Tag to GitHub

The tag must exist on GitHub before publishing:

```bash
git push origin path/v0.0.1
```

Example:

```bash
git push origin schemas/v0.0.1
```

### 4. Publish to Registry

From your module directory:

```bash
cue mod publish v0.0.1 --verbose
```

**Note:** Use just the version number (not the full tag path).

Successful output:

```
published github.com/user/repo/path@v0.0.1 to registry.cue.works/github.com/user/repo/path:v0.0.1
```

## Using Published Modules

### 1. Import in Other Modules

In your CUE files:

```cue
package mypackage

import "github.com/user/repo/path@v0"

myValue: path.#MyDefinition & {
    field: "value"
}
```

**Import syntax rules:**

- Full form: `import "module/path@v0:packagename"`
- Short form (when basename = package name): `import "module/path@v0"`

Example: If module path is `github.com/user/repo/schemas@v0` and package is `schemas`:

- Short: `import "github.com/user/repo/schemas@v0"` ✅ (recommended)
- Full: `import "github.com/user/repo/schemas@v0:schemas"` ✅ (explicit)

### 2. Fetch Dependencies

From your module directory:

```bash
cue mod tidy
```

This automatically:

- Fetches dependencies from the registry
- Updates `cue.mod/module.cue` with dependency versions
- Downloads modules to the shared cache

### 3. Validate

```bash
cue vet your-file.cue
```

## Common Issues & Solutions

### Issue: "repository must exist before publishing"

**Error:**

```
403 Forbidden: denied: repository github.com/user/module must exist before publishing
```

**Cause:** The module path doesn't match an existing GitHub repository.

**Solution:** Align your module path with your actual repository structure.

Before:

```cue
module: "github.com/user/start-schemas@v0"
// But repo is at github.com/user/start-assets
```

After:

```cue
module: "github.com/user/start-assets/schemas@v0"
// Matches repo structure
```

### Issue: "no source field found"

**Error:**

```
no source field found in cue.mod/module.cue
```

**Solution:** Add the source field:

```cue
source: {
    kind: "git"
}
```

### Issue: "VCS state is not clean"

**Error:**

```
VCS state is not clean
```

**Solution:** Commit all changes before publishing:

```bash
git status
git add .
git commit -m "your message"
```

### Issue: Import fails after publishing

**Error:**

```
cannot find package "github.com/user/repo/path@v0"
```

**Solution:** Ensure tag was pushed to GitHub:

```bash
git push origin path/v0.0.1
```

## Best Practices

### 1. Versioning Strategy

- Start with `v0.0.1` for experimental releases
- Use `v0.x.x` for pre-stable releases
- Increment to `v1.0.0` when stable
- Major version (`@v1`, `@v2`) goes in module path

### 2. Module Organization

Organize modules by purpose in subdirectories:

```
repo/
├── schemas/        → github.com/user/repo/schemas@v0
├── tasks/          → github.com/user/repo/tasks@v0
└── roles/          → github.com/user/repo/roles@v0
```

### 3. Documentation

Include a README.md in each module directory explaining:

- What the module provides
- How to import it
- Example usage
- Schema constraints

### 4. Testing Before Publishing

Always validate locally before publishing:

```bash
# From module directory
cue vet *.cue

# Test imports work
cd ../test-project
cue mod tidy
cue vet .
```

## Real-World Example: Publishing Schemas

Here's the actual workflow we used to publish the schemas module:

```bash
# 1. Navigate to module
cd reference/start-assets/schemas

# 2. Verify module.cue
cat cue.mod/module.cue
# module: "github.com/grantcarthew/start-assets/schemas@v0"
# source: {kind: "git"}

# 3. Ensure clean state
cd ../..
git add .
git commit -m "feat(schemas): add source field for publishing"

# 4. Create annotated tag
git tag -a schemas/v0.0.1 -m "Release schemas v0.0.1 - initial schema definitions"

# 5. Push tag
git push origin schemas/v0.0.1

# 6. Publish
cd reference/start-assets/schemas
cue mod publish v0.0.1 --verbose

# Output:
# published github.com/grantcarthew/start-assets/schemas@v0.0.1 to registry.cue.works/github.com/grantcarthew/start-assets/schemas:v0.0.1
```

## Updating Published Modules

To release a new version:

```bash
# 1. Make changes to CUE files
# 2. Commit changes
git add .
git commit -m "feat: add new schema definition"

# 3. Create new tag
git tag -a path/v0.0.2 -m "Release v0.0.2 - add feature X"

# 4. Push tag
git push origin path/v0.0.2

# 5. Publish new version
cue mod publish v0.0.2 --verbose
```

Users can update to the new version:

```bash
# In their project
cue mod tidy
# Or explicitly:
cue mod get github.com/user/repo/path@v0.0.2
```

## Updating the Index

After publishing a new asset (role, context, task, or agent), you must also update and publish the index module so the asset is discoverable via `start assets`.

```bash
# 1. Add entry to index/index.cue under the appropriate section
# Example for a new role:
"dotai/default": {
    module:      "github.com/grantcarthew/start-assets/roles/dotai/default@v0"
    description: "Project-specific default role from .ai/roles/default.md"
    tags: ["dotai", "project", "default"]
}

# 2. Validate the index
cd start-assets/index
cue vet

# 3. Get current index version
git tag -l 'index/*' | sort -V | tail -1
# Example output: index/v0.1.3

# 4. Commit, tag, push, and publish
cd start-assets
git add index/index.cue
git commit -m "feat(index): add dotai/default role"
git tag -a index/v0.1.4 -m "Release index v0.1.4 - add dotai/default role"
git push origin main
git push origin index/v0.1.4

cd index
cue mod publish v0.1.4 --verbose
```

## References

- [CUE Modules Documentation](https://cuelang.org/docs/concepts/modules-packages-instances/)
- [Publishing Tutorial](https://cuelang.org/docs/tutorial/publishing-modules-to-the-central-registry/)
- [CUE Central Registry](https://registry.cue.works)
