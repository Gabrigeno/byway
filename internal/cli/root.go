// Package cli wires byway's commands.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/gabrielegenovesi/byway/internal/runner"
	"github.com/gabrielegenovesi/byway/internal/store"
	"github.com/gabrielegenovesi/byway/internal/tmpl"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// exitError carries a wrapped command's exit code without printing anything.
type exitError int

func (e exitError) Error() string { return fmt.Sprintf("exit status %d", int(e)) }

// Main runs byway and returns the process exit code.
func Main(args []string, in io.Reader, out, errw io.Writer) int {
	root := newRoot()
	root.SetArgs(args)
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(errw)
	err := root.Execute()
	var ec exitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ec):
		return int(ec)
	default:
		fmt.Fprintln(errw, "byway:", err)
		return 1
	}
}

type app struct {
	file   string
	dryRun bool
}

func newRoot() *cobra.Command {
	a := &app{}
	root := &cobra.Command{
		Use:           "byway [flags] <name> [args]",
		Short:         "Save shell commands by name, run them anywhere",
		Version:       Version,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return a.run(cmd, args)
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.Flags().SetInterspersed(false) // everything after <name> belongs to the command
	root.PersistentFlags().StringVar(&a.file, "file", "", "registry file (default $BYWAY_FILE or ~/.config/byway/byway.toml)")
	root.PersistentFlags().BoolVarP(&a.dryRun, "dry-run", "n", false, "print the resolved command instead of running it")
	root.AddCommand(a.runCmd(), a.showCmd(), a.lsCmd(), a.namesCmd())
	a.addManage(root)
	a.addSetup(root)
	return root
}

func (a *app) path() string {
	if a.file != "" {
		return a.file
	}
	return store.Path()
}

func (a *app) load() (*store.Registry, error) { return store.Load(a.path()) }

// run resolves args to a command and executes (or prints) it.
func (a *app) run(cmd *cobra.Command, args []string) error {
	reg, err := a.load()
	if err != nil {
		return err
	}
	name, rest, err := resolveName(reg, args)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	refs(reg, name, known)
	flags, pos := splitArgs(rest, known)
	steps, err := plan(reg, name, pos, flags)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if a.dryRun {
		for _, s := range steps {
			fmt.Fprintln(cmd.OutOrStdout(), s.Line)
		}
		return nil
	}
	code, err := runner.Run(steps, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	if code != 0 {
		return exitError(code)
	}
	return nil
}

// resolveName finds the longest prefix of args that, joined by ".", names a command.
func resolveName(reg *store.Registry, args []string) (string, []string, error) {
	for i := len(args); i >= 1; i-- {
		n := strings.Join(args[:i], ".")
		if _, ok := reg.Cmd[n]; ok {
			return n, args[i:], nil
		}
	}
	return "", nil, notFound(reg, args[0])
}

func notFound(reg *store.Registry, typed string) error {
	var near []string
	for _, n := range reg.Names() {
		if levenshtein(typed, n) <= 2 {
			near = append(near, n)
		}
	}
	if len(near) > 0 {
		return fmt.Errorf("%q not found. Did you mean: %s?", typed, strings.Join(near, ", "))
	}
	return fmt.Errorf("%q not found (see: byway ls)", typed)
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func refs(reg *store.Registry, name string, into map[string]bool) {
	c := reg.Cmd[name]
	for _, k := range tmpl.Refs(c.Run) {
		into[k] = true
	}
	for _, s := range c.Steps {
		refs(reg, s, into)
	}
}

var flagRe = regexp.MustCompile(`^--([A-Za-z0-9_-]+)=(.*)$`)

// splitArgs pulls --k=v for referenced k (before "--"); the rest is positional.
func splitArgs(rest []string, known map[string]bool) (map[string]string, []string) {
	flags := map[string]string{}
	var pos []string
	for i, arg := range rest {
		if arg == "--" {
			return flags, append(pos, rest[i+1:]...)
		}
		if m := flagRe.FindStringSubmatch(arg); m != nil && known[m[1]] {
			flags[m[1]] = m[2]
			continue
		}
		pos = append(pos, arg)
	}
	return flags, pos
}

func plan(reg *store.Registry, name string, pos []string, flags map[string]string) ([]runner.Step, error) {
	c := reg.Cmd[name]
	if c.Run != "" {
		line, err := tmpl.Resolve(c.Run, tmpl.Input{Args: pos, Flags: flags, Vars: reg.Vars, Shell: c.Shell})
		if err != nil {
			return nil, err
		}
		return []runner.Step{{Shell: c.Shell, Line: line}}, nil
	}
	if len(pos) > 0 {
		return nil, errors.New("takes no arguments (it runs steps)")
	}
	var out []runner.Step
	for _, s := range c.Steps {
		sub, err := plan(reg, s, nil, flags)
		if err != nil {
			return nil, fmt.Errorf("step %s: %w", s, err)
		}
		out = append(out, sub...)
	}
	return out, nil
}

// eatGlobal parses -n/--dry-run/--file for subcommands that disable flag parsing.
func (a *app) eatGlobal(args []string) []string {
	for len(args) > 0 {
		switch {
		case args[0] == "-n" || args[0] == "--dry-run":
			a.dryRun = true
		case args[0] == "--file" && len(args) > 1:
			a.file = args[1]
			args = args[1:]
		case strings.HasPrefix(args[0], "--file="):
			a.file = strings.TrimPrefix(args[0], "--file=")
		default:
			return args
		}
		args = args[1:]
	}
	return args
}

func (a *app) runCmd() *cobra.Command {
	return &cobra.Command{
		Use: "run <name> [args]", Short: "Run a command (for names that collide with subcommands)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if args = a.eatGlobal(args); len(args) == 0 {
				return errors.New("usage: byway run <name> [args]")
			}
			return a.run(cmd, args)
		},
	}
}

func (a *app) showCmd() *cobra.Command {
	return &cobra.Command{
		Use: "show <name> [args]", Short: "Print the resolved command without running it",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if args = a.eatGlobal(args); len(args) == 0 {
				return errors.New("usage: byway show <name> [args]")
			}
			a.dryRun = true
			return a.run(cmd, args)
		},
	}
}

func summary(c store.Command) string {
	switch {
	case c.Desc != "":
		return c.Desc
	case c.Run != "":
		return c.Run
	default:
		return "steps: " + strings.Join(c.Steps, ", ")
	}
}

func (a *app) lsCmd() *cobra.Command {
	return &cobra.Command{
		Use: "ls [namespace]", Short: "List saved commands", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := a.load()
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			for _, n := range reg.Names() {
				if len(args) == 1 && n != args[0] && !strings.HasPrefix(n, args[0]+".") {
					continue
				}
				fmt.Fprintf(w, "%s\t%s\n", n, summary(reg.Cmd[n]))
			}
			return w.Flush()
		},
	}
}

// namesCmd feeds shell completions: "name<TAB>summary" per line.
func (a *app) namesCmd() *cobra.Command {
	return &cobra.Command{
		Use: "__names", Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := a.load()
			if err != nil {
				return err
			}
			for _, n := range reg.Names() {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", n, summary(reg.Cmd[n]))
			}
			return nil
		},
	}
}

// isTTY reports whether v is an *os.File attached to a terminal.
func isTTY(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func (a *app) addManage(root *cobra.Command) {} // replaced in Task 7
func (a *app) addSetup(root *cobra.Command)  {} // replaced in Task 8
