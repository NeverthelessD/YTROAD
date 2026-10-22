//go:build windows

package main

import "os"

// Windows 에서는 YTROAD.exe 의 실행기(launcher_windows.go)가 엔진을 켜요. 지킴이는 맥(Dock 앱)에서만 써요.
func runSupervisor(support string) {}

func env(k string) string { return os.Getenv("YTROAD_" + k) }
