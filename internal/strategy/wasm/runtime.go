package wasm

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Runtime manages WASM module execution.
type Runtime struct {
	runtime wazero.Runtime
	logger  *logrus.Logger
	modules map[string]wazero.CompiledModule
}

// NewRuntime creates a new WASM runtime.
func NewRuntime(logger *logrus.Logger) (*Runtime, error) {
	if logger == nil {
		logger = logrus.New()
	}

	// Create wazero runtime with default configuration
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)

	// Instantiate WASI for file system access
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		logger.WithError(err).Error("Failed to instantiate WASI")
		return nil, fmt.Errorf("instantiate WASI: %w", err)
	}

	logger.Info("WASM runtime initialized")

	return &Runtime{
		runtime: runtime,
		logger:  logger,
		modules: make(map[string]wazero.CompiledModule),
	}, nil
}

// LoadModule loads a WASM module from a file.
func (r *Runtime) LoadModule(ctx context.Context, moduleID, wasmPath string) error {
	// Read WASM binary
	wasmBinary, err := os.ReadFile(wasmPath)
	if err != nil {
		r.logger.WithError(err).WithField("path", wasmPath).Error("Failed to read WASM file")
		return fmt.Errorf("read WASM file: %w", err)
	}

	// Compile module
	compiled, err := r.runtime.CompileModule(ctx, wasmBinary)
	if err != nil {
		r.logger.WithError(err).WithField("module_id", moduleID).Error("Failed to compile WASM module")
		return fmt.Errorf("compile module: %w", err)
	}

	if old, exists := r.modules[moduleID]; exists {
		if err := old.Close(ctx); err != nil {
			r.logger.WithError(err).WithField("module_id", moduleID).Warn("Failed to close old compiled module")
		}
	}

	r.modules[moduleID] = compiled
	r.logger.WithField("module_id", moduleID).WithField("size_bytes", len(wasmBinary)).Info("WASM module loaded")

	return nil
}

// InstantiateModule instantiates a compiled WASM module.
func (r *Runtime) InstantiateModule(ctx context.Context, moduleID string) (api.Module, error) {
	compiled, exists := r.modules[moduleID]
	if !exists {
		return nil, fmt.Errorf("module %s not found", moduleID)
	}

	// Instantiate the module
	instance, err := r.runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(moduleID).WithStartFunctions())
	if err != nil {
		r.logger.WithError(err).WithField("module_id", moduleID).Error("Failed to instantiate module")
		return nil, fmt.Errorf("instantiate module: %w", err)
	}

	r.logger.WithField("module_id", moduleID).Info("WASM module instantiated")

	return instance, nil
}

// CallFunction calls an exported function in a WASM module.
func (r *Runtime) CallFunction(ctx context.Context, instance api.Module, functionName string, args ...uint64) ([]uint64, error) {
	function := instance.ExportedFunction(functionName)
	if function == nil {
		return nil, fmt.Errorf("function %s not exported", functionName)
	}

	results, err := function.Call(ctx, args...)
	if err != nil {
		r.logger.WithError(err).WithField("function", functionName).Error("Failed to call function")
		return nil, fmt.Errorf("call function: %w", err)
	}

	return results, nil
}

// GetMemory returns the memory of a module instance.
func (r *Runtime) GetMemory(instance api.Module) api.Memory {
	return instance.Memory()
}

// Close closes the runtime and releases resources.
func (r *Runtime) Close(ctx context.Context) error {
	r.logger.Info("Closing WASM runtime")

	if err := r.runtime.Close(ctx); err != nil {
		r.logger.WithError(err).Error("Failed to close runtime")
		return fmt.Errorf("close runtime: %w", err)
	}

	r.logger.Info("WASM runtime closed")
	return nil
}

// ModuleLoader handles loading WASM modules from the filesystem.
type ModuleLoader struct {
	pluginDir string
	logger    *logrus.Logger
}

