//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"unsafe"
)

const (
	createNoWindow        = 0x08000000
	createNewProcessGroup = 0x00000200
	detachedProcess       = 0x00000008
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
	user32            = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW   = user32.NewProc("MessageBoxW")
)

// hideWindow keeps console tools (yt-dlp, ffmpeg, taskkill…) from flashing a black window.
func hideWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}

func setProcessGroup(cmd *exec.Cmd) {
	hideWindow(cmd)
	cmd.SysProcAttr.CreationFlags |= createNewProcessGroup
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		k := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		hideWindow(k)
		k.Run()
		cmd.Process.Kill()
	}
}

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: detachedProcess | createNewProcessGroup}
}

func freeSpace(path string) int64 { return 0 }

func shellExecute(target string) error {
	verb, _ := syscall.UTF16PtrFromString("open")
	t, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	r, _, _ := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(t)), 0, 0, 1)
	if r <= 32 {
		return fmt.Errorf("ShellExecute failed (%d)", r)
	}
	return nil
}

func openPath(p string) { shellExecute(p) }

// revealPath opens Explorer with the file selected.
func revealPath(p string) {
	explorer := filepath.Join(os.Getenv("WINDIR"), "explorer.exe")
	cmd := exec.Command(explorer)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + p + `"`}
	cmd.Start()
}

func messageBox(title, msg string) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(msg)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x30|0x10000) // MB_ICONWARNING | MB_SETFOREGROUND
}

// notifyOS shows a Windows notification (tray balloon → shown as a toast on Windows 10/11).
func notifyOS(title, msg string) {
	script := `Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing;
$n = New-Object System.Windows.Forms.NotifyIcon;
try { $n.Icon = [System.Drawing.Icon]::ExtractAssociatedIcon($env:YTROAD_ICON) } catch { $n.Icon = [System.Drawing.SystemIcons]::Information };
$n.BalloonTipTitle = $env:YTROAD_N_TITLE; $n.BalloonTipText = $env:YTROAD_N_MSG; $n.Visible = $true;
$n.ShowBalloonTip(5000); Start-Sleep -Seconds 6; $n.Dispose()`
	exe, _ := os.Executable()
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-Command", script)
	cmd.Env = append(os.Environ(), "YTROAD_ICON="+exe, "YTROAD_N_TITLE="+title, "YTROAD_N_MSG="+msg)
	hideWindow(cmd)
	cmd.Start()
}
