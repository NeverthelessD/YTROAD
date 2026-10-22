//go:build !windows

package main

// macOS에서는 Dock 앱 껍데기(app/shell/main.swift)가 실행기 역할을 해요.
func launcherMode() bool { return false }
func runLauncher()       {}
