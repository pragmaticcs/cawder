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

var envDenylistSubstrings = []string{
	"KEY", "TOKEN", "SECRET", "PASSWORD", "PASSWD", "CREDENTIAL", "AUTH", "APIKEY",
}

var envDenylistExact = map[string]bool{
	"SSH_AUTH_SOCK": true, "SSH_AGENT_PID": true, "GPG_AGENT_INFO": true,
}

func sanitizedEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, val, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(name)
		if envDenylistExact[upper] {
			continue
		}
		blocked := false
		for _, frag := range envDenylistSubstrings {
			if strings.Contains(upper, frag) {
				blocked = true
				break
			}
		}
		if !blocked {
			out = append(out, name+"="+val)
		}
	}
	return out
}

type namedPattern struct {
	re     *regexp.Regexp
	reason string
}

var forkBombPattern = regexp.MustCompile(`:\s*\(\)\s*\{\s*:\s*\|\s*:\s*&?\s*;?\s*\}\s*;\s*:`)

var destructivePatterns = []namedPattern{
	{regexp.MustCompile(`\bmkfs(\.\w+)?\b`), "formats a filesystem (mkfs)"},
	{regexp.MustCompile(`\bdd\b[^|;\n]*\bof=/dev/`), "writes raw data directly to a block device"},
	{regexp.MustCompile(`\b(fdisk|parted|sgdisk|wipefs)\b`), "modifies disk partitions"},
	{regexp.MustCompile(`\b(shutdown|reboot|halt|poweroff)\b`), "shuts down or restarts the machine"},
	{regexp.MustCompile(`\binit\s+[06]\b`), "changes the system runlevel to halt/reboot"},
	{regexp.MustCompile(`\bstop-computer\b|\brestart-computer\b`), "shuts down or restarts the machine (PowerShell)"},
	{regexp.MustCompile(`\bformat\s+[a-z]:`), "formats a Windows drive"},
	{regexp.MustCompile(`\bdiskpart\b`), "invokes the Windows disk-partitioning tool"},
	{regexp.MustCompile(`\b(del|erase|rd|rmdir)\b[^|;\n]*\b[a-z]:\\\s*("|')?(\s|$)`), "deletes an entire Windows drive root"},
	{regexp.MustCompile(`\bvssadmin\b[^|;\n]*\bdelete\b`), "deletes Windows shadow-copy backups"},
	{regexp.MustCompile(`\bbcdedit\b`), "modifies the Windows boot configuration"},
	{regexp.MustCompile(`\breg\s+delete\s+hklm`), "deletes a local-machine registry hive"},
}

var rootPaths = map[string]bool{
	"/": true, "/*": true, "~": true, "~/": true, "$home": true,
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

func hasDestructiveRemoveItem(lower string) bool {
	if !strings.Contains(lower, "remove-item") {
		return false
	}
	if !strings.Contains(lower, "-recurse") || !strings.Contains(lower, "-force") {
		return false
	}
	roots := []string{`c:\`, "c:/", "$env:systemdrive", "$env:windir", "$env:systemroot"}
	for _, r := range roots {
		if strings.Contains(lower, r) {
			return true
		}
	}
	return false
}

func blockedReason(rawCommand string) string {
	lower := strings.ToLower(rawCommand)

	if hasRecursiveRootOp(lower, []string{"rm"}, "rf") {
		return "destructive command: recursive force-delete of a root path (rm -rf / or similar)"
	}
	if hasRecursiveRootOp(lower, []string{"chmod", "chown"}, "r") {
		return "destructive command: recursive permission/ownership change on a root path"
	}
	if hasDestructiveRemoveItem(lower) {
		return "destructive command: recursive force-delete of a Windows system path"
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
		w.buf.Write(p[:room])
		w.truncated = true
		return len(p), nil
	}
	w.buf.Write(p)
	return len(p), nil
}

func shellInvocation(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "/bin/sh", []string{"-c", command}
}

type ExecCommandTool struct{}

func NewExecCommandTool() *ExecCommandTool {
	return &ExecCommandTool{}
}

func (t *ExecCommandTool) Name() string {
	return "exec_command"
}

func (t *ExecCommandTool) Description() string {
	return "Runs a shell command (`sh -c` on Linux/macOS, `cmd /C` on Windows) and returns its combined stdout/stderr. " +
		"The command is refused outright if it looks like a well-known destructive operation." +
		"Runs with a timeout and is killed (along with any child processes) if it hangs."
}

func (t *ExecCommandTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to run.",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory to run the command in (default: current directory).",
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("Max seconds to let the command run before it's killed (default %d, max %d).", int(defaultExecTimeout.Seconds()), int(maxExecTimeout.Seconds())),
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
		return ToolCallResult{Content: "Failed to parse arguments: " + err.Error(), Error: true}, nil
	}

	command := strings.TrimSpace(input.Command)
	if command == "" {
		return ToolCallResult{Content: "command is required", Error: true}, nil
	}

	if reason := blockedReason(command); reason != "" {
		return ToolCallResult{Content: "refused to run command: " + reason, Error: true}, nil
	}

	cwd := "."
	if input.Cwd != "" {
		cwd = filepath.Clean(input.Cwd)
	}
	if info, err := os.Stat(cwd); err != nil {
		return ToolCallResult{Content: "invalid cwd: " + err.Error(), Error: true}, nil
	} else if !info.IsDir() {
		return ToolCallResult{Content: "invalid cwd: not a directory: " + cwd, Error: true}, nil
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
	cmd.Env = sanitizedEnv()
	setPlatformAttrs(cmd)
	cmd.Cancel = func() error { return killProcessTree(cmd) }
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
		return ToolCallResult{Content: b.String(), Error: true}, nil
	}

	var exitErr *exec.ExitError
	switch {
	case runErr == nil:
		fmt.Fprintf(&b, "\n[exit code 0, took %s]", elapsed)
		return ToolCallResult{Content: b.String(), MainArg: command}, nil
	case errors.As(runErr, &exitErr):
		fmt.Fprintf(&b, "\n[exit code %d, took %s]", exitErr.ExitCode(), elapsed)
		return ToolCallResult{Content: b.String(), Error: true}, nil
	default:
		fmt.Fprintf(&b, "\n[failed to run: %s]", runErr.Error())
		return ToolCallResult{Content: b.String(), Error: true}, nil
	}
}
