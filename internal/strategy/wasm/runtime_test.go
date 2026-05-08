package wasm

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wasmTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

var minimalStrategyWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
	0x01, 0x0d, 0x03, 0x60, 0x00, 0x01, 0x7f, 0x60, 0x02, 0x7f, 0x7f, 0x00, 0x60, 0x00, 0x00,
	0x03, 0x07, 0x06, 0x00, 0x01, 0x01, 0x01, 0x01, 0x02,
	0x05, 0x03, 0x01, 0x00, 0x01,
	0x07, 0x5c, 0x07,
	0x06, 'm', 'e', 'm', 'o', 'r', 'y', 0x02, 0x00,
	0x06, 'a', 'n', 's', 'w', 'e', 'r', 0x00, 0x00,
	0x07, 'o', 'n', '_', 't', 'i', 'c', 'k', 0x00, 0x01,
	0x0f, 'o', 'n', '_', 'o', 'r', 'd', 'e', 'r', '_', 'f', 'i', 'l', 'l', 'e', 'd', 0x00, 0x02,
	0x11, 'o', 'n', '_', 'o', 'r', 'd', 'e', 'r', '_', 'c', 'a', 'n', 'c', 'e', 'l', 'e', 'd', 0x00, 0x03,
	0x0e, 's', 'e', 't', '_', 'p', 'a', 'r', 'a', 'm', 'e', 't', 'e', 'r', 's', 0x00, 0x04,
	0x05, 'r', 'e', 's', 'e', 't', 0x00, 0x05,
	0x0a, 0x15, 0x06,
	0x04, 0x00, 0x41, 0x2a, 0x0b,
	0x02, 0x00, 0x0b,
	0x02, 0x00, 0x0b,
	0x02, 0x00, 0x0b,
	0x02, 0x00, 0x0b,
	0x02, 0x00, 0x0b,
}

func writeMinimalWasm(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, minimalStrategyWasm, 0o600))
	return path
}

func TestRuntimeLifecycleAndMissingModule(t *testing.T) {
	rt, err := NewRuntime(wasmTestLogger())
	require.NoError(t, err)
	_, err = rt.InstantiateModule(context.Background(), "missing")
	assert.Error(t, err)
	assert.NoError(t, rt.Close(context.Background()))
}

func TestModuleLoaderListAndMissingFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alpha.wasm"), []byte("bad"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "note.txt"), []byte("ignore"), 0o600))
	loader := NewModuleLoader(dir, wasmTestLogger())
	strategies, err := loader.ListAvailableStrategies()
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha"}, strategies)

	rt, err := NewRuntime(wasmTestLogger())
	require.NoError(t, err)
	defer rt.Close(context.Background())
	assert.Error(t, loader.LoadFromFile(context.Background(), rt, "missing"))
	assert.Error(t, loader.LoadFromFile(context.Background(), rt, "alpha"))
}

func TestRuntimeLoadsInstantiatesCallsAndGetsMemory(t *testing.T) {
	dir := t.TempDir()
	wasmPath := writeMinimalWasm(t, dir, "strategy.wasm")

	rt, err := NewRuntime(wasmTestLogger())
	require.NoError(t, err)
	defer rt.Close(context.Background())

	require.NoError(t, rt.LoadModule(context.Background(), "strategy", wasmPath))
	instance, err := rt.InstantiateModule(context.Background(), "strategy")
	require.NoError(t, err)
	defer instance.Close(context.Background())

	results, err := rt.CallFunction(context.Background(), instance, "answer")
	require.NoError(t, err)
	assert.Equal(t, []uint64{42}, results)
	assert.NotNil(t, rt.GetMemory(instance))

	_, err = rt.CallFunction(context.Background(), instance, "missing")
	assert.Error(t, err)
}

func TestValidateAndCopyWasmSuccess(t *testing.T) {
	dir := t.TempDir()
	src := writeMinimalWasm(t, dir, "valid.wasm")
	dest := filepath.Join(dir, "plugins")
	require.NoError(t, os.Mkdir(dest, 0o755))

	assert.NoError(t, ValidateWasmBinary(minimalStrategyWasm, wasmTestLogger()))
	assert.NoError(t, ValidateWasmFile(src, wasmTestLogger()))
	require.NoError(t, CopyWasmFile(src, dest, "copied", wasmTestLogger()))
	assert.FileExists(t, filepath.Join(dest, "copied.wasm"))
}

func TestWasmStrategyAccessorsCallbacksParametersAndClose(t *testing.T) {
	dir := t.TempDir()
	wasmPath := writeMinimalWasm(t, dir, "strategy.wasm")
	rt, err := NewRuntime(wasmTestLogger())
	require.NoError(t, err)
	defer rt.Close(context.Background())
	require.NoError(t, rt.LoadModule(context.Background(), "strategy", wasmPath))
	instance, err := rt.InstantiateModule(context.Background(), "strategy")
	require.NoError(t, err)

	ws := NewWasmStrategy("strategy", "Grid", "1.0.0", instance, rt, nil)
	assert.Equal(t, "Grid", ws.Name())
	assert.Equal(t, "1.0.0", ws.Version())
	assert.NotNil(t, ws.memory)
	assert.NotNil(t, ws.allocator)

	ptr, err := ws.writeToMemory([]byte("hello"))
	require.NoError(t, err)
	body, ok := ws.memory.Read(ptr, 5)
	require.True(t, ok)
	assert.Equal(t, []byte("hello"), body)

	_, err = ws.OnTick(context.Background(), nil)
	require.NoError(t, err)
	assert.NoError(t, ws.OnOrderFilled(context.Background(), "order-1", decimal.NewFromInt(1), decimal.NewFromInt(100)))
	assert.NoError(t, ws.OnOrderCanceled(context.Background(), "order-1"))
	assert.Empty(t, ws.GetParameters())
	assert.NoError(t, ws.SetParameters(map[string]interface{}{"size": "1"}))
	assert.NoError(t, ws.Reset())
	assert.NoError(t, ws.Close(context.Background()))
}

