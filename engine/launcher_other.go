//go:build !windows

package main

// macOS에서는 bash 실행 스크립트(app/launcher.sh)가 실행기 역할을 해요.
func launcherMode() bool { return false }
func runLauncher()       {}
