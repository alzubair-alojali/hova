package ssh

import (
	"reflect"
	"strings"
	"testing"

	"github.com/alzubair-alojali/hova/internal/inventory"
)

func TestPlanPreservesLiteralArguments(t *testing.T) {
	s := inventory.Server{Name: "prod", Host: "203.0.113.10", User: "deploy", Port: 2222, IdentityFile: "C:/my keys/prod"}
	p, err := Plan(s, "printf '%s' 'hello world'", true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=10", "-T", "-p", "2222", "-l", "deploy", "-i", "C:/my keys/prod", "--", "203.0.113.10", "printf '%s' 'hello world'"}
	if p.Executable != "ssh" || !reflect.DeepEqual(p.Args, want) {
		t.Fatalf("plan: %#v", p)
	}
}

func TestImportedAliasUsesSourceConfigWithoutOverrides(t *testing.T) {
	s := inventory.Server{Name: "prod", Host: "203.0.113.10", Port: 22, SSHAlias: "prod-eu", SSHConfig: "/home/me/.ssh/config"}
	p, err := Plan(s, "", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-F", "/home/me/.ssh/config", "--", "prod-eu"}
	if !reflect.DeepEqual(p.Args, want) {
		t.Fatalf("alias changed: %#v", p.Args)
	}
}

func TestPlanRejectsUnsafeTarget(t *testing.T) {
	_, err := Plan(inventory.Server{Name: "prod", Host: "-oProxyCommand=evil", Port: 22}, "", false)
	if err == nil {
		t.Fatal("option injection accepted")
	}
}

func TestImportAliasesWithoutEvaluatingCommands(t *testing.T) {
	config := `Host *
 User root
Host prod other # comment
 HostName 203.0.113.10
 User deploy
 Port 2222
 IdentityFile "~/.ssh/work key"
 ProxyCommand touch /never-run
Host *.internal !blocked
Host staging
 HostName 2001:db8::1
Match exec "touch /never-run"
 User different
`
	servers, err := Import(strings.NewReader(config), "/config")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 3 {
		t.Fatalf("expected concrete aliases, got %#v", servers)
	}
	if servers[0].Name != "prod" || servers[0].Host != "203.0.113.10" || servers[0].Port != 2222 || servers[0].IdentityFile != "~/.ssh/work key" || servers[0].SSHAlias != "prod" || servers[0].SSHConfig != "/config" {
		t.Fatalf("import: %#v", servers[0])
	}
	if servers[2].User != "" {
		t.Fatal("Match block leaked into metadata")
	}
}

func TestImportRejectsMalformedInput(t *testing.T) {
	for _, config := range []string{"Host prod\n Port potato", "Host prod\n HostName -evil", "Host \"unterminated"} {
		if _, err := Import(strings.NewReader(config), "/config"); err == nil {
			t.Errorf("invalid import accepted: %q", config)
		}
	}
}

func TestImportAcceptsEqualsWithWhitespace(t *testing.T) {
	items, err := Import(strings.NewReader("Host = prod\n HostName = 203.0.113.10\n User=deploy\n Port = 2222"), "/config")
	if err != nil || len(items) != 1 {
		t.Fatalf("valid config rejected: %#v %v", items, err)
	}
	if items[0].Name != "prod" || items[0].Host != "203.0.113.10" || items[0].Port != 2222 || items[0].User != "deploy" {
		t.Fatalf("incorrect metadata: %#v", items)
	}
}
