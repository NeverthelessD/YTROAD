//go:build !windows

package main

import (
	"os/exec"
	"runtime"
	"syscall"
)

func hideWindow(cmd *exec.Cmd) {}

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Process.Kill()
	}
}

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func freeSpace(path string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0
	}
	return int64(st.Bavail) * int64(st.Bsize)
}

func openPath(p string) {
	if runtime.GOOS == "darwin" {
		exec.Command("/usr/bin/open", p).Start()
	}
}

func revealPath(p string) {
	if runtime.GOOS == "darwin" {
		exec.Command("/usr/bin/open", "-R", p).Start()
	}
}

func messageBox(title, msg string) {}

func notifyOS(title, msg string) {
	if runtime.GOOS != "darwin" {
		return
	}
	exec.Command("/usr/bin/osascript", "-e", "on run argv", "-e",
		`display notification (item 2 of argv) with title (item 1 of argv) sound name "Glass"`, "-e", "end run", title, msg).Start()
}
