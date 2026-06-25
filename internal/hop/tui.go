package hop

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	styleFilter   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleSelected = lipgloss.NewStyle().Reverse(true)
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleInfo     = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// ============================================
// Project Picker
// ============================================

type ProjectPickerModel struct {
	projects  []string
	filter    string
	cursor    int
	Selected  string
	Cancelled bool
	Done      bool
	initErr   error
}

func NewProjectPickerModel() ProjectPickerModel {
	entries, _ := os.ReadDir(ProjectsDir)
	projects := []string{"scratch"}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			projects = append(projects, entry.Name())
		}
	}
	sort.Strings(projects[1:])
	return ProjectPickerModel{projects: projects, cursor: 0, initErr: checkProjectsDir()}
}

func checkProjectsDir() error {
	info, err := os.Stat(ProjectsDir)
	if err != nil {
		return fmt.Errorf("projects directory %s not found: %w", ProjectsDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", ProjectsDir)
	}
	return nil
}

func NewCleanProjectPickerModel() ProjectPickerModel {
	m := NewProjectPickerModel()
	projects := make([]string, 0, len(m.projects)-1)
	for _, project := range m.projects {
		if project != "scratch" {
			projects = append(projects, project)
		}
	}
	m.projects = projects
	return m
}

func (m ProjectPickerModel) Init() tea.Cmd { return nil }

func (m ProjectPickerModel) View() string {
	if m.Done {
		return ""
	}

	var b strings.Builder
	b.WriteString(styleTitle.Render("hop — pick a project"))
	b.WriteByte('\n')

	if m.initErr != nil {
		b.WriteByte('\n')
		b.WriteString(styleError.Render("  " + m.initErr.Error()))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(styleDim.Render("press any key to quit"))
		return b.String()
	}

	b.WriteByte('\n')

	filtered := m.filtered()
	for i, name := range filtered {
		line := "  " + name
		if i == m.cursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(filterLine(m.filter))
	b.WriteByte('\n')

	return b.String()
}

func filterLine(filter string) string {
	if filter == "" {
		return styleFilter.Render("> type to search...")
	}
	return "> " + filter
}

func (m ProjectPickerModel) filtered() []string {
	return fuzzyFilter(m.filter, m.projects)
}

func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}
	query = strings.ToLower(query)
	target = strings.ToLower(target)
	i := 0
	for j := 0; j < len(target) && i < len(query); j++ {
		if query[i] == target[j] {
			i++
		}
	}
	return i == len(query)
}

func fuzzyFilter(query string, items []string) []string {
	if query == "" {
		return items
	}
	result := make([]string, 0)
	for _, item := range items {
		if fuzzyMatch(query, item) {
			result = append(result, item)
		}
	}
	return result
}

func (m ProjectPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.initErr != nil {
			m.Done = true
			return m, tea.Quit
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.Cancelled = true
			m.Done = true
			return m, tea.Quit
		case "enter":
			filtered := m.filtered()
			if len(filtered) > 0 {
				m.Selected = filtered[m.cursor]
				m.Done = true
				return m, tea.Quit
			}
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered())-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.cursor = 0
			}
		default:
			if len(msg.Runes) == 1 && msg.Runes[0] >= 32 && msg.Runes[0] != 127 {
				m.filter += string(msg.Runes[0])
				m.cursor = 0
			}
		}
	}
	return m, nil
}

// ============================================
// Branch Picker
// ============================================

type fetchMsg struct {
	remoteBranches []string
	err            error
}

type BranchPickerModel struct {
	projectPath   string
	projectName   string
	locals        []string
	remotes       []string
	fetched       bool
	fetching      bool
	fetchErr      error
	filter        string
	cursor        int
	Selected      string
	IsRemote      bool
	NewBranch     bool
	Cancelled     bool
	Done          bool
	forNewBranch  bool
	newBranchName string
}

func NewBranchPickerModel(projectName string) BranchPickerModel {
	repoPath := filepath.Join(ProjectsDir, projectName)
	local, _ := ListLocalBranches(repoPath)
	return BranchPickerModel{
		projectPath: repoPath,
		projectName: projectName,
		locals:      local,
		cursor:      0,
	}
}

