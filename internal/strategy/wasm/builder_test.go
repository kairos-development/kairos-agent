package wasm

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var (
	buildOnce    sync.Once
	mockWasmPath string
)

func getRealMockWasm(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		mockGoCode := `package main

import "unsafe"

//export on_tick
//go:wasmexport on_tick
func on_tick(ctxPtr uint32, ctxLen uint32) uint64 {
	sig := ` + "`" + `{"action":"buy","symbol":"BTCUSDT","quantity":"1.0","confidence":"0.9"}` + "`" + `
	b := []byte(sig)
	ptr := uint32(uintptr(unsafe.Pointer(&b[0])))
	length := uint32(len(b))
	return (uint64(ptr) << 32) | uint64(length)
}

//export get_parameters
//go:wasmexport get_parameters
func get_parameters() uint64 {
	params := ` + "`" + `{"foo":"bar"}` + "`" + `
	b := []byte(params)
	ptr := uint32(uintptr(unsafe.Pointer(&b[0])))
	length := uint32(len(b))
	return (uint64(ptr) << 32) | uint64(length)
}

//export on_order_filled
//go:wasmexport on_order_filled
func on_order_filled(ptr uint32, length uint32) {}

//export on_order_canceled
//go:wasmexport on_order_canceled
func on_order_canceled(ptr uint32, length uint32) {}

//export set_parameters
//go:wasmexport set_parameters
func set_parameters(ptr uint32, length uint32) {}

//export reset
//go:wasmexport reset
func reset() {}

//export answer
//go:wasmexport answer
func answer() uint32 {
	return 42
}

func main() {
	c := make(chan struct{})
	go func() {
		<-c
	}()
	<-c
}
`
		dir := os.TempDir()
		codePath := filepath.Join(dir, "mock_strategy.go")
		err := os.WriteFile(codePath, []byte(mockGoCode), 0600)
		if err != nil {
			t.Fatalf("failed to write mock go code: %v", err)
		}

		mockWasmPath = filepath.Join(dir, "mock_strategy.wasm")
		cmd := exec.Command("go", "build", "-o", mockWasmPath, codePath)
		cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to compile mock.wasm: %v\nOutput: %s", err, string(out))
		}
	})
	return mockWasmPath
}
