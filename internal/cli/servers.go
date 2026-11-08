package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/alzubair-alojali/hova/internal/inventory"
	"github.com/alzubair-alojali/hova/internal/tui"
	"github.com/spf13/cobra"
)

func (a *app) addServerCommands() {
	var query, tag string
	list := &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List and search saved servers", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		store, err := a.store()
		if err != nil {
			return err
		}
		servers, err := store.Load()
		if err != nil {
			return err
		}
		servers = inventory.Filter(servers, query, tag)
		if a.json {
			return a.output(cmd, servers, "")
		}
		if len(servers) == 0 {
			return a.output(cmd, nil, "No servers found. Add one with: hova add")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\n  H O V A  /  %d servers\n\n", len(servers))
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 3, ' ', 0)
		fmt.Fprintln(w, "  NAME\tADDRESS\tUSER\tPORT\tTAGS")
		for _, s := range servers {
			fmt.Fprintf(w, "  %s\t%s\t%s\t%d\t%s\n", s.Name, s.Host, s.User, s.Port, strings.Join(s.Tags, ", "))
		}
		return w.Flush()
	}}
	list.Flags().StringVarP(&query, "search", "s", "", "Search name, host, user, tags and notes")
	list.Flags().StringVarP(&tag, "tag", "t", "", "Filter by exact tag")
	show := &cobra.Command{Use: "show NAME", Short: "Show connection details without connecting", Args: cobra.ExactArgs(1), ValidArgsFunction: a.complete, RunE: func(cmd *cobra.Command, args []string) error {
		s, err := a.server(args[0])
		if err != nil {
			return err
		}
		text := fmt.Sprintf("\n  %s\n\n  Address   %s\n  User      %s\n  Port      %d\n  Key       %s\n  Tags      %s\n  Notes     %s\n", s.Name, s.Host, s.User, s.Port, s.IdentityFile, strings.Join(s.Tags, ", "), s.Notes)
		if s.SSHAlias != "" {
			text += fmt.Sprintf("  SSH alias %s\n  Config    %s\n", s.SSHAlias, s.SSHConfig)
		}
		return a.output(cmd, s, text)
	}}
	a.root.AddCommand(list, show, a.mutationCommand(false), a.mutationCommand(true), a.removeCommand())
}

func (a *app) mutationCommand(edit bool) *cobra.Command {
	s := inventory.Server{Port: 22}
	var tags string
	verb := "add"
	if edit {
		verb = "edit"
	}
	cmd := &cobra.Command{Use: verb + " [NAME]", Short: strings.ToUpper(verb[:1]) + verb[1:] + " a server; omit flags for a guided form", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		store, err := a.store()
		if err != nil {
			return err
		}
		if len(args) == 0 {
			if edit {
				return fmt.Errorf("usage: hova edit NAME [flags]")
			}
			if a.json || !interactive() {
				return fmt.Errorf("usage: hova add NAME --host ADDRESS [flags]; run hova add in a terminal for the guided form")
			}
			return tui.RunForm(store, inventory.Server{Port: 22}, false)
		}
		original := ""
		if edit {
			original = args[0]
		}
		if edit && !hasServerFlags(cmd) {
			existing, err := a.server(original)
			if err != nil {
				return err
			}
			if a.json || !interactive() {
				return fmt.Errorf("provide at least one field to edit, for example --host ADDRESS")
			}
			return tui.RunForm(store, existing, true)
		}
		if !edit {
			s.Name = args[0]
			s.Tags = parseTags(tags)
			if err = inventory.Validate(s); err != nil {
				return err
			}
		}
		err = store.Update(func(items []inventory.Server) ([]inventory.Server, error) {
			if !edit {
				return append(items, s), nil
			}
			for i, item := range items {
				if strings.EqualFold(item.Name, original) {
					s = mergeChanges(cmd, item, s, tags)
					items[i] = s
					return items, nil
				}
			}
			return nil, fmt.Errorf("server %q no longer exists", original)
		})
		if err != nil {
			return err
		}
		return a.output(cmd, s, fmt.Sprintf("Saved %s · %s", s.Name, s.Host))
	}}
	cmd.ValidArgsFunction = a.complete
	cmd.Flags().StringVar(&s.Host, "host", "", "IP address, hostname or SSH alias")
	cmd.Flags().StringVarP(&s.User, "user", "u", "", "SSH username (default: native SSH settings)")
	cmd.Flags().IntVarP(&s.Port, "port", "p", 22, "SSH port")
	cmd.Flags().StringVarP(&s.IdentityFile, "identity", "i", "", "Path to SSH private key (key contents are never stored)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags; replaces existing tags")
	cmd.Flags().StringVar(&s.Notes, "notes", "", "Short server notes")
	if edit {
		cmd.Flags().StringVar(&s.Name, "name", "", "Rename the server")
	}
	return cmd
}

func hasServerFlags(cmd *cobra.Command) bool {
	for _, name := range []string{"host", "user", "port", "identity", "tags", "notes", "name"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func mergeChanges(cmd *cobra.Command, old, next inventory.Server, tags string) inventory.Server {
	connectionChanged := false
	if cmd.Flags().Changed("host") {
		old.Host = next.Host
		connectionChanged = true
	}
	if cmd.Flags().Changed("user") {
		old.User = next.User
		connectionChanged = true
	}
	if cmd.Flags().Changed("port") {
		old.Port = next.Port
		connectionChanged = true
	}
	if cmd.Flags().Changed("identity") {
		old.IdentityFile = next.IdentityFile
		connectionChanged = true
	}
	if cmd.Flags().Changed("name") {
		old.Name = next.Name
	}
	if cmd.Flags().Changed("tags") {
		old.Tags = parseTags(tags)
	}
	if cmd.Flags().Changed("notes") {
		old.Notes = next.Notes
	}
	if connectionChanged {
		old.SSHAlias = ""
		old.SSHConfig = ""
	}
	return old
}

func parseTags(value string) []string {
	var tags []string
	seen := map[string]bool{}
	for _, tag := range strings.Split(value, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" && !seen[strings.ToLower(tag)] {
			tags = append(tags, tag)
			seen[strings.ToLower(tag)] = true
		}
	}
	return tags
}

func (a *app) removeCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "remove NAME", Aliases: []string{"rm"}, Short: "Remove a saved server (does not change the server itself)", Args: cobra.ExactArgs(1), ValidArgsFunction: a.complete, RunE: func(cmd *cobra.Command, args []string) error {
		if !yes {
			return fmt.Errorf("removal needs --yes; use the interactive workspace for a confirmation dialog")
		}
		store, err := a.store()
		if err != nil {
			return err
		}
		err = store.Update(func(items []inventory.Server) ([]inventory.Server, error) {
			for i, s := range items {
				if strings.EqualFold(s.Name, args[0]) {
					return append(items[:i], items[i+1:]...), nil
				}
			}
			return nil, fmt.Errorf("server %q not found", args[0])
		})
		if err != nil {
			return err
		}
		return a.output(cmd, map[string]string{"removed": args[0]}, "Removed "+args[0]+" from Hova")
	}}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm removal")
	return cmd
}
