package hop

import (
	"fmt"
	"os"
	"os/exec"
)

func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

func SessionExists(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

func CreateProjectSession(name, cwd, project, branch string) error {
	create := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", cwd, "-n", "ai")
	if out, err := create.CombinedOutput(); err != nil {
		return fmt.Errorf("creating session: %s", string(out))
	}

	if err := exec.Command("tmux", "new-window", "-t", name, "-n", "term", "-c", cwd).Run(); err != nil {
		return fmt.Errorf("creating term window: %w", err)
	}

	if err := exec.Command("tmux", "new-window", "-t", name, "-n", "dev", "-c", cwd).Run(); err != nil {
		return fmt.Errorf("creating dev window: %w", err)
	}
	if err := exec.Command("tmux", "split-window", "-h", "-t", name+":dev", "-c", cwd).Run(); err != nil {
		return fmt.Errorf("splitting dev window: %w", err)
	}
	exec.Command("tmux", "select-layout", "-t", name+":dev", "even-horizontal").Run()

	if err := exec.Command("tmux", "new-window", "-t", name, "-n", "git", "-c", cwd).Run(); err != nil {
		return fmt.Errorf("creating git window: %w", err)
	}

	exec.Command("tmux", "set-option", "-t", name, "status-left", StatusLabel(project, branch)).Run()
	exec.Command("tmux", "send-keys", "-t", name+":ai", "opencode", "Enter").Run()
	exec.Command("tmux", "send-keys", "-t", name+":git", "lazygit", "Enter").Run()
	exec.Command("tmux", "select-window", "-t", name+":ai").Run()

	return nil
}

func CreateScratchSession(name string) error {
	create := exec.Command("tmux", "new-session", "-d", "-s", name, "-n", "scratch")
	if out, err := create.CombinedOutput(); err != nil {
		return fmt.Errorf("creating scratch session: %s", string(out))
	}
	exec.Command("tmux", "set-option", "-t", name, "status-left", "scratch").Run()
	return nil
}

func SwitchToSession(name string) error {
	verb := "attach-session"
	if InTmux() {
		verb = "switch-client"
	}
	if out, err := exec.Command("tmux", verb, "-t", name).CombinedOutput(); err != nil {
		return fmt.Errorf("switching to session: %s", string(out))
	}
	return nil
}

func KillSession(name string) error {
	if out, err := exec.Command("tmux", "kill-session", "-t", name).CombinedOutput(); err != nil {
		return fmt.Errorf("killing session: %s", string(out))
	}
	return nil
}