func NewBaseBranchPickerModel(projectName, newBranchName string) BranchPickerModel {
	repoPath := filepath.Join(ProjectsDir, projectName)
	local, _ := ListLocalBranches(repoPath)
	return BranchPickerModel{
		projectPath:   repoPath,
		projectName:   projectName,
		locals:        local,
		cursor:        0,
		forNewBranch:  true,
		newBranchName: newBranchName,
	}
}

func (m BranchPickerModel) Init() tea.Cmd { return nil }

func (m BranchPickerModel) allBranches() []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(m.locals)+len(m.remotes))
	for _, branch := range m.locals {
		if !seen[branch] {
			seen[branch] = true
			result = append(result, branch)
		}
	}
	for _, branch := range m.remotes {
		if !seen[branch] {
			seen[branch] = true
			result = append(result, branch)
		}
	}
	return result
}

func (m BranchPickerModel) filtered() []string {
	branches := m.locals
	if m.fetched {
		branches = m.allBranches()
	}
	result := fuzzyFilter(m.filter, branches)
	if m.NewBranchOptionVisible() {
		result = append(result, "+ Create new branch")
	}
	return result
}

func (m BranchPickerModel) NewBranchOptionVisible() bool {
	return !m.forNewBranch && m.filter != ""
}

func (m BranchPickerModel) View() string {
	if m.Done {
		return ""
	}

	var b strings.Builder

	if m.forNewBranch {
		b.WriteString(styleTitle.Render(fmt.Sprintf("hop — %s — base for %s", m.projectName, m.newBranchName)))
	} else {
		b.WriteString(styleTitle.Render("hop — " + m.projectName + " — pick a branch"))
	}
	b.WriteByte('\n')
	b.WriteByte('\n')

	filtered := m.filtered()

	if m.fetching {
		b.WriteString(styleDim.Render("  Fetching remotes..."))
		b.WriteByte('\n')
	}

	if m.fetchErr != nil {
		b.WriteString(styleError.Render("  " + m.fetchErr.Error()))
		b.WriteByte('\n')
	}

	for i, entry := range filtered {
		isNewBranchOption := entry == "+ Create new branch"

		line := "  " + entry
		if isNewBranchOption {
			line = styleDim.Render(line)
		}
		if i == m.cursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(filterLine(m.filter))
	if m.fetched {
		b.WriteString(styleDim.Render("  (local + remote)"))
	} else {
		b.WriteString(styleDim.Render("  (local only — type to fetch remotes)"))
	}
	b.WriteByte('\n')

	return b.String()
}

func (m BranchPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchMsg:
		m.fetching = false
		m.fetched = true
		if msg.err != nil {
			m.fetchErr = msg.err
		} else {
			m.remotes = msg.remoteBranches
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Cancelled = true
			m.Done = true
			return m, tea.Quit
		case "enter":
			filtered := m.filtered()
			if len(filtered) == 0 {
				return m, nil
			}
			selected := filtered[m.cursor]
			if selected == "+ Create new branch" {
				m.NewBranch = true
				m.Done = true
				return m, tea.Quit
			}
			m.Selected = selected
			m.IsRemote = false
			foundInLocals := false
			for _, local := range m.locals {
				if local == selected {
					foundInLocals = true
					break
				}
			}
			if !foundInLocals {
				for _, remote := range m.remotes {
					if remote == selected {
						m.IsRemote = true
						break
					}
				}
			}
			m.Done = true
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered())-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.cursor = 0
				if !m.fetched && !m.fetching && m.filter != "" {
					localMatches := fuzzyFilter(m.filter, m.locals)
					if len(localMatches) == 0 {
						return m, m.fetchCmd()
					}
				}
			}
		default:
			if len(msg.Runes) == 1 && msg.Runes[0] >= 32 && msg.Runes[0] != 127 {
				m.filter += string(msg.Runes[0])
				m.cursor = 0

				if !m.fetched && !m.fetching {
					localMatches := fuzzyFilter(m.filter, m.locals)
					if len(localMatches) == 0 {
						m.fetching = true
						return m, m.fetchCmd()
					}
				}
			}
		}
	}
	return m, nil
}

