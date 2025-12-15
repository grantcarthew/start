package orchestration

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"

	"github.com/grantcarthew/start/internal/config"
	"github.com/grantcarthew/start/internal/detection"
	"github.com/grantcarthew/start/internal/registry"
)

// AutoSetupResult contains the result of auto-setup.
type AutoSetupResult struct {
	Agent      Agent
	ConfigPath string
}

// AutoSetup performs first-run auto-setup.
// It detects installed AI CLI tools, prompts if needed, and writes config.
type AutoSetup struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
	isTTY  bool
}

// NewAutoSetup creates a new auto-setup handler.
func NewAutoSetup(stdout, stderr io.Writer, stdin io.Reader, isTTY bool) *AutoSetup {
	return &AutoSetup{
		stdout: stdout,
		stderr: stderr,
		stdin:  stdin,
		isTTY:  isTTY,
	}
}

// NeedsSetup checks if auto-setup is required.
func NeedsSetup(paths config.Paths) bool {
	return !paths.AnyExists()
}

// Run executes the auto-setup flow.
func (a *AutoSetup) Run(ctx context.Context) (*AutoSetupResult, error) {
	// Create registry client
	client, err := registry.NewClient()
	if err != nil {
		return nil, fmt.Errorf("creating registry client: %w", err)
	}

	// Fetch index
	fmt.Fprintln(a.stdout, "Fetching agent index...")
	index, err := client.FetchIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}

	// Detect installed agents
	detected := detection.DetectAgents(index)
	if len(detected) == 0 {
		return nil, a.noAgentsError(index)
	}

	// Sort for consistent ordering
	sort.Slice(detected, func(i, j int) bool {
		return detected[i].Key < detected[j].Key
	})

	// Select agent
	var selected detection.DetectedAgent
	if len(detected) == 1 {
		selected = detected[0]
		fmt.Fprintf(a.stdout, "Detected: %s\n", selected.Entry.Bin)
	} else {
		selected, err = a.promptSelection(detected)
		if err != nil {
			return nil, err
		}
	}

	// Resolve to canonical version and fetch agent module
	fmt.Fprintln(a.stdout, "Fetching configuration...")
	resolvedPath, err := client.ResolveLatestVersion(ctx, selected.Entry.Module)
	if err != nil {
		return nil, fmt.Errorf("resolving agent version: %w", err)
	}

	agentResult, err := client.Fetch(ctx, resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("fetching agent module: %w", err)
	}

	// Load agent from fetched module
	agent, err := loadAgentFromModule(agentResult.SourceDir, selected.Key)
	if err != nil {
		return nil, fmt.Errorf("loading agent: %w", err)
	}

	// Write config
	configPath, err := a.writeConfig(agent)
	if err != nil {
		return nil, fmt.Errorf("writing config: %w", err)
	}

	fmt.Fprintf(a.stdout, "Configuration saved to %s\n", configPath)

	return &AutoSetupResult{
		Agent:      agent,
		ConfigPath: configPath,
	}, nil
}

// noAgentsError returns a helpful error when no agents are detected.
func (a *AutoSetup) noAgentsError(index *registry.Index) error {
	var sb strings.Builder
	sb.WriteString("No AI CLI tools detected in PATH.\n\n")
	sb.WriteString("Install one of:\n")

	// List available agents from index
	var agents []struct {
		bin  string
		desc string
	}
	for _, entry := range index.Agents {
		if entry.Bin != "" {
			agents = append(agents, struct {
				bin  string
				desc string
			}{entry.Bin, entry.Description})
		}
	}
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].bin < agents[j].bin
	})

	for _, ag := range agents {
		if ag.desc != "" {
			sb.WriteString(fmt.Sprintf("  %s - %s\n", ag.bin, ag.desc))
		} else {
			sb.WriteString(fmt.Sprintf("  %s\n", ag.bin))
		}
	}

	sb.WriteString("\nThen run 'start' again.")
	return fmt.Errorf("%s", sb.String())
}

