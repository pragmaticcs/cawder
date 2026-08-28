package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	defaultExecTimeout = 60 * time.Second
	maxExecTimeout     = 10 * time.Minute
	maxExecOutputBytes = 100 * 1024
)

var destructivePatterns = []namedPattern{
	{regexp.MustCompile(`\bmkfs(\.\w+)?\b`), "formats a filesystem"},
	{regexp.MustCompile(`\bdd\b[^|;\n]*\bof=/dev/`), "writes raw data directly to a block device"},
	{regexp.MustCompile(`\b(fdisk|parted|sgdisk|wipefs)\b`), "modifies disk partitions"},
	{regexp.MustCompile(`\b(shutdown|reboot|halt|poweroff)\b`), "shuts down or restarts the machine"},
	{regexp.MustCompile(`\binit\s+[06]\b`), "changes the system runlevel"},
	{regexp.MustCompile(`\bstop-computer\b|\brestart-computer\b`), "shuts down or restarts the machine"},
	{regexp.MustCompile(`\bformat\s+[a-z]:`), "formats a Windows drive"},
	{regexp.MustCompile(`\bdiskpart\b`), "invokes the Windows disk-partitioning tool"},
	{regexp.MustCompile(`\bvssadmin\b[^|;\n]*\bdelete\b`), "deletes Windows shadow-copy backups"},
	{regexp.MustCompile(`\bbcdedit\b`), "modifies Windows boot configuration"},
	{regexp.MustCompile(`\breg\s+delete\s+hklm`), "deletes a local-machine registry hive"},
}

type namedPattern struct {
	re     *regexp.Regexp
	reason string
}

var forkBombPattern = regexp.MustCompile(`:\s*\(\)\s*\{\s*:\s*\|\s*:\s*&?\s*;?\s*\}\s*;\s*:`)

var rootPaths = map[string]bool{
	"/":                true,
	"/*":               true,
	"~":                true,
	"~/":               true,
	"$home":            true,
	"$env:systemdrive": true,
	"$env:windir":      true,
	"$env:systemroot":  true,
}

func hasRecursiveRootOp(lower string, verbs []string, requiredFlags string) bool {
	verbSet := make(map[string]bool, len(verbs))
	for _, v := range verbs {
		verbSet[v] = true
	}

	fields := strings.Fields(lower)

	for i, f := range fields {
		if !verbSet[f] {
			continue
		}

		have := map[rune]bool{}

		for j := i + 1; j < len(fields); j++ {
			tok := fields[j]

			if rootPaths[tok] {
				ok := true
				for _, r := range requiredFlags {
					if !have[r] {
						ok = false
						break
					}
				}
				if ok {
					return true
				}
				continue
			}

			switch {
			case tok == "--recursive":
				have['r'] = true
			case tok == "--force":
				have['f'] = true
			case strings.HasPrefix(tok, "-") && !strings.HasPrefix(tok, "--"):
				for _, r := range tok[1:] {
					have[r] = true
				}
			}
		}
	}

	return false
}