// NewModuleLoader creates a new module loader.
func NewModuleLoader(pluginDir string, logger *logrus.Logger) *ModuleLoader {
	if logger == nil {
		logger = logrus.New()
	}

	return &ModuleLoader{
		pluginDir: pluginDir,
		logger:    logger,
	}
}

// LoadFromFile loads a WASM module from a file in the plugin directory.
func (ml *ModuleLoader) LoadFromFile(ctx context.Context, runtime *Runtime, strategyName string) error {
	cleanName := filepath.Base(strategyName)
	wasmPath := filepath.Join(ml.pluginDir, cleanName+".wasm")

	// Check if file exists
	if _, err := os.Stat(wasmPath); err != nil {
		ml.logger.WithError(err).WithField("path", wasmPath).Error("WASM file not found")
		return fmt.Errorf("WASM file not found: %w", err)
	}

	// Load into runtime
	if err := runtime.LoadModule(ctx, strategyName, wasmPath); err != nil {
		return err
	}

	ml.logger.WithField("strategy", strategyName).WithField("path", wasmPath).Info("Strategy loaded from file")
	return nil
}

// ListAvailableStrategies lists all available WASM strategy files.
func (ml *ModuleLoader) ListAvailableStrategies() ([]string, error) {
	entries, err := os.ReadDir(ml.pluginDir)
	if err != nil {
		ml.logger.WithError(err).WithField("dir", ml.pluginDir).Error("Failed to read plugin directory")
		return nil, fmt.Errorf("read plugin directory: %w", err)
	}

	var strategies []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".wasm" {
			name := entry.Name()[:len(entry.Name())-5] // Remove .wasm extension
			strategies = append(strategies, name)
		}
	}

	ml.logger.WithField("count", len(strategies)).Info("Available strategies listed")
	return strategies, nil
}

// ValidateWasmBinary validates a WASM binary without loading it.
func ValidateWasmBinary(wasmBinary []byte, logger *logrus.Logger) error {
	if logger == nil {
		logger = logrus.New()
	}

	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	_, err := runtime.CompileModule(ctx, wasmBinary)
	if err != nil {
		logger.WithError(err).Error("WASM binary validation failed")
		return fmt.Errorf("validate WASM binary: %w", err)
	}

	logger.Info("WASM binary validation successful")
	return nil
}

// ValidateWasmFile validates a WASM file.
func ValidateWasmFile(wasmPath string, logger *logrus.Logger) error {
	if logger == nil {
		logger = logrus.New()
	}

	wasmBinary, err := os.ReadFile(wasmPath)
	if err != nil {
		logger.WithError(err).WithField("path", wasmPath).Error("Failed to read WASM file for validation")
		return fmt.Errorf("read WASM file: %w", err)
	}

	return ValidateWasmBinary(wasmBinary, logger)
}

// CopyWasmFile copies a WASM file to the plugin directory.
func CopyWasmFile(src, destDir, strategyName string, logger *logrus.Logger) error {
	if logger == nil {
		logger = logrus.New()
	}

	// Validate source file
	if err := ValidateWasmFile(src, logger); err != nil {
		return err
	}

	// Read source
	srcFile, err := os.Open(src)
	if err != nil {
		logger.WithError(err).WithField("src", src).Error("Failed to open source WASM file")
		return fmt.Errorf("open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination
	cleanName := filepath.Base(strategyName)
	destPath := filepath.Join(destDir, cleanName+".wasm")
	destFile, err := os.Create(destPath)
	if err != nil {
		logger.WithError(err).WithField("dest", destPath).Error("Failed to create destination WASM file")
		return fmt.Errorf("create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy
	if _, err := io.Copy(destFile, srcFile); err != nil {
		logger.WithError(err).Error("Failed to copy WASM file")
		return fmt.Errorf("copy file: %w", err)
	}

	logger.WithField("src", src).WithField("dest", destPath).Info("WASM file copied successfully")
	return nil
}
