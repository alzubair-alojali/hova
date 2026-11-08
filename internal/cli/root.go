package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alzubair-alojali/hova/internal/inventory"
	sshclient "github.com/alzubair-alojali/hova/internal/ssh"
	"github.com/alzubair-alojali/hova/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type app struct {
	root   *cobra.Command
	config string
	json   bool
}
type exitStatus struct{ code int }

func (e exitStatus) Error() string { return fmt.Sprintf("SSH exited with status %d", e.code) }

func Execute(args []string, out, errOut io.Writer, version string) int {
	root := New(version)
	root.SetArgs(args)
	root.SetOut(out)
	root.SetErr(errOut)
	err := root.Execute()
	if err == nil {
		return 0
	}
	var status exitStatus
	if errors.As(err, &status) {
		return status.code
	}
	jsonMode := false
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "--json" || arg == "--json=true" {
			jsonMode = true
		}
	}
	if jsonMode {
		json.NewEncoder(errOut).Encode(map[string]string{"error": err.Error()})
	} else {
		fmt.Fprintf(errOut, "hova: %v\n", err)
	}
	return 1
}

func New(version string) *cobra.Command {
	a := &app{}
	a.root = &cobra.Command{Use: "hova", Short: "Your servers. One command away.", Long: "H O V A\nYour servers. One command away.\n\nA local SSH address book for your terminal and your agents.\nRun hova to open the interactive workspace.", Version: version, SilenceErrors: true, SilenceUsage: true, Args: cobra.NoArgs}
	a.root.PersistentFlags().StringVar(&a.config, "config", "", "Inventory path (or HOVA_CONFIG)")
	a.root.PersistentFlags().BoolVar(&a.json, "json", false, "Machine-readable JSON output")
	a.root.RunE = func(cmd *cobra.Command, args []string) error {
		if a.json {
			return fmt.Errorf("choose a command with --json, for example: hova list --json")
		}
		if !interactive() {
			return cmd.Help()
		}
		store, err := a.store()
		if err != nil {
			return err
		}
		return tui.Run(store)
	}
	a.addServerCommands()
	a.addConnectionCommands()
	a.addImportCommand()
	config := &cobra.Command{Use: "config", Short: "Inspect local configuration", Args: cobra.NoArgs}
	config.AddCommand(&cobra.Command{Use: "path", Short: "Print the inventory file path", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		store, err := a.store()
		if err != nil {
			return err
		}
		return a.output(cmd, map[string]string{"path": store.Path}, store.Path)
	}})
	a.root.AddCommand(config)
	return a.root
}

func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func (a *app) store() (inventory.Store, error) {
	path := a.config
	var err error
	if path == "" {
		path, err = inventory.DefaultPath()
	}
	if err != nil {
		return inventory.Store{}, err
	}
	path, err = filepath.Abs(sshclient.ExpandHome(path))
	return inventory.Store{Path: path}, err
}

func (a *app) output(cmd *cobra.Command, value any, text string) error {
	if a.json {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), text)
	return err
}

func (a *app) server(name string) (inventory.Server, error) {
	store, err := a.store()
	if err != nil {
		return inventory.Server{}, err
	}
	servers, err := store.Load()
	if err != nil {
		return inventory.Server{}, err
	}
	return inventory.Find(servers, name)
}

func (a *app) complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	store, err := a.store()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	servers, err := store.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	names := []string{}
	for _, s := range inventory.Filter(servers, toComplete, "") {
		names = append(names, s.Name+"\t"+s.Host)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
