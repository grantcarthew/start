package config

import (
	"os"
	"path/filepath"
	"testing"

	internalcue "github.com/grantcarthew/start/internal/cue"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupGlobal    func(dir string) // nil means don't create directory
		setupLocal     func(dir string) // nil means don't create directory
		wantGlobalValid bool
		wantLocalValid  bool
		wantGlobalErr   bool
		wantLocalErr    bool
		wantAnyValid    bool
	}{
		{
			name:           "no directories exist",
			setupGlobal:    nil,
			setupLocal:     nil,
			wantGlobalValid: false,
			wantLocalValid:  false,
			wantAnyValid:    false,
		},
		{
			name: "empty global directory",
			setupGlobal: func(dir string) {
				// Just create the directory, no files
			},
			setupLocal:     nil,
			wantGlobalValid: false,
			wantLocalValid:  false,
			wantAnyValid:    false,
			wantGlobalErr:   false, // Empty dir is not an error
		},
		{
			name: "valid global config",
			setupGlobal: func(dir string) {
				content := `agents: { test: { bin: "test", command: "{role}" } }`
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte(content), 0644)
			},
			setupLocal:     nil,
			wantGlobalValid: true,
			wantLocalValid:  false,
			wantAnyValid:    true,
		},
		{
			name: "invalid global config - syntax error",
			setupGlobal: func(dir string) {
				content := `agents: { test: { bin: "test" command: "{role}" } }` // missing comma
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte(content), 0644)
			},
			setupLocal:     nil,
			wantGlobalValid: false,
			wantLocalValid:  false,
			wantGlobalErr:   true,
			wantAnyValid:    false,
		},
		{
			name: "valid local config",
			setupGlobal: nil,
			setupLocal: func(dir string) {
				content := `roles: { expert: { content: "You are an expert" } }`
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte(content), 0644)
			},
			wantGlobalValid: false,
			wantLocalValid:  true,
			wantAnyValid:    true,
		},
		{
			name: "both valid",
			setupGlobal: func(dir string) {
				content := `agents: { test: { bin: "test", command: "{role}" } }`
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte(content), 0644)
			},
			setupLocal: func(dir string) {
				content := `roles: { expert: { content: "You are an expert" } }`
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte(content), 0644)
			},
			wantGlobalValid: true,
			wantLocalValid:  true,
			wantAnyValid:    true,
		},
		{
			name: "global valid, local invalid",
			setupGlobal: func(dir string) {
				content := `agents: { test: { bin: "test", command: "{role}" } }`
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte(content), 0644)
			},
			setupLocal: func(dir string) {
				content := `roles: { expert: { content: "unclosed string } }`
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte(content), 0644)
			},
			wantGlobalValid: true,
			wantLocalValid:  false,
			wantLocalErr:    true,
			wantAnyValid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directories
			tempDir := t.TempDir()
			globalDir := filepath.Join(tempDir, "global")
			localDir := filepath.Join(tempDir, "local")

			// Setup directories based on test case
			if tt.setupGlobal != nil {
				os.MkdirAll(globalDir, 0755)
				tt.setupGlobal(globalDir)
			}
			if tt.setupLocal != nil {
				os.MkdirAll(localDir, 0755)
				tt.setupLocal(localDir)
			}

			// Create Paths struct
			paths := Paths{
				Global:       globalDir,
				Local:        localDir,
				GlobalExists: tt.setupGlobal != nil,
				LocalExists:  tt.setupLocal != nil,
			}

			// Run validation
			result := ValidateConfig(paths)

			// Check results
			if result.GlobalValid != tt.wantGlobalValid {
				t.Errorf("GlobalValid = %v, want %v", result.GlobalValid, tt.wantGlobalValid)
			}
			if result.LocalValid != tt.wantLocalValid {
				t.Errorf("LocalValid = %v, want %v", result.LocalValid, tt.wantLocalValid)
			}
			if result.AnyValid() != tt.wantAnyValid {
				t.Errorf("AnyValid() = %v, want %v", result.AnyValid(), tt.wantAnyValid)
			}
			if (result.GlobalError != nil) != tt.wantGlobalErr {
				t.Errorf("GlobalError = %v, wantErr %v", result.GlobalError, tt.wantGlobalErr)
			}
			if (result.LocalError != nil) != tt.wantLocalErr {
				t.Errorf("LocalError = %v, wantErr %v", result.LocalError, tt.wantLocalErr)
			}
		})
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	tests := []struct {
		name       string
		result     ValidationResult
		wantErrors bool
	}{
		{
			name:       "no errors",
			result:     ValidationResult{},
			wantErrors: false,
		},
		{
			name:       "global error only",
			result:     ValidationResult{GlobalError: &internalcue.ValidationError{Message: "test"}},
			wantErrors: true,
		},
		{
			name:       "local error only",
			result:     ValidationResult{LocalError: &internalcue.ValidationError{Message: "test"}},
			wantErrors: true,
		},
		{
			name: "both errors",
			result: ValidationResult{
				GlobalError: &internalcue.ValidationError{Message: "global"},
				LocalError:  &internalcue.ValidationError{Message: "local"},
			},
			wantErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasErrors(); got != tt.wantErrors {
				t.Errorf("HasErrors() = %v, want %v", got, tt.wantErrors)
			}
		})
	}
}

func TestHasCUEFiles(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string)
		wantHas  bool
		wantErr  bool
	}{
		{
			name:    "empty directory",
			setup:   func(dir string) {},
			wantHas: false,
		},
		{
			name: "has cue file",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte("{}"), 0644)
			},
			wantHas: true,
		},
		{
			name: "has other files only",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Readme"), 0644)
				os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0644)
			},
			wantHas: false,
		},
		{
			name: "has mixed files",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Readme"), 0644)
				os.WriteFile(filepath.Join(dir, "config.cue"), []byte("{}"), 0644)
			},
			wantHas: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			got, err := hasCUEFiles(dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("hasCUEFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantHas {
				t.Errorf("hasCUEFiles() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}
