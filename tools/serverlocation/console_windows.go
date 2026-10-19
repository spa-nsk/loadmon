//go:build windows

package main

import "syscall"

// Переключает консоль Windows на UTF-8, чтобы русский текст не превращался в "кракозябры".
func init() {
	syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleOutputCP").Call(65001)
}
