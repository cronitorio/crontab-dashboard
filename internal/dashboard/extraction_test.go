package dashboard

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestRunDrainsOutputLargerThanOneChunk(t *testing.T) {
	t.Setenv("CRONTAB_DASHBOARD_CRONITOR", "/nonexistent/cronitor")
	// Each burst exceeds one stream chunk. Reading must not move the shared
	// output descriptor's offset or discard the final burst after process exit.
	command := `i=0; while [ "$i" -lt 20000 ]; do printf a; i=$((i+1)); done; sleep 0.3; i=0; while [ "$i" -lt 20000 ]; do printf b; i=$((i+1)); done`
	payload, _ := json.Marshal(map[string]string{"command": command})
	response := httptest.NewRecorder()
	handleRunJob(response, httptest.NewRequest("POST", "/api/jobs/run", strings.NewReader(string(payload))))
	var output strings.Builder
	for _, line := range strings.Split(response.Body.String(), "\n") {
		if strings.HasPrefix(line, "data: ") {
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
				t.Fatal(err)
			}
			if chunk, ok := event["output"].(string); ok {
				output.WriteString(chunk)
			}
		}
	}
	if output.String() != strings.Repeat("a", 20000)+strings.Repeat("b", 20000) {
		t.Fatalf("stream lost or overwrote output: received %d bytes", output.Len())
	}
}

func TestUnmonitoredRunNeedsNoCLIAndStreamsBeforeCompletion(t *testing.T) {
	t.Setenv("CRONTAB_DASHBOARD_CRONITOR", "/nonexistent/cronitor")
	request := httptest.NewRequest("POST", "/api/jobs/run", strings.NewReader(`{"command":"printf hello; exit 7"}`))
	response := httptest.NewRecorder()
	handleRunJob(response, request)
	body := response.Body.String()
	if !strings.Contains(body, "hello") || !strings.Contains(body, "Exit code 7") {
		t.Fatalf("response: %s", body)
	}
	if strings.Index(body, "hello") > strings.Index(body, "completion") {
		t.Fatal("completion preceded final output")
	}
}

func TestMonitoringDelegatesArgumentsAndEffectiveSettings(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	resetConfigureTestState(t)
	path := filepath.Join(t.TempDir(), "cronitor")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CRONTAB_DASHBOARD_CRONITOR", path)
	t.Setenv(varDashPassword, "fake-dashboard-password-not-real")
	viper.Set(varApiKey, testAPIKey)
	viper.Set(varEnv, "staging")
	viper.Set(varConfig, filepath.Join(t.TempDir(), "custom config.json"))
	command := `printf '%s' 'a "quoted" value'; exit 9`
	cmd, err := newJobCommand(context.Background(), "/bin/sh", command, "myjob", true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cmd.Args, []string{path, "exec", "--no-stdout", "myjob", "/bin/sh", "-c", command}) {
		t.Fatalf("args: %q", cmd.Args)
	}
	if strings.Contains(strings.Join(cmd.Args, " "), testAPIKey) {
		t.Fatal("key leaked into argv")
	}
	env := strings.Join(cmd.Env, "\n")
	if !strings.Contains(env, varApiKey+"="+testAPIKey) || !strings.Contains(env, varEnv+"=staging") {
		t.Fatal("effective settings not forwarded")
	}
	if strings.Contains(env, "fake-dashboard-password-not-real") {
		t.Fatal("dashboard password forwarded to job")
	}
}

func TestMonitoredRunReportsMissingCLI(t *testing.T) {
	t.Setenv("CRONTAB_DASHBOARD_CRONITOR", "/nonexistent/cronitor")
	if _, err := newJobCommand(context.Background(), "/bin/sh", "true", "myjob", false); err == nil {
		t.Fatal("missing companion accepted")
	}
}

func TestCancelStopsDescendantProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix process groups")
	}
	path := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd, err := newJobCommand(ctx, "/bin/sh", "sleep 60 & echo $! > '"+path+"'; wait", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var pid string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			pid = strings.TrimSpace(string(data))
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == "" {
		cancel()
		cmd.Wait()
		t.Fatal("child did not start")
	}
	t.Cleanup(func() { exec.Command("kill", "-9", pid).Run() })
	cancel()
	if err := cmd.Wait(); err == nil {
		t.Fatal("cancelled command succeeded")
	}
	for time.Now().Before(deadline) {
		if err := exec.Command("kill", "-0", pid).Run(); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("child %s survived cancellation", pid)
}

func TestPatchVersionUpdates(t *testing.T) {
	if !isNewer("0.1.1", "0.1.0") || isNewer("0.1.0", "0.1.1") || isNewer("v0.1.0", "0.1.0") {
		t.Fatal("patch version comparison failed")
	}
}
