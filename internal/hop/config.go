package hop

import (
	"os"
	"path/filepath"
	"strings"
)

var homeDir = func() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return dir
}()

var (
	WorktreeDir = filepath.Join(homeDir, ".local", "share", "hop", "worktrees")
	ProjectsDir = filepath.Join(homeDir, "Projects")
)

func SessionName(project, branch string) string {
	proj := strings.ToLower(project)
	b := SanitizeBranch(branch)
	return proj + "-" + b
}

func WorktreePath(project, branch string) string {
	proj := capitalCase(project)
	b := SanitizeBranch(branch)
	return filepath.Join(WorktreeDir, proj, b)
}

func SanitizeBranch(branch string) string {
	b := strings.TrimSpace(branch)
	b = strings.TrimPrefix(b, "origin/")
	b = strings.ReplaceAll(b, "/", "-")
	return b
}

func StatusLabel(project, branch string) string {
	return project + "/" + branch
}

func capitalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