func (m BranchPickerModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		repoPath := m.projectPath

		if err := FetchRemotes(repoPath); err != nil {
			return fetchMsg{err: err}
		}

		remote, err := ListRemoteBranches(repoPath)
		if err != nil {
			return fetchMsg{err: err}
		}

		return fetchMsg{remoteBranches: RemoveRemotePrefix(remote)}
	}
}

// ============================================
// Text Input
// ============================================

type InputModel struct {
	prompt    string
	value     string
	pos       int
	Result    string
	Cancelled bool
	Done      bool
}

func NewInputModel(prompt string) InputModel {
	return InputModel{prompt: prompt, pos: 0}
}

func (m InputModel) Init() tea.Cmd { return nil }

func (m InputModel) View() string {
	if m.Done {
		return ""
	}

	var b strings.Builder
	b.WriteString(styleTitle.Render("hop — " + m.prompt))
	b.WriteByte('\n')
	b.WriteByte('\n')

	b.WriteString("> " + m.value + "█")
	b.WriteByte('\n')
	b.WriteByte('\n')

	b.WriteString(styleDim.Render("enter: confirm  |  esc: cancel"))
	b.WriteByte('\n')

	return b.String()
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Cancelled = true
			m.Done = true
			return m, tea.Quit
		case "enter":
			if strings.TrimSpace(m.value) != "" {
				m.Result = strings.TrimSpace(m.value)
				m.Done = true
				return m, tea.Quit
			}
		case "backspace":
			if len(m.value) > 0 && m.pos > 0 {
				m.value = m.value[:m.pos-1] + m.value[m.pos:]
				m.pos--
			}
		case "left", "ctrl+b":
			if m.pos > 0 {
				m.pos--
			}
		case "right", "ctrl+f":
			if m.pos < len(m.value) {
				m.pos++
			}
		case "ctrl+a":
			m.pos = 0
		case "ctrl+e":
			m.pos = len(m.value)
		default:
			if len(msg.Runes) == 1 && msg.Runes[0] >= 32 && msg.Runes[0] != 127 {
				m.value = m.value[:m.pos] + string(msg.Runes[0]) + m.value[m.pos:]
				m.pos++
			}
		}
	}
	return m, nil
}

// ============================================
// Clean (Worktree) Picker
// ============================================

const (
	cleanStepPick = iota
	cleanStepKillSession
	cleanStepConfirm
	cleanStepDone
)

type CleanWorktreePickerModel struct {
	projectName  string
	projectPath  string
	worktrees    []string
	filter       string
	cursor       int
	Selected     string
	Cancelled    bool
	Done         bool
	step         int
	message      string
	worktreePath string
}

func NewCleanWorktreePickerModel(projectName string) CleanWorktreePickerModel {
	repoPath := filepath.Join(ProjectsDir, projectName)
	worktrees, err := findHopWorktrees(projectName)
	if err != nil {
		worktrees = nil
	}
	return CleanWorktreePickerModel{
		projectName: projectName,
		projectPath: repoPath,
		worktrees:   worktrees,
		step:        cleanStepPick,
	}
}

func findHopWorktrees(projectName string) ([]string, error) {
	projDir := filepath.Join(WorktreeDir, capitalCase(projectName))
	entries, err := os.ReadDir(projDir)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, entry.Name())
		}
	}
	sort.Strings(result)
	return result, nil
}

func (m CleanWorktreePickerModel) Init() tea.Cmd { return nil }

