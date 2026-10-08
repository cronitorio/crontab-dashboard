package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

func TestPinnedCLIExecutionTelemetryLogsAndExitCode(t *testing.T) {
	path := os.Getenv("CRONTAB_DASHBOARD_TEST_CLI")
	if path == "" {
		t.Skip("run scripts/build-test-cli.sh and set CRONTAB_DASHBOARD_TEST_CLI")
	}
	for _, exit := range []int{0, 7} {
		t.Run(fmt.Sprint(exit), func(t *testing.T) {
			resetConfigureTestState(t)
			var mutex sync.Mutex
			states := map[string]bool{}
			uploaded := false
			server := httptest.NewUnstartedServer(nil)
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutex.Lock()
				defer mutex.Unlock()
				switch r.URL.Path {
				case "/logs/presign":
					json.NewEncoder(w).Encode(map[string]string{"url": server.URL + "/upload"})
				case "/upload":
					data, _ := io.ReadAll(r.Body)
					uploaded = len(data) > 0
					w.WriteHeader(200)
				default:
					states[r.URL.Query().Get("state")] = true
					w.WriteHeader(200)
				}
			})
			server.Start()
			defer server.Close()
			t.Setenv("CRONTAB_DASHBOARD_CRONITOR", path)
			viperKeyForTest(t)
			command := fmt.Sprintf("printf '%%s' 'a quoted \"value\"'; exit %d", exit)
			cmd, err := newJobCommand(context.Background(), "/bin/sh", command, "job-fixture", false)
			if err != nil {
				t.Fatal(err)
			}
			cmd.Env = append(cmd.Env, "DASHBOARD_TEST_API="+server.URL)
			output, err := cmd.CombinedOutput()
			if exit == 0 && err != nil {
				t.Fatalf("run failed: %s %v", output, err)
			}
			if exit != 0 {
				exited, ok := err.(*exec.ExitError)
				if !ok || exited.ExitCode() != exit {
					t.Fatalf("exit: %v %s", err, output)
				}
			}
			if !strings.Contains(string(output), `a quoted "value"`) {
				t.Fatalf("shell quoting changed: %s", output)
			}
			mutex.Lock()
			defer mutex.Unlock()
			end := "complete"
			if exit != 0 {
				end = "fail"
			}
			if !states["run"] || !states[end] || !uploaded {
				t.Fatalf("missing lifecycle/logs: %v uploaded=%v", states, uploaded)
			}
		})
	}
}

func viperKeyForTest(t *testing.T) {
	t.Helper()
	// Configuration set through the UI is passed as environment to the CLI.
	viper.Set(varApiKey, testAPIKey)
}
