package main

import "unsafe"

//export on_tick
func on_tick(ctxPtr uint32, ctxLen uint32) uint64 {
	sig := `{"action":"buy","symbol":"BTCUSDT","quantity":"1.0","confidence":"0.9"}`
	b := []byte(sig)
	ptr := uint32(uintptr(unsafe.Pointer(&b[0])))
	length := uint32(len(b))
	return (uint64(ptr) << 32) | uint64(length)
}

//export get_parameters
func get_parameters() uint64 {
	params := `{"foo":"bar"}`
	b := []byte(params)
	ptr := uint32(uintptr(unsafe.Pointer(&b[0])))
	length := uint32(len(b))
	return (uint64(ptr) << 32) | uint64(length)
}

//export on_order_filled
func on_order_filled(ptr uint32, length uint32) {}

//export on_order_canceled
func on_order_canceled(ptr uint32, length uint32) {}

//export set_parameters
func set_parameters(ptr uint32, length uint32) {}

//export reset
func reset() {}

func main() {}
