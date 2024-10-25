package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF7CCB")).
			Padding(0, 1).
			AlignHorizontal(lipgloss.Center).Border(lipgloss.NormalBorder())
	tab = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777")).
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(lipgloss.Color("#7D5674")).
		Padding(0, 1)
	activeTab = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(lipgloss.Color("#7D5674")).
			Padding(0, 1)
	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(1, 2)
	helpStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#444444")).
			Foreground(lipgloss.Color("#FFFFFF"))
	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#444444")).
			Foreground(lipgloss.Color("#00FF00"))
	helpSeparatorStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#444444")).
				Foreground(lipgloss.Color("#888888"))
)

const (
	Errors = iota
	Warnings
	Information
)

type Log struct {
	timestamp string
	message   string
}

type focusedInput int

const (
	logFocus focusedInput = iota
	searchBoxFocused
	startDateFocused
	endDateFocused
)

type model struct {
	width        int
	height       int
	activeTab    int
	focused      focusedInput
	searchBox    textinput.Model
	startDate    textinput.Model
	endDate      textinput.Model
	searchQuery  string
	errors       []Log
	warnings     []Log
	info         []Log
	filteredLogs []Log
	logTable     table.Model
}

func (m *model) Init() tea.Cmd {
	// Initialize tables
	m.initLogTable()
	// m.initHelpTable()
	return tea.EnterAltScreen
}

func (m *model) initLogTable() {
	columns := []table.Column{
		{Title: "Timestamp", Width: 20},
		{Title: "Message", Width: m.width - 22}, // Remaining width for message
	}

	// Convert filtered logs to table rows
	rows := make([]table.Row, len(m.filteredLogs))
	for i, log := range m.filteredLogs {
		rows[i] = table.Row{log.timestamp, log.message}
	}

	m.logTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(10),
		table.WithFocused(m.focused == logFocus),
	)
}

func (m model) renderHelpFooter() string {
	var help strings.Builder

	// Define help items
	helpItems := []struct {
		key         string
		description string
	}{
		{"^Q", "Exit"},
		{"Tab", "Switch Tab"},
		{"/", "Search"},
		{"F", "Start Date"},
		{"E", "End Date"},
		{"^C", "Cancel"},
		{"Enter", "Apply"},
	}

	separator := helpSeparatorStyle.Render(" | ")

	// Create the help line
	columnWidth := 15
	width := 0
	help.WriteString(helpStyle.Render("  "))
	for i, item := range helpItems {
		if i > 0 {
			help.WriteString(separator)
		}
		if width > m.width {
			help.WriteString("\n")
		}
		help.WriteString(helpKeyStyle.Render(item.key))
		help.WriteString(helpStyle.Render(" " + item.description))
		width += columnWidth
	}

	// Create border line
	width = m.width
	if width == 0 {
		width = 80 // fallback width
	}

	// Pad the help text to full width
	helpText := help.String()
	padding := width - lipgloss.Width(helpText)
	if padding > 0 {
		helpText += helpStyle.Render(strings.Repeat(" ", padding))
	}

	return helpText
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if m.focused == logFocus {
				return m, tea.Quit
			}
		case "tab":
			m.activeTab = (m.activeTab + 1) % 3
			m.applyFilters() // Update filtered logs for new tab
			m.initLogTable() // Reinitialize table with new data
		case "shift+tab":
			m.activeTab = (m.activeTab + 3 - 1) % 3
			m.applyFilters() // Update filtered logs for new tab
			m.initLogTable() // Reinitialize table with new data
		case "/":
			if m.focused == logFocus {
				m.focused = searchBoxFocused
				m.searchBox.Focus()
				m.startDate.Blur()
				m.endDate.Blur()
			}
		case "f":
			if m.focused == logFocus {
				m.focused = startDateFocused
				m.startDate.Focus()
				m.searchBox.Blur()
				m.endDate.Blur()
			}
		case "e":
			if m.focused == logFocus {
				m.focused = endDateFocused
				m.endDate.Focus()
				m.searchBox.Blur()
				m.startDate.Blur()
			}
		case "esc":
			m.clearFocusedFilter()
			m.focused = logFocus
			m.searchBox.Blur()
			m.startDate.Blur()
			m.endDate.Blur()
			m.initLogTable() // Reinitialize table after clearing filter
		case "enter":
			if m.focused == searchBoxFocused || m.focused == startDateFocused || m.focused == endDateFocused {
				m.applyFilters()
				m.initLogTable() // Reinitialize table after applying filters
				m.focused = logFocus
			}
		}

		// Handle table navigation when focused on logs
		if m.focused == logFocus {
			var tableMsg tea.Msg = msg
			m.logTable, cmd = m.logTable.Update(tableMsg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initLogTable() // Reinitialize table with new dimensions
		return m, tea.ClearScreen
	}

	switch m.focused {
	case searchBoxFocused:
		m.searchBox, cmd = m.searchBox.Update(msg)
	case startDateFocused:
		m.startDate, cmd = m.startDate.Update(msg)
	case endDateFocused:
		m.endDate, cmd = m.endDate.Update(msg)
	}

	m.searchQuery = m.searchBox.Value()
	return m, cmd
}