func blockedReason(rawCommand string) string {
	lower := strings.ToLower(rawCommand)

	if hasRecursiveRootOp(lower, []string{"rm"}, "rf") {
		return "destructive command: recursive force-delete of a root path"
	}

	if hasRecursiveRootOp(lower, []string{"chmod", "chown"}, "r") {
		return "destructive command: recursive permission/ownership change on a root path"
	}

	if strings.Contains(lower, "remove-item") &&
		strings.Contains(lower, "-recurse") &&
		strings.Contains(lower, "-force") {

		for _, root := range []string{
			`c:\`,
			"c:/",
			"$env:systemdrive",
			"$env:windir",
			"$env:systemroot",
		} {
			if strings.Contains(lower, root) {
				return "destructive command: recursive force-delete of a Windows system path"
			}
		}
	}

	if forkBombPattern.MatchString(rawCommand) {
		return "destructive command: shell fork bomb"
	}

	for _, p := range destructivePatterns {
		if p.re.MatchString(lower) {
			return "destructive command: " + p.reason
		}
	}

	return ""
}

type limitedBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (w *limitedBuffer) Write(p []byte) (int, error) {
	room := maxExecOutputBytes - w.buf.Len()
	if room <= 0 {
		w.truncated = true
		return len(p), nil
	}

	if len(p) > room {
		_, _ = w.buf.Write(p[:room])
		w.truncated = true
		return len(p), nil
	}

	_, _ = w.buf.Write(p)
	return len(p), nil
}

func shellInvocation(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}

	return "/bin/sh", []string{"-c", command}
}

func execEnv() []string {
	if runtime.GOOS == "windows" {
		keys := []string{
			"SystemRoot",
			"ComSpec",
			"PATHEXT",
			"Path",
			"SystemDrive",
			"TEMP",
			"TMP",
			"USERPROFILE",
		}
		return allowEnv(keys)
	}

	keys := []string{
		"PATH",
		"HOME",
		"LANG",
		"LC_ALL",
		"LC_CTYPE",
		"TMPDIR",
		"TERM",
		"SHELL",
	}
	return allowEnv(keys)
}

func allowEnv(keys []string) []string {
	result := make([]string, 0, len(keys))

	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			result = append(result, key+"="+value)
		}
	}

	return result
}

type ExecCommandTool struct{}

func NewExecCommandTool() *ExecCommandTool {
	return &ExecCommandTool{}
}

func (t *ExecCommandTool) Name() string {
	return "exec_command"
}

func (t *ExecCommandTool) Description() string {
	return "Runs a shell command and returns combined stdout/stderr. " +
		"Commands matching common destructive operations are refused. " +
		"Execution has a timeout and output is limited."
}

func (t *ExecCommandTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to execute.",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory. Defaults to the current directory.",
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": "Maximum execution time in seconds. Defaults to 60; maximum 600.",
			},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}

func (t *ExecCommandTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		Command        string `json:"command"`
		Cwd            string `json:"cwd"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{
			Content: "error: invalid arguments: " + err.Error(),
			Error:   true,
		}, nil
	}

	command := strings.TrimSpace(input.Command)
	if command == "" {
		return ToolCallResult{
			Content: "error: command is required",
			Error:   true,
		}, nil
	}

	if reason := blockedReason(command); reason != "" {
		return ToolCallResult{
			Content: "error: refused to run command: " + reason,
			Error:   true,
		}, nil
	}

	cwd := "."
	if input.Cwd != "" {
		cwd = filepath.Clean(input.Cwd)
	}

	info, err := os.Stat(cwd)
	if err != nil {
		return ToolCallResult{
			Content: "error: invalid cwd: " + err.Error(),
			Error:   true,
		}, nil
	}

	if !info.IsDir() {
		return ToolCallResult{
			Content: "error: invalid cwd: not a directory: " + cwd,
			Error:   true,
		}, nil
	}

	timeout := defaultExecTimeout
	if input.TimeoutSeconds > 0 {
		timeout = min(time.Duration(input.TimeoutSeconds)*time.Second, maxExecTimeout)
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name, shellArgs := shellInvocation(command)

	cmd := exec.CommandContext(runCtx, name, shellArgs...)
	cmd.Dir = cwd
	cmd.Env = execEnv()

	// Keep your platform-specific implementation here.
	setPlatformAttrs(cmd)

	cmd.Cancel = func() error {
		return killProcessTree(cmd)
	}
	cmd.WaitDelay = 3 * time.Second

	var out limitedBuffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	start := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(start).Round(time.Millisecond)

	var b strings.Builder
	b.Write(out.buf.Bytes())

	if out.truncated {
		fmt.Fprintf(&b, "\n[output truncated to %d bytes]", maxExecOutputBytes)
	}

	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		fmt.Fprintf(&b, "\n[timed out after %s and was killed]", timeout)

		return ToolCallResult{
			Content: b.String(),
			Error:   true,
		}, nil
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		fmt.Fprintf(&b, "\n[cancelled and killed]")

		return ToolCallResult{
			Content: b.String(),
			Error:   true,
		}, nil
	}

	var exitErr *exec.ExitError

	switch {
	case runErr == nil:
		fmt.Fprintf(&b, "\n[exit code 0, took %s]", elapsed)

		return ToolCallResult{
			Content: b.String(),
			MainArg: command,
		}, nil

	case errors.As(runErr, &exitErr):
		fmt.Fprintf(&b, "\n[exit code %d, took %s]", exitErr.ExitCode(), elapsed)

		return ToolCallResult{
			Content: b.String(),
			MainArg: command,
			Error:   true,
		}, nil

	default:
		fmt.Fprintf(&b, "\n[failed to run: %s]", runErr)

		return ToolCallResult{
			Content: b.String(),
			MainArg: command,
			Error:   true,
		}, nil
	}
}
