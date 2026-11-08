package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestRemoteExecutionResultAndTimeout(t *testing.T) {
	dir := t.TempDir()
	exe := "ssh"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", filepath.Join(dir, exe), "testdata/ssh.go")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build subprocess fixture: %v %s", err, output)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	childFile := filepath.Join(dir, "child.pid")
	t.Setenv("HOVA_TEST_CHILD_PID", childFile)
	t.Cleanup(func() {
		data, err := os.ReadFile(childFile)
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(string(data))
		if err != nil {
			t.Error(err)
			return
		}
		child, err := os.FindProcess(pid)
		if err != nil {
			return
		}
		child.Kill()
		child.Wait()
		child.Release()
	})
	path := filepath.Join(t.TempDir(), "servers.json")
	invoke(path, "add", "prod", "--host", "203.0.113.10")
	for _, tc := range []struct {
		command, timeout string
		code             int
		timedOut         bool
	}{{"fixture-fail", "5s", 7, false}, {"fixture-sleep", "100ms", 124, true}, {"fixture-child", "100ms", 124, true}, {"fixture-success", "5s", 0, false}} {
		started := time.Now()
		out, errOut, code := invoke(path, "exec", "prod", "--json", "--timeout", tc.timeout, "--", tc.command)
		var result struct {
			Stdout, Stderr string
			ExitCode       int  `json:"exit_code"`
			TimedOut       bool `json:"timed_out"`
		}
		if json.Unmarshal([]byte(out), &result) != nil || code != tc.code || result.ExitCode != tc.code || result.TimedOut != tc.timedOut || errOut != "" {
			t.Errorf("%s: %q %q %d", tc.command, out, errOut, code)
		}
		if tc.code == 7 && (result.Stdout != "partial output\n" || result.Stderr != "permission denied\n") {
			t.Errorf("output lost: %#v", result)
		}
		if tc.timedOut && time.Since(started) > 2*time.Second {
			t.Errorf("timeout waited for inherited pipes: %v", time.Since(started))
		}
	}
}
