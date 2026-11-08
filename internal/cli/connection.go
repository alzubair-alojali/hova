package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	sshclient "github.com/alzubair-alojali/hova/internal/ssh"
	"github.com/spf13/cobra"
)

func (a *app) addConnectionCommands() {
	var dryRun bool
	connect := &cobra.Command{Use: "connect NAME", Aliases: []string{"c"}, Short: "Open an SSH session", Args: cobra.ExactArgs(1), ValidArgsFunction: a.complete, RunE: func(cmd *cobra.Command, args []string) error {
		s, err := a.server(args[0])
		if err != nil {
			return err
		}
		plan, err := sshclient.Plan(s, "", false)
		if err != nil {
			return err
		}
		if dryRun {
			return a.output(cmd, plan, fmt.Sprintf("Executable: %s\nArguments: %q", plan.Executable, plan.Args))
		}
		if a.json {
			return fmt.Errorf("connect --json requires --dry-run; use exec for machine-readable remote execution")
		}
		process := plan.Process(cmd.Context())
		process.Stdin = os.Stdin
		process.Stdout = cmd.OutOrStdout()
		process.Stderr = cmd.ErrOrStderr()
		if err = process.Run(); err != nil {
			return processError(err)
		}
		return nil
	}}
	connect.Flags().BoolVar(&dryRun, "dry-run", false, "Show the executable and literal arguments without connecting")
	var timeout time.Duration
	remote := &cobra.Command{Use: "exec NAME -- 'COMMAND'", Short: "Run one explicit remote shell command", Long: "Run exactly one remote shell command string after --. Uses SSH BatchMode and strict host-key checking.\nExample: hova exec prod -- 'uptime && df -h'", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 || cmd.ArgsLenAtDash() != 1 || args[1] == "" {
			return fmt.Errorf("usage: hova exec NAME [flags] -- 'COMMAND'; pass the remote shell command as one quoted string")
		}
		return nil
	}, ValidArgsFunction: a.complete, RunE: func(cmd *cobra.Command, args []string) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be greater than zero")
		}
		s, err := a.server(args[0])
		if err != nil {
			return err
		}
		plan, err := sshclient.Plan(s, args[1], true)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()
		process := plan.Process(ctx)
		var stdout, stderr bytes.Buffer
		if a.json {
			process.Stdout = &stdout
			process.Stderr = &stderr
		} else {
			process.Stdout = cmd.OutOrStdout()
			process.Stderr = cmd.ErrOrStderr()
		}
		err = process.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				code = exit.ExitCode()
			} else {
				return fmt.Errorf("start SSH: %w", err)
			}
		}
		timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
		if timedOut {
			code = 124
		}
		if a.json {
			result := struct {
				Stdout   string `json:"stdout"`
				Stderr   string `json:"stderr"`
				ExitCode int    `json:"exit_code"`
				TimedOut bool   `json:"timed_out"`
			}{stdout.String(), stderr.String(), code, timedOut}
			if err := a.output(cmd, result, ""); err != nil {
				return err
			}
		} else if timedOut {
			fmt.Fprintln(cmd.ErrOrStderr(), "hova: SSH timed out; the remote process may still be running")
		}
		if code != 0 {
			return exitStatus{code: code}
		}
		return nil
	}}
	remote.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Maximum time to wait for SSH (timeout exit code: 124)")
	a.root.AddCommand(connect, remote)
}

func processError(err error) error {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exitStatus{code: exit.ExitCode()}
	}
	return fmt.Errorf("start SSH (is OpenSSH installed?): %w", err)
}
