package ssh

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alzubair-alojali/hova/internal/inventory"
)

type Command struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
}

func Plan(s inventory.Server, remote string, batch bool) (Command, error) {
	if err := inventory.Validate(s); err != nil {
		return Command{}, err
	}
	args := make([]string, 0)
	if batch {
		args = append(args, "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=10", "-T")
	}
	target := s.Host
	if s.SSHAlias != "" {
		args = append(args, "-F", s.SSHConfig)
		target = s.SSHAlias
	} else {
		args = append(args, "-p", strconv.Itoa(s.Port))
		if s.User != "" {
			args = append(args, "-l", s.User)
		}
		if s.IdentityFile != "" {
			args = append(args, "-i", ExpandHome(s.IdentityFile))
		}
	}
	args = append(args, "--", target)
	if remote != "" {
		args = append(args, remote)
	}
	return Command{Executable: "ssh", Args: args}, nil
}

func ExpandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		if dir, err := os.UserHomeDir(); err == nil {
			return filepath.Join(dir, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"))
		}
	}
	return path
}

func (p Command) Process(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, p.Executable, p.Args...)
	// SSH config helpers can retain inherited pipes after the SSH client exits.
	cmd.WaitDelay = 500 * time.Millisecond
	return cmd
}
