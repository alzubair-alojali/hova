package inventory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestInventoryLifecycle(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "private", "servers.json")}
	servers, err := store.Load()
	if err != nil || len(servers) != 0 {
		t.Fatalf("missing inventory: %v %v", servers, err)
	}
	err = store.Update(func(items []Server) ([]Server, error) {
		return append(items, Server{Name: "prod", Host: "203.0.113.10", User: "deploy", Port: 22, Tags: []string{"work"}}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	servers, err = store.Load()
	if err != nil || len(servers) != 1 || servers[0].Host != "203.0.113.10" {
		t.Fatalf("roundtrip: %v %v", servers, err)
	}
	if len(Filter(servers, "PROD", "work")) != 1 || len(Filter(servers, "missing", "")) != 0 {
		t.Fatal("search did not filter")
	}
	if _, err = Find(servers, "PROD"); err != nil {
		t.Fatal(err)
	}
	err = store.Update(func(items []Server) ([]Server, error) {
		return append(items, Server{Name: "PROD", Host: "other", Port: 22}), nil
	})
	if err == nil {
		t.Fatal("duplicate accepted")
	}
	servers, _ = store.Load()
	if len(servers) != 1 {
		t.Fatal("failed mutation changed inventory")
	}
}

func TestOversizedMutationPreservesReadableInventory(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "servers.json")}
	if err := store.Update(func(items []Server) ([]Server, error) {
		return []Server{{Name: "original", Host: "localhost", Port: 22}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	err := store.Update(func(items []Server) ([]Server, error) {
		for i := 0; i < 2100; i++ {
			items = append(items, Server{Name: fmt.Sprintf("host-%d", i), Host: "localhost", Port: 22, Notes: strings.Repeat("x", 4096)})
		}
		return items, nil
	})
	if err == nil {
		t.Fatal("oversized mutation was accepted")
	}
	items, err := store.Load()
	if err != nil || len(items) != 1 || items[0].Name != "original" {
		t.Fatalf("original lost: %d %v", len(items), err)
	}
}

func TestInvalidDataIsNeverOverwritten(t *testing.T) {
	for _, data := range []string{`broken`, `{"version":99,"servers":[]}`, `{"version":1,"servers":null}`, `{"version":1,"servers":[{"name":"bad","host":"-oProxyCommand=evil","port":22}]}`, `{"version":1,"servers":[]} trailing`} {
		t.Run(data, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "servers.json")
			os.WriteFile(path, []byte(data), 0600)
			store := Store{Path: path}
			err := store.Update(func(items []Server) ([]Server, error) { return []Server{}, nil })
			if err == nil {
				t.Fatal("corruption accepted")
			}
			after, _ := os.ReadFile(path)
			if string(after) != data {
				t.Fatal("corrupt file overwritten")
			}
		})
	}
}

func TestValidationRejectsUnsafeFields(t *testing.T) {
	for _, s := range []Server{
		{Name: "", Host: "host", Port: 22}, {Name: "bad name", Host: "host", Port: 22},
		{Name: "prod", Host: "-evil", Port: 22}, {Name: "prod", Host: "a b", Port: 22},
		{Name: "prod", Host: "user@host", Port: 22}, {Name: "prod", Host: "$(evil)", Port: 22},
		{Name: "prod", Host: "host", User: "-evil", Port: 22}, {Name: "prod", Host: "host", Port: 0},
		{Name: "prod", Host: "host", Port: 65536}, {Name: "prod", Host: "host", Port: 22, Notes: "\x1b[31m"},
		{Name: "prod", Host: "host", Port: 22, IdentityFile: "key\nfile"},
	} {
		if err := Validate(s); err == nil {
			t.Errorf("accepted unsafe server: %#v", s)
		}
	}
	for _, host := range []string{"203.0.113.10", "db.internal", "::1", "[2001:db8::1]", "my_alias"} {
		if err := Validate(Server{Name: "valid", Host: host, Port: 22}); err != nil {
			t.Errorf("valid host %q: %v", host, err)
		}
	}
}

func TestConcurrentUpdatesPreserveBothWriters(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "servers.json")}
	var wg sync.WaitGroup
	for _, name := range []string{"one", "two", "three", "four"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			err := store.Update(func(items []Server) ([]Server, error) {
				return append(items, Server{Name: name, Host: "localhost", Port: 22}), nil
			})
			if err != nil {
				t.Error(err)
			}
		}(name)
	}
	wg.Wait()
	servers, err := store.Load()
	if err != nil || len(servers) != 4 {
		t.Fatalf("lost update: %v %v", servers, err)
	}
}

func TestLockTimeoutDoesNotRemoveAnotherWritersLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.json")
	os.WriteFile(path+".lock", []byte("busy"), 0600)
	err := (Store{Path: path}).Update(func(items []Server) ([]Server, error) { return items, nil })
	if err == nil {
		t.Fatal("locked write succeeded")
	}
	if _, err := os.Stat(path + ".lock"); err != nil {
		t.Fatal("another writer's lock removed")
	}
}
