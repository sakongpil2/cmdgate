package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sakongpil2/cmdgate/internal/allowlist"
	"github.com/sakongpil2/cmdgate/internal/audit"
	"github.com/sakongpil2/cmdgate/internal/matchers"
	"github.com/sakongpil2/cmdgate/internal/policy"
	"github.com/sakongpil2/cmdgate/internal/runner"
)

const (
	allowlistPath = "/opt/cmdgate/allowlist.yaml"
	auditLogPath  = "/var/log/cmdgate/audit.log"
)

// executor holds the privileged executor's configuration paths. Methods on
// executor implement the cmdgate-exec subcommands so they can be tested with
// temporary files.
type executor struct {
	allowlistPath string
	auditLogPath  string
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "--help" {
		printHelp()
		return
	}
	e := executor{allowlistPath: allowlistPath, auditLogPath: auditLogPath}
	switch os.Args[1] {
	case "run":
		if err := e.handleRun(os.Args[2:]); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "policy":
		if err := e.handlePolicy(os.Args[2:]); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "audit":
		if err := e.handleAudit(os.Args[2:]); err != nil {
			printError(err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`CmdGate - allowlist-based privileged command executor

Usage:
  cmdgate-exec <command> [args...]

Commands:
  run     Run a pre-approved command
  policy  Validate a policy bundle
  audit   View audit logs
  help    Show this help message

Examples:
  cmdgate-exec run list
  cmdgate-exec run systemctl restart kubelet
  cmdgate-exec policy validate --bundle cmdgate-policy-1.1.0.tar.gz
  cmdgate-exec audit tail 50`)
}

func printError(err error) {
	msg := err.Error()
	if strings.Contains(msg, "command not allowed") && isTerminal(os.Stderr) && os.Getenv("NO_COLOR") == "" {
		fmt.Fprintf(os.Stderr, "\x1b[31m%s\x1b[0m\n", msg)
		return
	}
	fmt.Fprintln(os.Stderr, msg)
}

func (e *executor) handleRun(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cmdgate-exec run <command> [args...]")
	}
	if args[0] == "list" {
		return e.runList()
	}
	commandText := strings.Join(args, " ")

	cfg, err := e.loadConfig()
	if err != nil {
		return err
	}

	cmd, placeholders, ok := cfg.FindCommandWithPlaceholders(args)
	rpmName := ""
	rpmPaths := []string(nil)

	// A single rpmFiles placeholder that consumes exactly one argv value.
	if ok && len(placeholders) == 1 {
		def, exists := cfg.Matchers[placeholders[0].Name]
		if exists && def.Type == "rpmFiles" {
			rpmName = placeholders[0].Name
			rpmPaths = []string{placeholders[0].Value}
		}
	}

	// A trailing rpmFiles placeholder that consumes the rest of argv.
	if !ok {
		cmd, rpmName, rpmPaths, ok = e.findTrailingRpmFiles(cfg, args)
	}

	if !ok {
		e.writeAuditWarning(audit.LogEntry{Action: "run", Command: commandText, Result: "denied", Reason: "no matching command"})
		return fmt.Errorf("command not allowed")
	}

	if rpmPaths != nil {
		if err := validateRpmFiles(cfg, rpmName, rpmPaths); err != nil {
			e.writeAuditWarning(audit.LogEntry{Action: "run", CommandID: cmd.ID, Command: commandText, Result: "denied", Reason: err.Error()})
			return fmt.Errorf("validation failed: %w", err)
		}
	} else {
		if err := e.validatePlaceholders(cfg, placeholders); err != nil {
			e.writeAuditWarning(audit.LogEntry{Action: "run", CommandID: cmd.ID, Command: commandText, Result: "denied", Reason: err.Error()})
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	err = runner.RunWithIO(args[0], args[1:], os.Stdin, os.Stdout, os.Stderr)
	result := "success"
	reason := ""
	if err != nil {
		result = "failure"
		reason = err.Error()
	}
	e.writeAuditWarning(audit.LogEntry{Action: "run", CommandID: cmd.ID, Command: commandText, Result: result, Reason: reason})
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func (e *executor) runList() error {
	cfg, err := e.loadConfig()
	if err != nil {
		return err
	}

	colors := newColors(os.Stdout)
	rows := [][]string{
		{colors.header("ID"), colors.header("DESCRIPTION"), colors.header("COMMAND")},
	}

	sorted := make([]allowlist.Command, len(cfg.Commands))
	copy(sorted, cfg.Commands)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	for _, c := range sorted {
		rows = append(rows, []string{
			colors.id(c.ID),
			colors.desc(c.Desc),
			colors.cmd(c.Cmd),
		})
	}

	widths := make([]int, 3)
	for _, row := range rows {
		for i, cell := range row {
			if w := visibleWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	for _, row := range rows {
		printRow(widths, row, "  ")
	}
	return nil
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleWidth(s string) int {
	return utf8.RuneCountInString(ansiEscape.ReplaceAllString(s, ""))
}

func printRow(widths []int, cells []string, indent string) {
	fmt.Print(indent)
	for i, cell := range cells {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Print(cell)
		pad := widths[i] - visibleWidth(cell)
		if pad > 0 {
			fmt.Print(strings.Repeat(" ", pad))
		}
	}
	fmt.Println()
}

// colors wraps ANSI escape sequences. When stdout is not a terminal or the
// NO_COLOR environment variable is set, all methods return the input unchanged.
type colors struct {
	enabled bool
}

func newColors(out *os.File) colors {
	return colors{enabled: isTerminal(out) && os.Getenv("NO_COLOR") == ""}
}

func (c colors) header(s string) string {
	if !c.enabled {
		return s
	}
	return "\x1b[1;36m" + s + "\x1b[0m"
}

func (c colors) id(s string) string {
	if !c.enabled {
		return s
	}
	return "\x1b[33m" + s + "\x1b[0m"
}

func (c colors) desc(s string) string {
	if !c.enabled {
		return s
	}
	return "\x1b[37m" + s + "\x1b[0m"
}

func (c colors) cmd(s string) string {
	if !c.enabled {
		return s
	}
	return "\x1b[32m" + s + "\x1b[0m"
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func (e *executor) handleAudit(args []string) error {
	if len(args) == 0 || args[0] == "--help" {
		return fmt.Errorf("usage: cmdgate-exec audit tail [n]")
	}
	if args[0] != "tail" {
		return fmt.Errorf("unknown audit subcommand: %s", args[0])
	}
	limit := 20
	if len(args) > 1 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n <= 0 {
			return fmt.Errorf("tail count must be a positive integer")
		}
		limit = n
	}

	entries, err := e.readAuditTail(limit)
	if err != nil {
		return err
	}
	for _, line := range entries {
		fmt.Println(line)
	}
	return nil
}

func (e *executor) readAuditTail(limit int) ([]string, error) {
	f, err := os.Open(e.auditLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read audit log: %w", err)
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines, nil
}

func (e *executor) handlePolicy(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cmdgate-exec policy validate --bundle <path>")
	}
	if args[0] != "validate" {
		return fmt.Errorf("unknown policy action: %s", args[0])
	}
	bundle := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--bundle" && i+1 < len(args) {
			bundle = args[i+1]
			break
		}
	}
	if bundle == "" {
		return fmt.Errorf("--bundle required")
	}
	if err := policy.ValidateBundle(bundle); err != nil {
		return err
	}
	e.writeAuditWarning(audit.LogEntry{Action: "policy_validate", CommandID: bundle, Command: bundle, Result: "success"})
	return nil
}

func (e *executor) loadConfig() (*allowlist.Config, error) {
	data, err := os.ReadFile(e.allowlistPath)
	if err != nil {
		return nil, fmt.Errorf("read allowlist: %w", err)
	}
	cfg, err := allowlist.Parse(data)
	if err != nil {
		return nil, err
	}
	if err := cfg.ValidateSchema(); err != nil {
		return nil, fmt.Errorf("invalid allowlist schema: %w", err)
	}
	return cfg, nil
}

func (e *executor) validatePlaceholders(cfg *allowlist.Config, placeholders []allowlist.Placeholder) error {
	for _, p := range placeholders {
		def, ok := cfg.Matchers[p.Name]
		if !ok {
			return fmt.Errorf("unknown matcher: %s", p.Name)
		}
		if p.Type != "" && p.Type != def.Type {
			return fmt.Errorf("placeholder type %q does not match matcher type %q", p.Type, def.Type)
		}
		switch def.Type {
		case "number":
			m := matchers.NumberMatcher{}
			if err := m.Validate(p.Value); err != nil {
				return err
			}
		case "string":
			m := matchers.StringMatcher{Pattern: def.Pattern}
			if err := m.Validate(p.Value); err != nil {
				return err
			}
		case "rpmFiles":
			return fmt.Errorf("rpmFiles matcher must be handled at command level, not single placeholder")
		default:
			return fmt.Errorf("unsupported matcher type: %s", def.Type)
		}
	}
	return nil
}

func validateRpmFiles(cfg *allowlist.Config, name string, paths []string) error {
	def, ok := cfg.Matchers[name]
	if !ok {
		return fmt.Errorf("unknown matcher: %s", name)
	}
	if def.Type != "rpmFiles" {
		return fmt.Errorf("unsupported matcher type: %s", def.Type)
	}
	m := matchers.RpmFilesMatcher{
		MetadataNameIn: def.MetadataNameIn,
		Multiple:       def.Multiple,
		AllowedDirs:    def.AllowedDirs,
	}
	return m.Validate(paths)
}

// findTrailingRpmFiles looks for a command whose last token is an rpmFiles
// placeholder and whose fixed prefix matches the start of argv. The placeholder
// is allowed to consume all remaining argv values, enabling patterns such as
// "dnf install <rpmFiles:k8s-rpms>" with multiple RPM files.
func (e *executor) findTrailingRpmFiles(cfg *allowlist.Config, argv []string) (allowlist.Command, string, []string, bool) {
	for _, cmd := range cfg.Commands {
		parts := strings.Fields(cmd.Cmd)
		if len(parts) == 0 {
			continue
		}
		last := parts[len(parts)-1]
		_, name, ok := allowlist.PlaceholderParts(last)
		if !ok {
			continue
		}
		fixed := parts[:len(parts)-1]
		if len(argv) < len(fixed) {
			continue
		}
		match := true
		for i, p := range fixed {
			if p != argv[i] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		def, ok := cfg.Matchers[name]
		if !ok || def.Type != "rpmFiles" {
			continue
		}
		return cmd, name, argv[len(fixed):], true
	}
	return allowlist.Command{}, "", nil, false
}

func (e *executor) writeAudit(entry audit.LogEntry) error {
	entry.User = effectiveUser()
	w, err := audit.NewWriter(e.auditLogPath)
	if err != nil {
		return err
	}
	defer w.Close()
	return w.Write(entry)
}

func (e *executor) writeAuditWarning(entry audit.LogEntry) {
	if err := e.writeAudit(entry); err != nil {
		fmt.Fprintf(os.Stderr, "audit log warning: %v\n", err)
	}
}

func effectiveUser() string {
	if u := os.Getenv("SUDO_USER"); u != "" {
		return u
	}
	if u, err := user.Current(); err == nil && u != nil {
		return u.Username
	}
	return ""
}
