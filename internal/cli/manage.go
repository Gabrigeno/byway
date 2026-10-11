package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Gabrigeno/byway/internal/capture"
	"github.com/Gabrigeno/byway/internal/shell"
	"github.com/Gabrigeno/byway/internal/store"
)

func (a *app) addManage(root *cobra.Command) {
	root.AddCommand(a.saveCmd(), a.addCmd(), a.rmCmd(), a.editCmd())
}

// reserved: a name whose first segment is a subcommand would be unreachable.
func reserved(root *cobra.Command, name string) bool {
	first, _, _ := strings.Cut(name, ".")
	if first == "help" {
		return true
	}
	for _, c := range root.Commands() {
		if c.Name() == first {
			return true
		}
	}
	return false
}

func (a *app) put(cmd *cobra.Command, name string, c store.Command, force bool) error {
	if reserved(cmd.Root(), name) {
		return fmt.Errorf("%q clashes with a byway subcommand; pick another name", name)
	}
	if err := store.Put(a.path(), name, c, force); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "saved %s: %s\n", name, c.Run)
	return nil
}

func (a *app) saveCmd() *cobra.Command {
	var desc string
	var force bool
	var subs []string
	c := &cobra.Command{
		Use: "save <name>", Short: "Save the previous shell command as <name>", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw := os.Getenv("BYWAY_LAST_CMD")
			if raw == "" {
				return errors.New("no shell history available: run `byway setup` and open a new shell, or use: byway add <name> -- <command>")
			}
			line, err := capture.PickLast(raw)
			if err != nil {
				return err
			}
			if line, err = capture.Parametrize(line, subs); err != nil {
				return err
			}
			return a.put(cmd, args[0], store.Command{Run: line, Desc: desc}, force)
		},
	}
	c.Flags().StringVarP(&desc, "desc", "d", "", "description")
	c.Flags().BoolVarP(&force, "force", "f", false, "overwrite an existing command")
	c.Flags().StringArrayVar(&subs, "arg", nil, "turn a value into a placeholder: N=value or name=value")
	return c
}

func (a *app) addCmd() *cobra.Command {
	var desc string
	var force bool
	c := &cobra.Command{
		Use: "add <name> -- <command...>", Short: "Save a command given explicitly or on stdin", Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var line string
			switch dash := cmd.ArgsLenAtDash(); {
			case dash == 1 && len(args) > 1:
				words := make([]string, len(args)-1)
				for i, w := range args[1:] {
					words[i] = shell.Quote(w, "sh")
				}
				line = strings.Join(words, " ")
			case len(args) == 1 && dash < 0 && !isTTY(cmd.InOrStdin()):
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				line = strings.TrimSpace(string(data))
			default:
				return errors.New("usage: byway add <name> -- <command>   (or pipe the command on stdin)")
			}
			if line == "" {
				return errors.New("empty command")
			}
			return a.put(cmd, args[0], store.Command{Run: line, Desc: desc}, force)
		},
	}
	c.Flags().StringVarP(&desc, "desc", "d", "", "description")
	c.Flags().BoolVarP(&force, "force", "f", false, "overwrite an existing command")
	return c
}

func (a *app) rmCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use: "rm <name>", Short: "Delete a saved command", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := a.load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := reg.Cmd[name]; !ok {
				return notFound(reg, name)
			}
			if !yes {
				if !isTTY(cmd.InOrStdin()) {
					return errors.New("refusing to delete without confirmation: use -y")
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "delete %s (%s)? [y/N] ", name, summary(reg.Cmd[name]))
				ans, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				if r := strings.ToLower(strings.TrimSpace(ans)); r != "y" && r != "yes" {
					fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
					return nil
				}
			}
			if err := store.Remove(a.path(), name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", name)
			return nil
		},
	}
	c.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	return c
}

func (a *app) editCmd() *cobra.Command {
	return &cobra.Command{
		Use: "edit [name]", Short: "Open the registry in $EDITOR", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := a.path()
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				if err := store.Put(path, "hello", store.Command{Run: "echo hello from byway", Desc: "example"}, false); err != nil {
					return err
				}
			}
			editor := os.Getenv("VISUAL")
			if editor == "" {
				editor = os.Getenv("EDITOR")
			}
			if editor == "" {
				editor = "vi"
			}
			argv := strings.Fields(editor)
			if len(args) == 1 {
				n, err := store.Line(path, args[0])
				if errors.Is(err, store.ErrNotFound) {
					reg, _ := a.load()
					return notFound(reg, args[0])
				}
				if err != nil {
					return err
				}
				argv = append(argv, fmt.Sprintf("+%d", n))
			}
			argv = append(argv, path)
			e := exec.Command(argv[0], argv[1:]...)
			e.Stdin, e.Stdout, e.Stderr = cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()
			if err := e.Run(); err != nil {
				return fmt.Errorf("editor: %w", err)
			}
			if _, err := a.load(); err != nil {
				return fmt.Errorf("registry is now invalid, run `byway edit` again: %w", err)
			}
			return nil
		},
	}
}
