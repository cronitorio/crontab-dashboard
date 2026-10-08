//go:build !windows
// +build !windows

package dashboard

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// getPlatformSysProcAttr returns platform-specific SysProcAttr configuration
func getPlatformSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true, // Put child in its own process group
	}
}

// getPlatformSysProcAttrForDash returns platform-specific SysProcAttr configuration for dash command
func getPlatformSysProcAttrForDash() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true, // Create a new process group for each "run now" command
	}
}

// The CLI places its child in another process group. Stop all descendant
// groups as well as the wrapper, so cancellation does not leave jobs running.
func cancelJobProcess(cmd *exec.Cmd) error {
	root := cmd.Process.Pid
	output, err := exec.Command("ps", "-axo", "pid=,ppid=,pgid=").Output()
	if err != nil {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	type process struct{ pid, parent, group int }
	processes := []process{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		parent, _ := strconv.Atoi(fields[1])
		group, _ := strconv.Atoi(fields[2])
		processes = append(processes, process{pid, parent, group})
	}
	descendants := map[int]bool{root: true}
	for changed := true; changed; {
		changed = false
		for _, p := range processes {
			if descendants[p.parent] && !descendants[p.pid] {
				descendants[p.pid] = true
				changed = true
			}
		}
	}
	ownGroup, _ := syscall.Getpgid(0)
	groups := map[int]bool{}
	for _, p := range processes {
		if descendants[p.pid] && p.group > 1 && p.group != ownGroup {
			groups[p.group] = true
		}
	}
	for group := range groups {
		_ = syscall.Kill(-group, syscall.SIGTERM)
	}
	return cmd.Process.Signal(syscall.SIGTERM)
}

func restartDashboard(path string, args, env []string) error {
	return syscall.Exec(path, append([]string{path}, args...), env)
}
