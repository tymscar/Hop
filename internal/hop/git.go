package hop

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func ListLocalBranches(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--format=%(refname:short)")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing local branches: %w", err)
	}
	branches := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(branches) == 1 && branches[0] == "" {
		return nil, nil
	}
	return branches, nil
}

func FetchRemotes(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "fetch", "--all", "--quiet")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fetching remotes: %w", err)
	}
	return nil
}

func ListRemoteBranches(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "-r", "--format=%(refname:short)")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing remote branches: %w", err)
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if _, branch, isBranch := strings.Cut(line, "/"); isBranch && branch != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

func RemoveRemotePrefix(branches []string) []string {
	result := make([]string, 0, len(branches))
	seen := make(map[string]bool)
	for _, branch := range branches {
		cleaned := SanitizeBranch(branch)
		if !seen[cleaned] {
			seen[cleaned] = true
			result = append(result, cleaned)
		}
	}
	return result
}

func FindWorktree(repoPath, branch string) (string, bool) {
	worktrees, _ := ListWorktrees(repoPath)
	for _, worktree := range worktrees {
		if worktree.Branch == branch {
			return worktree.Path, true
		}
	}
	return "", false
}

type WorktreeInfo struct {
	Path   string
	Branch string
}

func ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo
	flush := func() {
		if current.Path == "" {
			return
		}
		worktrees = append(worktrees, current)
		current = WorktreeInfo{}
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			current.Branch = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "":
			flush()
		}
	}
	flush()

	return worktrees, nil
}

func CreateWorktree(repoPath, branch, worktreePath string, isRemote bool) error {
	var cmd *exec.Cmd
	if isRemote {
		cmd = exec.Command("git", "-C", repoPath, "worktree", "add", "-b", branch, worktreePath, "origin/"+branch)
	} else {
		cmd = exec.Command("git", "-C", repoPath, "worktree", "add", worktreePath, branch)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}
	return nil
}

func HasUncommittedChanges(repoPath string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("checking git status: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func RemoveWorktree(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "remove", worktreePath, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("removing worktree: %s", string(out))
	}
	return nil
}

func DeleteBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "-C", repoPath, "branch", "-D", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("deleting branch: %s", string(out))
	}
	return nil
}

func CreateBranch(repoPath, branch, base string) error {
	cmd := exec.Command("git", "-C", repoPath, "branch", branch, base)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating branch: %s", string(out))
	}
	return nil
}

func IsBranchCheckedOutInMain(repoPath, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("getting HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)) == branch, nil
}