func TestStrategyExecutorLifecycle(t *testing.T) {
	dir := t.TempDir()
	wasmPath := writeMinimalWasm(t, dir, "strategy.wasm")
	rt, err := NewRuntime(wasmTestLogger())
	require.NoError(t, err)
	defer rt.Close(context.Background())
	require.NoError(t, rt.LoadModule(context.Background(), "strategy", wasmPath))

	executor := NewStrategyExecutor(rt, nil)
	_, err = executor.GetStrategy("strategy")
	assert.Error(t, err)

	loaded, err := executor.LoadStrategy(context.Background(), "strategy", "Grid", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "Grid", loaded.Name())
	_, err = executor.LoadStrategy(context.Background(), "strategy", "Grid", "1.0.0")
	assert.Error(t, err)

	listed := executor.ListLoadedStrategies()
	require.Len(t, listed, 1)
	delete(listed, "strategy")
	assert.Len(t, executor.ListLoadedStrategies(), 1)

	_, err = executor.LoadStrategy(context.Background(), "missing", "Missing", "1.0.0")
	assert.Error(t, err)
	assert.NoError(t, executor.UnloadStrategy(context.Background(), "strategy"))
	assert.Error(t, executor.UnloadStrategy(context.Background(), "strategy"))
	assert.NoError(t, executor.UnloadAll(context.Background()))
}

func TestValidateAndCopyWasmFailures(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.wasm")
	assert.Error(t, ValidateWasmFile(missing, wasmTestLogger()))
	assert.Error(t, ValidateWasmBinary([]byte("not wasm"), wasmTestLogger()))
	bad := filepath.Join(dir, "bad.wasm")
	require.NoError(t, os.WriteFile(bad, []byte("not wasm"), 0o600))
	assert.Error(t, CopyWasmFile(bad, filepath.Join(dir, "out"), "bad", wasmTestLogger()))
}

func TestMemoryAllocator(t *testing.T) {
	allocator := newMemoryAllocator()
	offset, err := allocator.allocate(7)
	require.NoError(t, err)
	assert.Equal(t, uint32(65536), offset)
	assert.Equal(t, uint32(65544), allocator.nextOffset)
	_, err = allocator.allocate(allocator.maxSize + 1)
	assert.Error(t, err)
}

func TestNewRuntime_Error(t *testing.T) {
	// NewRuntime should succeed with nil logger
	rt, err := NewRuntime(nil)
	require.NoError(t, err)
	assert.NotNil(t, rt)
	rt.Close(context.Background())
}

func TestLoadModule_InvalidPath(t *testing.T) {
	rt, err := NewRuntime(wasmTestLogger())
	require.NoError(t, err)
	defer rt.Close(context.Background())

	err = rt.LoadModule(context.Background(), "invalid", "/nonexistent/file.wasm")
	assert.Error(t, err)
}

func TestCopyWasmFile_CreateError(t *testing.T) {
	dir := t.TempDir()
	src := writeMinimalWasm(t, dir, "valid.wasm")

	// Try to copy to invalid destination
	err := CopyWasmFile(src, "/nonexistent/dir", "copied", wasmTestLogger())
	assert.Error(t, err)
}

func TestCopyWasmFile_ReadError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "plugins")
	require.NoError(t, os.Mkdir(dest, 0o755))

	// Try to copy non-existent file
	err := CopyWasmFile("/nonexistent/file.wasm", dest, "copied", wasmTestLogger())
	assert.Error(t, err)
}

func TestListAvailableStrategies_ReadDirError(t *testing.T) {
	loader := NewModuleLoader("/nonexistent/dir", wasmTestLogger())
	_, err := loader.ListAvailableStrategies()
	assert.Error(t, err)
}

func TestStrategyExecutor_UnloadAll_WithStrategies(t *testing.T) {
	dir := t.TempDir()
	wasmPath := writeMinimalWasm(t, dir, "strategy.wasm")
	rt, err := NewRuntime(wasmTestLogger())
	require.NoError(t, err)
	defer rt.Close(context.Background())
	require.NoError(t, rt.LoadModule(context.Background(), "strategy", wasmPath))

	executor := NewStrategyExecutor(rt, nil)
	_, err = executor.LoadStrategy(context.Background(), "strategy", "Grid", "1.0.0")
	require.NoError(t, err)

	// UnloadAll should close all strategies
	assert.NoError(t, executor.UnloadAll(context.Background()))
	assert.Empty(t, executor.ListLoadedStrategies())
}
