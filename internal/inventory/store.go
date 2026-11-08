package inventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Store struct{ Path string }

const maxInventoryBytes = 8 << 20

type document struct {
	Version int      `json:"version"`
	Servers []Server `json:"servers"`
}

func DefaultPath() (string, error) {
	if path := os.Getenv("HOVA_CONFIG"); path != "" {
		return filepath.Abs(path)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hova", "servers.json"), nil
}

func (s Store) Load() ([]Server, error) {
	f, err := os.Open(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return []Server{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open inventory: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > maxInventoryBytes {
		return nil, fmt.Errorf("inventory exceeds the 8 MiB limit; original file preserved")
	}
	var doc document
	decoder := json.NewDecoder(io.LimitReader(f, maxInventoryBytes))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("invalid inventory %s: %w (original file preserved)", s.Path, err)
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("invalid inventory: unexpected trailing data")
	}
	if doc.Version != 1 || doc.Servers == nil {
		return nil, fmt.Errorf("unsupported inventory format: expected version 1 and a servers array")
	}
	if err = validateAll(doc.Servers); err != nil {
		return nil, err
	}
	sort.Slice(doc.Servers, func(i, j int) bool {
		return strings.ToLower(doc.Servers[i].Name) < strings.ToLower(doc.Servers[j].Name)
	})
	return doc.Servers, nil
}

func validateAll(servers []Server) error {
	names := map[string]bool{}
	for _, s := range servers {
		if err := Validate(s); err != nil {
			return err
		}
		name := strings.ToLower(s.Name)
		if names[name] {
			return fmt.Errorf("server %q already exists", s.Name)
		}
		names[name] = true
	}
	return nil
}

func (s Store) Update(change func([]Server) ([]Server, error)) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return err
	}
	unlock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock()
	servers, err := s.Load()
	if err != nil {
		return err
	}
	servers, err = change(servers)
	if err != nil {
		return err
	}
	if servers == nil {
		servers = []Server{}
	}
	if err = validateAll(servers); err != nil {
		return err
	}
	return s.save(servers)
}

func (s Store) lock() (func(), error) {
	path := s.Path + ".lock"
	deadline := time.Now().Add(2 * time.Second)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			fmt.Fprintf(f, "pid=%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("inventory is locked: %s; retry, or remove this lock only after confirming no Hova process is writing", path)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (s Store) save(servers []Server) error {
	f, err := os.CreateTemp(filepath.Dir(s.Path), ".hova-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = f.Chmod(0600); err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(document{Version: 1, Servers: servers}); err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() > maxInventoryBytes {
		return fmt.Errorf("inventory would exceed the 8 MiB limit; original file preserved")
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), s.Path); err != nil {
		return fmt.Errorf("replace inventory (original preserved): %w", err)
	}
	return nil
}
