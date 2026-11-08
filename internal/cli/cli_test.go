package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func invoke(path string, args ...string) (string, string, int) {
	var out, errOut bytes.Buffer
	code := Execute(append([]string{"--config", path}, args...), &out, &errOut, "test")
	return out.String(), errOut.String(), code
}

func TestConcurrentEditsPreserveOmittedFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.json")
	invoke(path, "add", "prod", "--host", "203.0.113.10")
	start := make(chan struct{})
	var ready, done sync.WaitGroup
	for _, flags := range [][]string{{"--host", "203.0.113.20"}, {"--user", "deploy"}, {"--notes", "new notes"}, {"--tags", "work"}} {
		ready.Add(1)
		done.Add(1)
		go func(flags []string) {
			defer done.Done()
			ready.Done()
			<-start
			_, errOut, code := invoke(path, append([]string{"edit", "prod"}, flags...)...)
			if code != 0 {
				t.Error(errOut)
			}
		}(flags)
	}
	ready.Wait()
	close(start)
	done.Wait()
	out, _, _ := invoke(path, "show", "prod", "--json")
	var s struct {
		Host, User, Notes string
		Tags              []string
	}
	json.Unmarshal([]byte(out), &s)
	if s.Host != "203.0.113.20" || s.User != "deploy" || s.Notes != "new notes" || len(s.Tags) != 1 {
		t.Fatalf("concurrent edit lost fields: %s", out)
	}
}

func TestCLILifecycleAndJSONContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.json")
	out, errOut, code := invoke(path, "list", "--json")
	if code != 0 || strings.TrimSpace(out) != "[]" || errOut != "" {
		t.Fatalf("empty list: %q %q %d", out, errOut, code)
	}
	_, errOut, code = invoke(path, "add", "prod", "--host", "203.0.113.10", "--user", "deploy", "--tags", "work,production", "--json")
	if code != 0 {
		t.Fatal(errOut)
	}
	out, _, code = invoke(path, "show", "prod", "--json")
	var server map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &server) != nil || server["host"] != "203.0.113.10" {
		t.Fatalf("show: %s", out)
	}
	_, errOut, code = invoke(path, "edit", "prod", "--notes", "important", "--tags", "", "--json")
	if code != 0 {
		t.Fatal(errOut)
	}
	out, _, _ = invoke(path, "show", "prod", "--json")
	if strings.Contains(out, `"tags"`) || !strings.Contains(out, "important") {
		t.Fatalf("edit: %s", out)
	}
	_, _, code = invoke(path, "remove", "prod")
	if code == 0 {
		t.Fatal("noninteractive delete did not require --yes")
	}
	_, errOut, code = invoke(path, "remove", "prod", "--yes", "--json")
	if code != 0 {
		t.Fatal(errOut)
	}
	out, _, _ = invoke(path, "list", "--json")
	if strings.TrimSpace(out) != "[]" {
		t.Fatal(out)
	}
}

func TestJSONErrorsStayOffStdout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.json")
	for _, args := range [][]string{{"show", "missing", "--json"}, {"add", "bad", "--host", "-evil", "--json"}, {"list", "--no-such-flag", "--json"}, {"edit", "missing", "--json"}} {
		out, errOut, code := invoke(path, args...)
		var problem map[string]any
		if code == 0 || out != "" || json.Unmarshal([]byte(errOut), &problem) != nil || problem["error"] == nil {
			t.Errorf("invalid failure contract: %q %q %d", out, errOut, code)
		}
	}
}

func TestDryRunNeverConnectsAndExecRequiresExplicitBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.json")
	invoke(path, "add", "prod", "--host", "203.0.113.10")
	out, errOut, code := invoke(path, "connect", "prod", "--dry-run", "--json")
	if code != 0 || !strings.Contains(out, `"executable": "ssh"`) || !strings.Contains(out, "203.0.113.10") {
		t.Fatalf("plan: %s %s", out, errOut)
	}
	for _, args := range [][]string{{"exec", "prod", "uptime"}, {"exec", "prod", "--", "echo", "hello"}, {"exec", "prod", "--", ""}} {
		_, _, code = invoke(path, args...)
		if code == 0 {
			t.Errorf("unsafe exec syntax accepted: %v", args)
		}
	}
}
