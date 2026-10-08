package dashboard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Prefer the release's companion over an unrelated CLI on PATH. Source builds
// can explicitly select a CLI with CRONTAB_DASHBOARD_CRONITOR.
func companionPath() (string, error) {
	if path := os.Getenv("CRONTAB_DASHBOARD_CRONITOR"); path != "" {
		return exec.LookPath(path)
	}
	if executable, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(executable); err == nil {
			executable = real
		}
		path := filepath.Join(filepath.Dir(executable), "cronitor")
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return path, nil
		}
	}
	path, err := exec.LookPath("cronitor")
	if err != nil {
		return "", fmt.Errorf("Cronitor CLI is required for monitored jobs; install the dashboard release bundle or set CRONTAB_DASHBOARD_CRONITOR")
	}
	return path, nil
}

func jobEnvironment() []string {
	env := []string{"SHELL=/bin/sh"}
	for _, name := range []string{"HOME", "PATH", "TZ", "CRON_TZ"} {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}

func newJobCommand(ctx context.Context, shell, command, code string, noStdout bool) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	if code == "" {
		cmd = exec.CommandContext(ctx, shell, "-c", command)
		cmd.Env = jobEnvironment()
	} else {
		path, err := companionPath()
		if err != nil {
			return nil, err
		}
		args := []string{"exec"}
		if noStdout {
			args = append(args, "--no-stdout")
		}
		args = append(args, code, shell, "-c", command)
		cmd = exec.CommandContext(ctx, path, args...)
		cmd.Env = jobEnvironment()
		// Forward effective settings, including values set through flags or the
		// settings UI. Credentials never enter argv or the output stream.
		for _, name := range []string{varApiKey, varPingApiKey, varPingApiHost, varEnv, varHostname, varApiVersion} {
			if value := viper.GetString(name); value != "" {
				cmd.Env = append(cmd.Env, name+"="+value)
			}
		}
		config, err := filepath.Abs(configFilePath())
		if err != nil {
			return nil, err
		}
		cmd.Env = append(cmd.Env, varConfig+"="+config)
		// The CLI handles SIGTERM and sends its completion/failure telemetry.
		cmd.Cancel = func() error { return cancelJobProcess(cmd) }
		cmd.WaitDelay = 15 * time.Second
	}
	cmd.SysProcAttr = getPlatformSysProcAttrForDash()
	if code == "" {
		cmd.Cancel = func() error { return cancelJobProcess(cmd) }
	}
	return cmd, nil
}