// promptSelection prompts the user to select an agent.
func (a *AutoSetup) promptSelection(detected []detection.DetectedAgent) (detection.DetectedAgent, error) {
	if !a.isTTY {
		var names []string
		for _, d := range detected {
			names = append(names, d.Entry.Bin)
		}
		return detection.DetectedAgent{}, fmt.Errorf(
			"multiple AI CLI tools detected: %s\nRun interactively to select, or set default_agent in config",
			strings.Join(names, ", "),
		)
	}

	fmt.Fprintln(a.stdout, "Multiple AI CLI tools detected:")
	fmt.Fprintln(a.stdout)

	for i, d := range detected {
		if d.Entry.Description != "" {
			fmt.Fprintf(a.stdout, "  %d. %s - %s\n", i+1, d.Entry.Bin, d.Entry.Description)
		} else {
			fmt.Fprintf(a.stdout, "  %d. %s\n", i+1, d.Entry.Bin)
		}
	}

	fmt.Fprintln(a.stdout)
	fmt.Fprintf(a.stdout, "Select agent [1-%d]: ", len(detected))

	reader := bufio.NewReader(a.stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return detection.DetectedAgent{}, fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(detected) {
		return detection.DetectedAgent{}, fmt.Errorf("invalid selection: %s", input)
	}

	return detected[choice-1], nil
}

// loadAgentFromModule loads an agent from a fetched module directory.
func loadAgentFromModule(dir, key string) (Agent, error) {
	cctx := cuecontext.New()

	cfg := &load.Config{
		Dir:     dir,
		Package: "*",
	}

	insts := load.Instances([]string{"."}, cfg)
	if len(insts) == 0 {
		return Agent{}, fmt.Errorf("no CUE instances found in %s", dir)
	}

	inst := insts[0]
	if inst.Err != nil {
		return Agent{}, fmt.Errorf("loading module: %w", inst.Err)
	}

	v := cctx.BuildInstance(inst)
	if err := v.Err(); err != nil {
		return Agent{}, fmt.Errorf("building module: %w", err)
	}

	// Extract agent name from key (e.g., "ai/claude" -> "claude")
	name := key
	if idx := strings.LastIndex(key, "/"); idx != -1 {
		name = key[idx+1:]
	}

	return extractAgentFromValue(v, name)
}

// extractAgentFromValue extracts agent config from a CUE value.
func extractAgentFromValue(v cue.Value, name string) (Agent, error) {
	var agent Agent
	agent.Name = name

	// Try looking up under "agents" first
	agentVal := v.LookupPath(cue.ParsePath("agents")).LookupPath(cue.MakePath(cue.Str(name)))
	if !agentVal.Exists() {
		// Try root level (module might define agent directly)
		agentVal = v
	}

	// Extract fields
	if bin := agentVal.LookupPath(cue.ParsePath("bin")); bin.Exists() {
		agent.Bin, _ = bin.String()
	}
	if cmd := agentVal.LookupPath(cue.ParsePath("command")); cmd.Exists() {
		agent.Command, _ = cmd.String()
	}
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

				if s, err := modelVal.String(); err == nil {
					agent.Models[modelName] = s
					continue
				}

				if idVal := modelVal.LookupPath(cue.ParsePath("id")); idVal.Exists() {
					if s, err := idVal.String(); err == nil {
						agent.Models[modelName] = s
					}
				}
			}
		}
	}

	if agent.Bin == "" {
		return agent, fmt.Errorf("agent %s missing required 'bin' field", name)
	}
	if agent.Command == "" {
		return agent, fmt.Errorf("agent %s missing required 'command' field", name)
	}

	return agent, nil
}

// writeConfig writes the agent configuration to the global config directory.
func (a *AutoSetup) writeConfig(agent Agent) (string, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return "", err
	}

	// Create config directory
	if err := os.MkdirAll(paths.Global, 0755); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}

	// Generate CUE content
	content := generateAgentCUE(agent)

	// Write to agents.cue
	configPath := filepath.Join(paths.Global, "agents.cue")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing config file: %w", err)
	}

	return configPath, nil
}

// generateAgentCUE generates CUE content for an agent.
func generateAgentCUE(agent Agent) string {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by start auto-setup\n")
	sb.WriteString("// Edit this file to customize your agent configuration\n\n")
	sb.WriteString("agents: {\n")
	sb.WriteString(fmt.Sprintf("\t%q: {\n", agent.Name))
	sb.WriteString(fmt.Sprintf("\t\tbin:     %q\n", agent.Bin))
	sb.WriteString(fmt.Sprintf("\t\tcommand: %q\n", agent.Command))

	if agent.DefaultModel != "" {
		sb.WriteString(fmt.Sprintf("\t\tdefault_model: %q\n", agent.DefaultModel))
	}

	if agent.Description != "" {
		sb.WriteString(fmt.Sprintf("\t\tdescription: %q\n", agent.Description))
	}

	if len(agent.Models) > 0 {
		sb.WriteString("\t\tmodels: {\n")

		// Sort model names for consistent output
		var modelNames []string
		for name := range agent.Models {
			modelNames = append(modelNames, name)
		}
		sort.Strings(modelNames)

		for _, name := range modelNames {
			sb.WriteString(fmt.Sprintf("\t\t\t%s: %q\n", name, agent.Models[name]))
		}
		sb.WriteString("\t\t}\n")
	}

	sb.WriteString("\t}\n")
	sb.WriteString("}\n")
	sb.WriteString("\n")
	sb.WriteString("settings: {\n")
	sb.WriteString(fmt.Sprintf("\tdefault_agent: %q\n", agent.Name))
	sb.WriteString("}\n")

	return sb.String()
}