func (m model) View() string {
	content := strings.Builder{}

	// Title
	title := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, titleStyle.Render("System Log Analyzer"))
	content.WriteString(title + "\n\n")

	// Tab bar
	tabBar := ""
	switch m.activeTab {
	case Errors:
		tabBar = lipgloss.JoinHorizontal(lipgloss.Top,
			activeTab.Render("Errors"),
			tab.Render("Warnings"),
			tab.Render("Information"))
	case Warnings:
		tabBar = lipgloss.JoinHorizontal(lipgloss.Top,
			tab.Render("Errors"),
			activeTab.Render("Warnings"),
			tab.Render("Information"))
	case Information:
		tabBar = lipgloss.JoinHorizontal(lipgloss.Top,
			tab.Render("Errors"),
			tab.Render("Warnings"),
			activeTab.Render("Information"))
	}
	content.WriteString(tabBar + "\n\n")

	// Search and date filters
	content.WriteString("Search: " + m.searchBox.View() + "\n\n")
	content.WriteString("Start Date (YYYY-MM-DD): " + m.startDate.View() + "\n")
	content.WriteString("End Date (YYYY-MM-DD): " + m.endDate.View() + "\n\n")

	// Log table
	content.WriteString("\nLogs:\n")
	content.WriteString(m.logTable.View())

	// Help table
	content.WriteString("\nHelp:\n")
	content.WriteString(m.renderHelpFooter())

	return content.String()
}

func filterLogs(logs []Log, query, start, end string) []Log {
	var result []Log
	for _, log := range logs {
		if query != "" && !strings.Contains(strings.ToLower(log.message), strings.ToLower(query)) {
			continue
		}
		if start != "" && log.timestamp < start {
			continue
		}
		if end != "" && log.timestamp > end {
			continue
		}
		result = append(result, log)
	}
	return result
}

func (m *model) clearFocusedFilter() {
	switch m.focused {
	case searchBoxFocused:
		m.searchBox.SetValue("")
	case startDateFocused:
		m.startDate.SetValue("")
	case endDateFocused:
		m.endDate.SetValue("")
	}
	m.applyFilters()
}

func (m *model) applyFilters() {
	var logs []Log
	switch m.activeTab {
	case Errors:
		logs = m.errors
	case Warnings:
		logs = m.warnings
	case Information:
		logs = m.info
	}
	m.filteredLogs = filterLogs(logs, m.searchBox.Value(), m.startDate.Value(), m.endDate.Value())
}

func main() {
	searchBox := textinput.New()
	searchBox.Placeholder = "Enter keyword"
	searchBox.Width = 30

	startDate := textinput.New()
	startDate.Placeholder = "YYYY-MM-DD"
	startDate.Width = 12

	endDate := textinput.New()
	endDate.Placeholder = "YYYY-MM-DD"
	endDate.Width = 12

	m := model{
		searchBox: searchBox,
		startDate: startDate,
		endDate:   endDate,
		errors: []Log{
			{timestamp: "2024-10-01", message: "authentication failure"},
			{timestamp: "2024-10-05", message: "out of memory"},
		},
		warnings: []Log{
			{timestamp: "2024-10-02", message: "disk usage high"},
			{timestamp: "2024-10-06", message: "CPU usage high"},
		},
		info: []Log{
			{timestamp: "2024-10-01", message: "service started"},
			{timestamp: "2024-10-04", message: "configuration loaded"},
		},
	}

	m.applyFilters() // Initialize filtered logs

	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}