func (m CleanWorktreePickerModel) View() string {
	if m.Done {
		return ""
	}

	var b strings.Builder
	b.WriteString(styleTitle.Render("hop — " + m.projectName + " — clean"))
	b.WriteByte('\n')
	b.WriteByte('\n')

	if m.step == cleanStepDone {
		b.WriteString(styleError.Render("  " + m.message))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(styleDim.Render("press any key to quit"))
		b.WriteByte('\n')
		return b.String()
	}

	if m.step == cleanStepConfirm {
		b.WriteString("  " + m.message)
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(styleDim.Render("enter: confirm  |  esc: cancel"))
		b.WriteByte('\n')
		return b.String()
	}

	if m.step == cleanStepKillSession {
		b.WriteString(styleInfo.Render("  " + m.message))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(styleDim.Render("enter: kill session  |  esc: cancel"))
		b.WriteByte('\n')
		return b.String()
	}

	if m.step == cleanStepPick && len(m.worktrees) == 0 {
		b.WriteString(styleDim.Render("  No hop worktrees found for this project"))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(styleDim.Render("press any key to go back"))
		b.WriteByte('\n')
		return b.String()
	}

	filtered := fuzzyFilter(m.filter, m.worktrees)
	for i, worktree := range filtered {
		line := "  " + worktree
		if i == m.cursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(filterLine(m.filter))
	b.WriteByte('\n')

	return b.String()
}

func (m CleanWorktreePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.step == cleanStepDone {
			m.Done = true
			return m, tea.Quit
		}

		if m.step == cleanStepPick && len(m.worktrees) == 0 {
			m.Cancelled = true
			m.Done = true
			return m, tea.Quit
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.Cancelled = true
			m.Done = true
			return m, tea.Quit
		case "enter":
			if m.step == cleanStepPick {
				filtered := fuzzyFilter(m.filter, m.worktrees)
				if len(filtered) == 0 {
					return m, nil
				}
				m.Selected = filtered[m.cursor]
				m.worktreePath = WorktreePath(m.projectName, m.Selected)

				sessionName := SessionName(m.projectName, m.Selected)
				if SessionExists(sessionName) {
					m.message = fmt.Sprintf("Session '%s' is active. Kill it first?", sessionName)
					m.step = cleanStepKillSession
					return m, nil
				}

				dirty, err := HasUncommittedChanges(m.worktreePath)
				if err != nil {
					m.message = fmt.Sprintf("Error checking status: %v", err)
					m.step = cleanStepDone
					return m, nil
				}
				if dirty {
					m.message = "Worktree has uncommitted changes. Commit or stash them first."
					m.step = cleanStepDone
					return m, nil
				}

				m.message = fmt.Sprintf("Delete worktree and branch '%s'?", m.Selected)
				m.step = cleanStepConfirm
				return m, nil
			}

			if m.step == cleanStepKillSession {
				KillSession(SessionName(m.projectName, m.Selected))

				dirty, err := HasUncommittedChanges(m.worktreePath)
				if err != nil {
					m.message = fmt.Sprintf("Error checking status: %v", err)
					m.step = cleanStepDone
					return m, nil
				}
				if dirty {
					m.message = "Worktree has uncommitted changes. Commit or stash them first."
					m.step = cleanStepDone
					return m, nil
				}

				m.message = fmt.Sprintf("Delete worktree and branch '%s'?", m.Selected)
				m.step = cleanStepConfirm
				return m, nil
			}

			if m.step == cleanStepConfirm {
				if err := RemoveWorktree(m.projectPath, m.worktreePath); err != nil {
					m.message = fmt.Sprintf("Failed to remove worktree: %v", err)
					m.step = cleanStepDone
					return m, nil
				}
				if err := DeleteBranch(m.projectPath, m.Selected); err != nil {
					m.message = fmt.Sprintf("Worktree removed, but failed to delete branch: %v", err)
					m.step = cleanStepDone
					return m, nil
				}
				m.message = fmt.Sprintf("Deleted worktree and branch '%s'.", m.Selected)
				m.step = cleanStepDone
				return m, nil
			}

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(fuzzyFilter(m.filter, m.worktrees))-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.cursor = 0
			}
		default:
			if m.step == cleanStepPick && len(msg.Runes) == 1 && msg.Runes[0] >= 32 && msg.Runes[0] != 127 {
				m.filter += string(msg.Runes[0])
				m.cursor = 0
			}
		}
	}
	return m, nil
}
