package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbletea"

	"github.com/tymscar/hop/internal/hop"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "clean" {
		runClean()
		return
	}
	runOpen()
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "hop: %v\n", err)
	os.Exit(1)
}

func runPicker(model tea.Model) tea.Model {
	result, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		fail(err)
	}
	return result
}

func runOpen() {
	project := runPicker(hop.NewProjectPickerModel()).(hop.ProjectPickerModel)
	if project.Cancelled {
		return
	}

	if project.Selected == "scratch" {
		openScratch()
		return
	}

	branch := runPicker(hop.NewBranchPickerModel(project.Selected)).(hop.BranchPickerModel)
	if branch.Cancelled {
		return
	}
	if branch.NewBranch {
		runNewBranchFlow(project.Selected)
		return
	}

	openBranch(project.Selected, branch.Selected, branch.IsRemote)
}

func openScratch() {
	if !hop.SessionExists("scratch") {
		if err := hop.CreateScratchSession("scratch"); err != nil {
			fail(err)
		}
	}
	if err := hop.SwitchToSession("scratch"); err != nil {
		fail(err)
	}
}

func runNewBranchFlow(projectName string) {
	nameInput := runPicker(hop.NewInputModel("new branch name")).(hop.InputModel)
	if nameInput.Cancelled {
		return
	}
	newBranch := nameInput.Result

	baseBranch := runPicker(hop.NewBaseBranchPickerModel(projectName, newBranch)).(hop.BranchPickerModel)
	if baseBranch.Cancelled {
		return
	}

	repoPath := filepath.Join(hop.ProjectsDir, projectName)
	if err := hop.CreateBranch(repoPath, newBranch, baseBranch.Selected); err != nil {
		fail(err)
	}

	openBranch(projectName, newBranch, false)
}

func openBranch(projectName, branch string, isRemote bool) {
	sessionName := hop.SessionName(projectName, branch)

	if hop.SessionExists(sessionName) {
		if err := hop.SwitchToSession(sessionName); err != nil {
			fail(err)
		}
		return
	}

	repoPath := filepath.Join(hop.ProjectsDir, projectName)

	if worktreePath, found := hop.FindWorktree(repoPath, branch); found {
		if err := hop.CreateProjectSession(sessionName, worktreePath, projectName, branch); err != nil {
			fail(err)
		}
		if err := hop.SwitchToSession(sessionName); err != nil {
			fail(err)
		}
		return
	}

	checkedOutInMain, err := hop.IsBranchCheckedOutInMain(repoPath, branch)
	if err != nil {
		fail(err)
	}
	if checkedOutInMain {
		fmt.Fprintf(os.Stderr, "hop: '%s' is checked out in %s.\n", branch, repoPath)
		fmt.Fprintf(os.Stderr, "Switch that checkout to main first, then try again.\n")
		os.Exit(1)
	}

	worktreePath := hop.WorktreePath(projectName, branch)
	fmt.Printf("hop: creating worktree for %s (large repos may take a while)…\n", branch)
	if err := hop.CreateWorktree(repoPath, branch, worktreePath, isRemote); err != nil {
		fail(err)
	}
	if err := hop.CreateProjectSession(sessionName, worktreePath, projectName, branch); err != nil {
		fail(err)
	}
	if err := hop.SwitchToSession(sessionName); err != nil {
		fail(err)
	}
}

func runClean() {
	project := runPicker(hop.NewCleanProjectPickerModel()).(hop.ProjectPickerModel)
	if project.Cancelled {
		return
	}
	runPicker(hop.NewCleanWorktreePickerModel(project.Selected))
}
