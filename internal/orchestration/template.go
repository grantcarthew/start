// Package orchestration handles UTD template processing, prompt composition, and agent execution.
package orchestration

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// TemplateData holds the data available for UTD template substitution.
// Uses lowercase keys to match documented placeholder names (e.g., {{.file}}, {{.instructions}}).
type TemplateData map[string]string

// UTDFields represents the raw Unified Template Design (UTD) fields extracted from CUE configuration.
type UTDFields struct {
	// File is the path to read content from.
	File string
	// Command is the shell command to execute.
	Command string
	// Prompt is the template string to render.
	Prompt string
	// Shell is the shell to use for command execution (optional).
	Shell string
	// Timeout is the command timeout in seconds (optional, 0 = default).
	Timeout int
}

// ShellRunner executes shell commands and returns output.
// This interface allows for dependency injection in tests.
type ShellRunner interface {
	// Run executes a command and returns stdout.
	// workingDir specifies the directory to run in.
	// shell specifies the shell to use (e.g., "bash -c").
	// timeout specifies the timeout in seconds (0 = default).
	Run(command, workingDir, shell string, timeout int) (string, error)
}

// FileReader reads file contents.
// This interface allows for dependency injection in tests.
type FileReader interface {
	// Read reads the contents of a file.
	Read(path string) (string, error)
}

// DefaultFileReader implements FileReader using os.ReadFile.
type DefaultFileReader struct{}

// Read reads file contents from the filesystem.
func (r *DefaultFileReader) Read(path string) (string, error) {
	return ReadFilePath(path)
}

// TemplateProcessor handles UTD template resolution.
type TemplateProcessor struct {
	fileReader  FileReader
	shellRunner ShellRunner
	workingDir  string
}

// NewTemplateProcessor creates a new template processor.
func NewTemplateProcessor(fr FileReader, sr ShellRunner, workingDir string) *TemplateProcessor {
	if fr == nil {
		fr = &DefaultFileReader{}
	}
	return &TemplateProcessor{
		fileReader:  fr,
		shellRunner: sr,
		workingDir:  workingDir,
	}
}

// ProcessResult contains the result of template processing.
type ProcessResult struct {
	// Content is the rendered template output.
	Content string
	// TempFile is the path to the temp file created for {{.file}} placeholder.
	// Only set when the source was file-based and temp file was created.
	TempFile string
	// FileRead indicates whether a file was read.
	FileRead bool
	// CommandExecuted indicates whether a command was executed.
	CommandExecuted bool
	// Warnings contains any non-fatal issues encountered.
	Warnings []string
}

// Process resolves a UTD template with lazy evaluation.
// It only reads files or executes commands if the template references them.
func (p *TemplateProcessor) Process(fields UTDFields, instructions string) (ProcessResult, error) {
	var result ProcessResult

	// Determine the template source
	templateStr := fields.Prompt
	if templateStr == "" {
		// If no prompt, check for file content
		if fields.File != "" {
			content, err := p.fileReader.Read(fields.File)
			if err != nil {
				return result, fmt.Errorf("reading file %s: %w", fields.File, err)
			}
			templateStr = content
			result.FileRead = true
		} else if fields.Command != "" {
			// Command output becomes the template
			if p.shellRunner == nil {
				return result, fmt.Errorf("shell runner required for command execution")
			}
			output, err := p.shellRunner.Run(fields.Command, p.workingDir, fields.Shell, fields.Timeout)
			if err != nil {
				return result, fmt.Errorf("executing command: %w", err)
			}
			templateStr = output
			result.CommandExecuted = true
		} else {
			return result, fmt.Errorf("UTD requires at least one of: file, command, or prompt")
		}
	}

	// Check if template uses placeholders that require file/command execution
	// Match documented lowercase/snake_case placeholders
	needsFileContents := strings.Contains(templateStr, "{{.file_contents}}") ||
		strings.Contains(templateStr, "{{ .file_contents }}")
	needsCommandOutput := strings.Contains(templateStr, "{{.command_output}}") ||
		strings.Contains(templateStr, "{{ .command_output }}")

	// Build template data with lowercase keys to match documented placeholders
	data := TemplateData{
		"file":         fields.File,
		"command":      fields.Command,
		"date":         time.Now().Format(time.RFC3339),
		"instructions": instructions,
	}

	// Lazy evaluation: only read file if needed
	if needsFileContents && fields.File != "" && !result.FileRead {
		content, err := p.fileReader.Read(fields.File)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("could not read file %s: %v", fields.File, err))
		} else {
			data["file_contents"] = content
			result.FileRead = true
		}
	}

	// Lazy evaluation: only execute command if needed
	if needsCommandOutput && fields.Command != "" && !result.CommandExecuted {
		if p.shellRunner == nil {
			result.Warnings = append(result.Warnings, "shell runner not available for command execution")
		} else {
			output, err := p.shellRunner.Run(fields.Command, p.workingDir, fields.Shell, fields.Timeout)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("command failed: %v", err))
			} else {
				data["command_output"] = output
				result.CommandExecuted = true
			}
		}
	}

	// Parse and execute template
	// Use Option("missingkey=zero") to handle unknown placeholders gracefully.
	// This allows file-only contexts to contain template-like syntax (e.g., in code examples)
	// without causing errors.
	tmpl, err := template.New("utd").Option("missingkey=zero").Parse(templateStr)
	if err != nil {
		return result, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return result, fmt.Errorf("executing template: %w", err)
	}

	result.Content = buf.String()
	return result, nil
}

// IsUTDValid checks if UTD fields satisfy the minimum requirement.
// At least one of file, command, or prompt must be specified.
func IsUTDValid(fields UTDFields) bool {
	return fields.File != "" || fields.Command != "" || fields.Prompt != ""
}
