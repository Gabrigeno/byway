package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gabrielegenovesi/byway/internal/shell"
)

func (a *app) addSetup(root *cobra.Command) {
	root.AddCommand(initCmd(), setupCmd())
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		tty := isTTY(cmd.InOrStdin()) && isTTY(cmd.OutOrStdout())
		sh, ok := offerNeeded(cmd.Name(), os.Getenv, home, tty)
		if !ok {
			return
		}
		rc, _ := shell.RCPath(sh, home)
		fmt.Fprintf(cmd.ErrOrStderr(), "Add byway integration to %s? [Y/n] ", rc)
		ans, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(ans)) {
		case "", "y", "yes":
			if _, err := shell.Setup(sh, home); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "byway: setup failed:", err)
				return
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "done: open a new shell to enable `byway save` and completions")
		default:
			_ = shell.Decline()
		}
	}
}

// offerNeeded decides whether to ask once to install the shell integration.
func offerNeeded(cmdName string, getenv func(string) string, home string, tty bool) (string, bool) {
	switch cmdName {
	case "init", "setup", "__names", "help":
		return "", false
	}
	if !tty || getenv("BYWAY_INIT") != "" {
		return "", false
	}
	sh := shell.DetectShell(getenv("SHELL"))
	if sh == "" || shell.Installed(sh, home) || shell.Declined() {
		return "", false
	}
	return sh, true
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use: "init <fish|zsh|bash>", Short: "Print the shell integration script", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shell.InitScript(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), s)
			return nil
		},
	}
}

func setupCmd() *cobra.Command {
	var sh string
	var uninstall bool
	c := &cobra.Command{
		Use: "setup", Short: "Install (or remove) the shell integration", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sh == "" {
				sh = shell.DetectShell(os.Getenv("SHELL"))
			}
			if sh == "" {
				return fmt.Errorf("cannot detect shell from $SHELL=%q; pass --shell fish|zsh|bash", os.Getenv("SHELL"))
			}
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			if uninstall {
				p, err := shell.Uninstall(sh, home)
				if err == nil {
					fmt.Fprintf(cmd.OutOrStdout(), "removed byway integration from %s\n", p)
				}
				return err
			}
			p, err := shell.Setup(sh, home)
			if err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "byway integration added to %s: open a new shell\n", p)
			}
			return err
		},
	}
	c.Flags().StringVar(&sh, "shell", "", "fish, zsh or bash (default: from $SHELL)")
	c.Flags().BoolVar(&uninstall, "uninstall", false, "remove the integration")
	return c
}
