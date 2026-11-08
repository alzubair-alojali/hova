package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alzubair-alojali/hova/internal/inventory"
	sshclient "github.com/alzubair-alojali/hova/internal/ssh"
	"github.com/spf13/cobra"
)

func (a *app) addImportCommand() {
	var path string
	parent := &cobra.Command{Use: "import", Short: "Import existing connection profiles", Args: cobra.NoArgs}
	cmd := &cobra.Command{Use: "ssh", Short: "Import concrete aliases from an SSH config file", Long: "Imports concrete Host aliases from one SSH config file. Wildcards and negations are skipped.\nInclude files are not enumerated. Connections use the original config so OpenSSH resolves all settings.\nThe displayed address is metadata from the alias block; it is not a connectivity check.", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if path == "" {
			dir, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			path = filepath.Join(dir, ".ssh", "config")
		}
		absolute, err := filepath.Abs(sshclient.ExpandHome(path))
		if err != nil {
			return err
		}
		f, err := os.Open(absolute)
		if err != nil {
			return fmt.Errorf("open SSH config: %w", err)
		}
		defer f.Close()
		imported, err := sshclient.Import(f, absolute)
		if err != nil {
			return err
		}
		store, err := a.store()
		if err != nil {
			return err
		}
		added, skipped := []string{}, []string{}
		err = store.Update(func(items []inventory.Server) ([]inventory.Server, error) {
			names := map[string]bool{}
			for _, s := range items {
				names[strings.ToLower(s.Name)] = true
			}
			for _, s := range imported {
				if names[strings.ToLower(s.Name)] {
					skipped = append(skipped, s.Name)
					continue
				}
				items = append(items, s)
				added = append(added, s.Name)
				names[strings.ToLower(s.Name)] = true
			}
			return items, nil
		})
		if err != nil {
			return err
		}
		return a.output(cmd, struct {
			Added   []string `json:"added"`
			Skipped []string `json:"skipped"`
		}{added, skipped}, fmt.Sprintf("Imported %d servers · skipped %d existing names", len(added), len(skipped)))
	}}
	cmd.Flags().StringVarP(&path, "file", "f", "", "SSH config path (default: ~/.ssh/config)")
	parent.AddCommand(cmd)
	a.root.AddCommand(parent)
}
