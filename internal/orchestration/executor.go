package orchestration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"
	"time"

	"cuelang.org/go/cue"
	internalcue "github.com/grantcarthew/start/internal/cue"
)

// Agent represents an agent configuration.
type Agent struct {
	Name         string
	Bin          string
	Command      string
	DefaultModel string
	Models       map[string]string
	Description  string
}

// ExecuteConfig holds the configuration for agent execution.
type ExecuteConfig struct {
	Agent      Agent
	Model      string
	Role       string
	RoleFile   string
	Prompt     string
	PromptFile string
	WorkingDir string
	DryRun     bool
}

// CommandData holds data for command template substitution.
// Uses lowercase keys to match CUE field naming conventions.
type CommandData map[string]string

// Executor handles agent command execution.
type Executor struct {
	workingDir string
}

// NewExecutor creates a new agent executor.
func NewExecutor(workingDir string) *Executor {
	return &Executor{workingDir: workingDir}
}

// BuildCommand builds the agent command from template and config.
func (e *Executor) BuildCommand(cfg ExecuteConfig) (string, error) {
	// Resolve model name to actual model string
	model := cfg.Model
	if model == "" {
		model = cfg.Agent.DefaultModel
	}
	if model != "" {
		if resolved, ok := cfg.Agent.Models[model]; ok {
			model = resolved
		}
	}

	// Build template data with lowercase keys to match CUE conventions
	data := CommandData{
		"bin":       cfg.Agent.Bin,
		"model":     model,
		"role":      escapeForShell(cfg.Role),
		"role_file": cfg.RoleFile,
		"prompt":    escapeForShell(cfg.Prompt),
		"date":      time.Now().Format(time.RFC3339),
	}

	// Parse and execute command template
	tmpl, err := template.New("command").Parse(cfg.Agent.Command)
	if err != nil {
		return "", fmt.Errorf("parsing command template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing command template: %w", err)
	}

	return buf.String(), nil
}

// Execute runs the agent command, replacing the current process.
func (e *Executor) Execute(cfg ExecuteConfig) error {
	cmdStr, err := e.BuildCommand(cfg)
	if err != nil {
		return err
	}

	// Find shell
	shell, err := exec.LookPath("bash")
	if err != nil {
		shell, err = exec.LookPath("sh")
		if err != nil {
			return fmt.Errorf("no shell available")
		}
	}

	// Set working directory
	if cfg.WorkingDir != "" {
		if err := os.Chdir(cfg.WorkingDir); err != nil {
			return fmt.Errorf("changing directory: %w", err)
		}
	}

	// Unix-only: syscall.Exec replaces the current process with the agent.
	// This is intentional - no wrapper overhead, clean process model.
	// Windows is not supported. See DR-006 for platform scope.
	args := []string{shell, "-c", cmdStr}
	env := os.Environ()

	return syscall.Exec(shell, args, env)
}

// ExecuteWithoutReplace runs the agent command without process replacement.
// Useful for testing or when process replacement is not desired.
func (e *Executor) ExecuteWithoutReplace(cfg ExecuteConfig) (string, error) {
	cmdStr, err := e.BuildCommand(cfg)
	if err != nil {
		return "", err
	}

	// Find shell
	shell, err := exec.LookPath("bash")
	if err != nil {
		shell, err = exec.LookPath("sh")
		if err != nil {
			return "", fmt.Errorf("no shell available")
		}
	}

	cmd := exec.Command(shell, "-c", cmdStr)
	if cfg.WorkingDir != "" {
		cmd.Dir = cfg.WorkingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// escapeForShell prepares a string for safe use in shell commands.
// It escapes single quotes for shell safety, preventing shell command injection.
// Note: Environment variables (e.g., $HOME) are NOT expanded. Use literal values
// in prompts or the UTD command field for dynamic content.
func escapeForShell(s string) string {
	// Escape single quotes for shell safety
	return strings.ReplaceAll(s, "'", "'\"'\"'")
}

// ExtractAgent extracts agent configuration from CUE value.
func ExtractAgent(cfg cue.Value, name string) (Agent, error) {
	var agent Agent
	agent.Name = name

	agentVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyAgents)).LookupPath(cue.MakePath(cue.Str(name)))
	if !agentVal.Exists() {
		return agent, fmt.Errorf("agent %q not found", name)
	}

	// Extract required fields
	if bin := agentVal.LookupPath(cue.ParsePath("bin")); bin.Exists() {
		agent.Bin, _ = bin.String()
	}
	if cmd := agentVal.LookupPath(cue.ParsePath("command")); cmd.Exists() {
		agent.Command, _ = cmd.String()
	}

	// Extract optional fields
	if dm := agentVal.LookupPath(cue.ParsePath("default_model")); dm.Exists() {
		agent.DefaultModel, _ = dm.String()
	}
	if desc := agentVal.LookupPath(cue.ParsePath("description")); desc.Exists() {
		agent.Description, _ = desc.String()
	}

	// Extract models map
	if models := agentVal.LookupPath(cue.ParsePath("models")); models.Exists() {
		agent.Models = make(map[string]string)
		iter, err := models.Fields()
		if err == nil {
			for iter.Next() {
				modelName := iter.Selector().Unquoted()
				modelVal := iter.Value()

				// Try direct string first (simple format: models: { sonnet: "model-id" })
				if s, err := modelVal.String(); err == nil {
					agent.Models[modelName] = s
					continue
				}

				// Try nested id field (object format: models: { sonnet: { id: "model-id" } })
				if idVal := modelVal.LookupPath(cue.ParsePath("id")); idVal.Exists() {
					if s, err := idVal.String(); err == nil {
						agent.Models[modelName] = s
					}
				}
			}
		}
	}

	return agent, nil
}

// GetDefaultAgent returns the name of the default agent.
func GetDefaultAgent(cfg cue.Value) string {
	// Check settings.default_agent
	if def := cfg.LookupPath(cue.ParsePath(internalcue.KeySettings + ".default_agent")); def.Exists() {
		if s, err := def.String(); err == nil {
			return s
		}
	}

	// Fall back to first agent in definition order
	agents := cfg.LookupPath(cue.ParsePath(internalcue.KeyAgents))
	if agents.Exists() {
		iter, err := agents.Fields()
		if err == nil && iter.Next() {
			return iter.Selector().Unquoted()
		}
	}

	return ""
}

// GenerateDryRunCommand generates the command.txt content for dry-run.
func GenerateDryRunCommand(agent Agent, model, roleName string, contexts []string, workingDir string, cmdStr string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Agent: %s\n", agent.Name))
	sb.WriteString(fmt.Sprintf("# Model: %s\n", model))
	sb.WriteString(fmt.Sprintf("# Role: %s\n", roleName))
	sb.WriteString(fmt.Sprintf("# Contexts: %s\n", strings.Join(contexts, ", ")))
	sb.WriteString(fmt.Sprintf("# Working Directory: %s\n", workingDir))
	sb.WriteString(fmt.Sprintf("# Generated: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("\n")
	sb.WriteString(cmdStr)

	return sb.String()
}
