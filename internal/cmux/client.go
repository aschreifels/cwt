package cmux

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func run(args ...string) (string, error) {
	cmd := exec.Command("cmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("cmux %s: %w: %s", args[0], err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

type WorkspaceResult struct {
	ID string
}

func NewWorkspace(command string) (WorkspaceResult, error) {
	out, err := run("new-workspace", "--command", command)
	if err != nil {
		return WorkspaceResult{}, err
	}
	parts := strings.Fields(out)
	if len(parts) >= 2 && parts[0] == "OK" {
		return WorkspaceResult{ID: parts[1]}, nil
	}
	return WorkspaceResult{}, fmt.Errorf("unexpected output: %s", out)
}

func SelectWorkspace(wsID string) error {
	_, err := run("select-workspace", "--workspace", wsID)
	return err
}

func RenameWorkspace(wsID, name string) error {
	_, err := run("rename-workspace", "--workspace", wsID, name)
	return err
}

func CloseWorkspace(wsID string) error {
	_, err := run("close-workspace", "--workspace", wsID)
	return err
}

func NewSplit(direction, wsID string) (string, error) {
	out, err := run("new-split", direction, "--workspace", wsID)
	if err != nil {
		return "", err
	}
	return parseSurface(out), nil
}

func NewSplitOnPanel(direction, wsID, panelRef string) (string, error) {
	out, err := run("new-split", direction, "--workspace", wsID, "--panel", panelRef)
	if err != nil {
		return "", err
	}
	return parseSurface(out), nil
}

func ListPaneSurfaces(wsID string) ([]string, error) {
	out, err := run("list-pane-surfaces", "--workspace", wsID)
	if err != nil {
		return nil, err
	}
	var surfaces []string
	for _, line := range strings.Split(out, "\n") {
		for _, field := range strings.Fields(line) {
			if strings.HasPrefix(field, "surface:") {
				surfaces = append(surfaces, field)
			}
		}
	}
	return surfaces, nil
}

func SendText(wsID, surfaceRef, text string) error {
	_, err := run("send", "--workspace", wsID, "--surface", surfaceRef, text)
	return err
}

func SendPanel(wsID, panelRef, text string) error {
	_, err := run("send-panel", "--workspace", wsID, "--panel", panelRef, text)
	return err
}

func SendKeyPanel(wsID, panelRef, key string) error {
	_, err := run("send-key-panel", "--workspace", wsID, "--panel", panelRef, key)
	return err
}

func ReadScreen(wsID, surfaceRef string, lines int) (string, error) {
	return run("read-screen", "--workspace", wsID, "--surface", surfaceRef, "--lines", fmt.Sprintf("%d", lines))
}

func SetStatus(wsID, key, value, icon, color string) error {
	args := []string{"set-status", key, value, "--workspace", wsID}
	if icon != "" {
		args = append(args, "--icon", icon)
	}
	if color != "" {
		args = append(args, "--color", color)
	}
	_, err := run(args...)
	return err
}

func ClearStatus(wsID, key string) error {
	_, err := run("clear-status", key, "--workspace", wsID)
	return err
}

func SetProgress(wsID string, progress float64, label string) error {
	args := []string{"set-progress", fmt.Sprintf("%.2f", progress), "--workspace", wsID}
	if label != "" {
		args = append(args, "--label", label)
	}
	_, err := run(args...)
	return err
}

func ClearProgress(wsID string) error {
	_, err := run("clear-progress", "--workspace", wsID)
	return err
}

func Notify(wsID, title, body string) error {
	args := []string{"notify", "--title", title}
	if body != "" {
		args = append(args, "--body", body)
	}
	if wsID != "" {
		args = append(args, "--workspace", wsID)
	}
	_, err := run(args...)
	return err
}

func Log(wsID, level, source, message string) error {
	args := []string{"log", "--level", level, "--source", source, "--workspace", wsID, "--", message}
	_, err := run(args...)
	return err
}

func ListWorkspaces() (string, error) {
	return run("list-workspaces")
}

func FindWorkspaceByName(name string) string {
	out, err := ListWorkspaces()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == name && i > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

func WaitForReady(wsID, surfaceRef string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		screen, err := ReadScreen(wsID, surfaceRef, 5)
		if err == nil && (strings.Contains(screen, ">") || strings.Contains(screen, "crush")) {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func parseSurface(out string) string {
	for _, field := range strings.Fields(out) {
		if strings.HasPrefix(field, "surface:") {
			return field
		}
	}
	return ""
}
