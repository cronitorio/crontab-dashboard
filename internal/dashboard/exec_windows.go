//go:build windows
// +build windows

package dashboard

import (
	"os"
	"os/exec"
	"syscall"
)

// getPlatformSysProcAttr returns platform-specific SysProcAttr configuration
func getPlatformSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// Windows doesn't support Setpgid, so we return an empty SysProcAttr
		// Process group functionality is handled differently on Windows
	}
}

// getPlatformSysProcAttrForDash returns platform-specific SysProcAttr configuration for dash command
func getPlatformSysProcAttrForDash() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// Windows doesn't support Setpgid, so we return an empty SysProcAttr
		// Process group functionality is handled differently on Windows
	}
}

func cancelJobProcess(cmd *exec.Cmd) error { return cmd.Process.Kill() }

func restartDashboard(path string, args, env []string) error {
	cmd := exec.Command(path, args...)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
