package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/N-Erickson/termidar/internal/config"
	"github.com/N-Erickson/termidar/internal/geography"
	"github.com/N-Erickson/termidar/internal/radar"
	"github.com/N-Erickson/termidar/internal/weather"
)

// Model states
type State int

const (
	StateInput State = iota
	StateLoading
	StateDisplaying
	StateError
)

// Model represents the application state
type Model struct {
	state               State
	zipInput            textinput.Model
	spinner             spinner.Model
	progress            progress.Model
	radar               radar.Data
	currentFrame        int
	width               int
	height              int
	errorMsg            string
	showHelp            bool
	isPaused            bool
	frameRate           time.Duration
	lastRefresh         time.Time
	autoRefresh         bool
	zipCode             string
	animationActive     bool
	isBackgroundRefresh bool
}

// Messages
type TickMsg time.Time
type FrameTickMsg time.Time
type RefreshTickMsg time.Time
type ErrorMsg struct {
	Err error
}
type ProgressMsg float64

// InitialModel creates and returns a new model
func InitialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter ZIP code"
	ti.Focus()
	ti.CharLimit = 5
	ti.Width = 20
	ti.Prompt = "📍 "

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(config.SecondaryColor)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return Model{
		state:           StateInput,
		zipInput:        ti,
		spinner:         s,
		progress:        p,
		width:           80,
		height:          40,
		frameRate:       300 * time.Millisecond,
		autoRefresh:     true,
		animationActive: false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state == StateDisplaying || m.state == StateError {
				m.animationActive = false
				m = m.ResetToInput()
				return m, textinput.Blink
			}
		case "enter":
			if m.state == StateInput && len(m.zipInput.Value()) == 5 {
				m.state = StateLoading
				m.zipCode = m.zipInput.Value()
				cmds = append(cmds,
					m.spinner.Tick,
					radar.LoadData(m.zipCode),
					m.TrackProgress(),
				)
			}
		case "?", "h":
			m.showHelp = !m.showHelp
		case " ":
			if m.state == StateDisplaying {
				m.isPaused = !m.isPaused
				if !m.isPaused && !m.animationActive {
					m.animationActive = true
					cmds = append(cmds, m.AnimateFrame())
				}
			}
		case "r":
			if m.state == StateDisplaying && m.zipCode != "" {
				m.animationActive = false
				m.state = StateLoading
				cmds = append(cmds,
					m.spinner.Tick,
					radar.LoadData(m.zipCode),
					m.TrackProgress(),
				)
			}
		case "left", "a":
			if m.state == StateDisplaying && len(m.radar.Frames) > 0 {
				m.currentFrame = (m.currentFrame - 1 + len(m.radar.Frames)) % len(m.radar.Frames)
			}
		case "right", "d":
			if m.state == StateDisplaying && len(m.radar.Frames) > 0 {
				m.currentFrame = (m.currentFrame + 1) % len(m.radar.Frames)
			}
		case "+", "=":
			if m.frameRate > 100*time.Millisecond {
				m.frameRate -= 100 * time.Millisecond
			}
		case "-", "_":
			if m.frameRate < 2*time.Second {
				m.frameRate += 100 * time.Millisecond
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		if m.state == StateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ProgressMsg:
		if m.state == StateLoading {
			cmd := m.progress.SetPercent(float64(msg))
			cmds = append(cmds, cmd)
		}

	case radar.LoadedMsg:
		// oldRadar := m.radar
		m.radar = msg.Radar

		// If this is a background refresh, preserve the animation state
		if m.state == StateDisplaying && m.isBackgroundRefresh {
			m.isBackgroundRefresh = false
			m.lastRefresh = time.Now()
			// Don't reset frame or pause state during background refresh
			// Keep the animation running smoothly
		} else {
			// Normal load behavior
			m.state = StateDisplaying
			m.currentFrame = 0
			m.isPaused = false
			m.lastRefresh = time.Now()

			if !m.animationActive {
				m.animationActive = true
				cmds = append(cmds, m.AnimateFrame())
			}
		}

		if m.autoRefresh && !m.isBackgroundRefresh {
			cmds = append(cmds, m.ScheduleRefresh())
		}

	case RefreshTickMsg:
		if m.state == StateDisplaying && m.autoRefresh && m.zipCode != "" {
			// Don't show loading state during auto-refresh
			// Just load the data in the background
			cmds = append(cmds, radar.LoadData(m.zipCode))
			cmds = append(cmds, m.ScheduleRefresh())
		}

	case FrameTickMsg:
		if m.state == StateDisplaying && m.animationActive && !m.isPaused && len(m.radar.Frames) > 0 {
			m.currentFrame = (m.currentFrame + 1) % len(m.radar.Frames)
			cmds = append(cmds, m.AnimateFrame())
		} else {
			m.animationActive = false
		}

	case radar.ErrorMsg:
		m.state = StateError
		m.errorMsg = msg.Err.Error()
		m.animationActive = false

	case ErrorMsg:
		m.state = StateError
		m.errorMsg = msg.Err.Error()
		m.animationActive = false
	}

	if m.state == StateInput {
		var cmd tea.Cmd
		m.zipInput, cmd = m.zipInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	var content string

	header := config.TitleStyle.Render("🌦️  Termidar: Terminal Radar")

	switch m.state {
	case StateInput:
		inputBox := m.renderInputBox()
		help := m.renderHelp()
		content = lipgloss.JoinVertical(lipgloss.Left, header, inputBox, help)

	case StateLoading:
		loadingView := m.renderLoading()
		content = lipgloss.JoinVertical(lipgloss.Left, header, loadingView)

	case StateDisplaying:
		radarView := m.renderRadar()
		controls := m.renderControls()
		content = lipgloss.JoinVertical(lipgloss.Left, header, radarView, controls)

	case StateError:
		errorView := m.renderError()
		content = lipgloss.JoinVertical(lipgloss.Left, header, errorView)
	}

	return config.AppStyle.Render(content)
}

// Render functions
func (m Model) renderInputBox() string {
	style := config.InputContainerStyle
	if m.zipInput.Focused() {
		style = config.ActiveInputStyle
	}

	prompt := "Enter a US ZIP code to view weather radar:"
	input := m.zipInput.View()

	box := style.Render(
		lipgloss.JoinVertical(lipgloss.Left, prompt, "", input),
	)

	examples := config.SubtitleStyle.Render("Try: 10001 (NYC), 60601 (Chicago), 98101 (Seattle), 33101 (Miami)")

	return lipgloss.JoinVertical(lipgloss.Left, box, examples)
}

func (m Model) renderLoading() string {
	spinner := m.spinner.View()
	progress := config.ProgressStyle.Render(m.progress.View())

	messages := []string{
		"Locating ZIP code...",
		"Finding nearest radar station...",
		"Fetching radar data...",
		"Processing frames...",
	}

	progressPercent := m.progress.Percent()
	messageIdx := int(progressPercent * float64(len(messages)-1))
	if messageIdx >= len(messages) {
		messageIdx = len(messages) - 1
	}

	status := fmt.Sprintf("%s %s", spinner, messages[messageIdx])

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		status,
		progress,
		"",
		config.SubtitleStyle.Render("Please wait..."),
	)
}

func (m Model) renderRadar() string {
	if len(m.radar.Frames) == 0 {
		return "No radar data available"
	}

	info := m.renderInfoPanel()
	radarDisplay := m.renderRadarFrame()

	return lipgloss.JoinVertical(lipgloss.Left, info, radarDisplay)
}

func (m Model) renderInfoPanel() string {
	location := config.LocationStyle.Render(fmt.Sprintf("📍 %s", m.radar.Location))
	station := config.StationStyle.Render(fmt.Sprintf("📡 Station: %s", m.radar.Station))

	// Check for severe weather alerts
	alertDisplay := ""
	if len(m.radar.Alerts) > 0 {
		emoji, color, text := weather.GetAlertDisplay(m.radar.Alerts)
		if emoji != "" {
			alertStyle := lipgloss.NewStyle().
				Foreground(color).
				Bold(true)

			for _, alert := range m.radar.Alerts {
				if alert.Severity == "Extreme" {
					alertStyle = alertStyle.
						Background(lipgloss.Color("52")).
						Padding(0, 1)
					break
				}
			}

			alertDisplay = alertStyle.Render(fmt.Sprintf("%s %s", emoji, text))
		}
	}

	// Temperature display
	tempDisplay := ""
	if m.radar.Temperature != 0 {
		tempDisplay = fmt.Sprintf("%d°F", m.radar.Temperature)
		tempColor := lipgloss.Color("87")
		if m.radar.Temperature >= 90 {
			tempColor = lipgloss.Color("196")
		} else if m.radar.Temperature >= 70 {
			tempColor = lipgloss.Color("214")
		} else if m.radar.Temperature >= 50 {
			tempColor = lipgloss.Color("226")
		} else if m.radar.Temperature >= 32 {
			tempColor = lipgloss.Color("87")
		} else {
			tempColor = lipgloss.Color("51")
		}
		tempDisplay = lipgloss.NewStyle().Foreground(tempColor).Bold(true).Render(tempDisplay)
	}

	// Weather condition emoji
	conditionEmoji := weather.GetEmoji(m.radar.Conditions)

	// Show frame timestamp info
	var frameInfo string
	if len(m.radar.Frames) > 0 && m.currentFrame < len(m.radar.Frames) {
		frame := m.radar.Frames[m.currentFrame]
		timeAgo := time.Since(frame.Timestamp).Round(time.Minute)
		frameInfo = fmt.Sprintf("Frame %d/%d (%s ago)",
			m.currentFrame+1, len(m.radar.Frames), timeAgo)
	} else {
		frameInfo = fmt.Sprintf("Frame %d/%d", m.currentFrame+1, len(m.radar.Frames))
	}

	if m.isPaused {
		frameInfo += " (PAUSED)"
	}

	// Add last refresh time
	refreshInfo := ""
	if !m.lastRefresh.IsZero() {
		timeSinceRefresh := time.Since(m.lastRefresh).Round(time.Second)
		if timeSinceRefresh < time.Minute {
			refreshInfo = fmt.Sprintf(" • Updated %ds ago", int(timeSinceRefresh.Seconds()))
		} else {
			refreshInfo = fmt.Sprintf(" • Updated %dm ago", int(timeSinceRefresh.Minutes()))
		}
	}

	// Build the info panel
	infoItems := []string{location, station}

	if tempDisplay != "" {
		infoItems = append(infoItems, tempDisplay)
	}

	if conditionEmoji != "" {
		infoItems = append(infoItems, conditionEmoji)
	}

	topLine := strings.Join(infoItems, strings.Repeat(" ", 4))

	var lines []string
	if alertDisplay != "" {
		lines = append(lines, alertDisplay)
	}
	lines = append(lines, topLine)
	lines = append(lines, config.HelpStyle.Render(frameInfo+refreshInfo))

	return config.InfoPanelStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderRadarFrame() string {
	frame := m.radar.Frames[m.currentFrame]

	// Create the radar display grid
	display := make([][]string, config.RadarHeight)
	for i := range display {
		display[i] = make([]string, config.RadarWidth)
		for j := range display[i] {
			display[i][j] = " "
		}
	}

	// Get center coordinates from the radar station
	centerX, centerY := config.RadarWidth/2, config.RadarHeight/2

	// Draw geographic boundaries FIRST (so radar data appears on top)
	geography.DrawGeographicBoundaries(display, centerX, centerY, m.zipCode)

	// Draw simple distance markers
	geography.DrawDistanceMarkers(display, centerX, centerY)

	// Draw precipitation data
	if frame.Data != nil {
		m.DrawPrecipitation(display, frame.Data)
	}

	// Add scale indicator
	scaleInfo := "───── = 50 miles"

	// Add frame indicator dots at bottom
	var frameIndicator strings.Builder
	for i := 0; i < len(m.radar.Frames); i++ {
		if i == m.currentFrame {
			frameIndicator.WriteString("●")
		} else {
			frameIndicator.WriteString("·")
		}
		if i < len(m.radar.Frames)-1 {
			frameIndicator.WriteString(" ")
		}
	}

	// Convert to string
	var lines []string
	for _, row := range display {
		lines = append(lines, strings.Join(row, ""))
	}

	radarStr := strings.Join(lines, "\n")
	radarStr += "\n" + lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(config.RadarWidth).
		Align(lipgloss.Center).
		Render(frameIndicator.String())
	radarStr += "\n" + lipgloss.NewStyle().
		Foreground(lipgloss.Color("239")).
		Width(config.RadarWidth).
		Align(lipgloss.Center).
		Render(scaleInfo)

	return config.RadarContainerStyle.Render(radarStr)
}

func (m Model) DrawPrecipitation(display [][]string, data [][]int) {
	chars := []string{" ", "·", "∘", "○", "●", "◉", "◆", "◈", "▰", "▱", "█"}
	colors := []lipgloss.Color{
		lipgloss.Color("0"),
		lipgloss.Color("51"),
		lipgloss.Color("50"),
		lipgloss.Color("49"),
		lipgloss.Color("226"),
		lipgloss.Color("220"),
		lipgloss.Color("214"),
		lipgloss.Color("208"),
		lipgloss.Color("202"),
		lipgloss.Color("196"),
		lipgloss.Color("160"),
	}

	for y := 0; y < len(data) && y < config.RadarHeight; y++ {
		for x := 0; x < len(data[y]) && x < config.RadarWidth; x++ {
			intensity := data[y][x]
			if intensity > 0 && intensity < len(chars) {
				char := chars[intensity]
				color := colors[intensity]
				display[y][x] = lipgloss.NewStyle().Foreground(color).Render(char)
			}
		}
	}
}

func (m Model) renderControls() string {
	controls := []string{
		"[Space] Play/Pause",
		"[←/→] Previous/Next",
		"[R] Refresh",
		"[+/-] Speed",
		"[ESC] New location",
		"[Q] Quit",
	}

	if m.showHelp {
		controls = append(controls, "",
			fmt.Sprintf("Frame rate: %s", m.frameRate),
			fmt.Sprintf("Auto-refresh: Every 5 minutes"),
		)
	}

	controlStr := config.HelpStyle.Render(strings.Join(controls, " • "))
	return controlStr
}

func (m Model) renderError() string {
	errorMsg := config.ErrorStyle.Render("❌ " + m.errorMsg)
	help := config.HelpStyle.Render("Press ESC to try again or Q to quit")

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		errorMsg,
		"",
		help,
	)
}

func (m Model) renderHelp() string {
	help := []string{
		"🎮 Controls:",
		"  Enter - Submit ZIP code",
		"  ESC   - Cancel/Back",
		"  Q     - Quit",
		"",
		"📡 During radar display:",
		"  Space - Play/Pause animation",
		"  ←/→   - Navigate frames",
		"  +/-   - Adjust speed",
	}

	if m.showHelp {
		return config.HelpStyle.Render(strings.Join(help, "\n"))
	}

	return config.HelpStyle.Render("Press ? for help")
}

// Helper methods
func (m Model) ResetToInput() Model {
	m.state = StateInput
	m.radar = radar.Data{}
	m.currentFrame = 0
	m.errorMsg = ""
	m.zipInput.SetValue("")
	m.zipInput.Focus()
	m.animationActive = false
	return m
}

// Animation commands
func (m Model) AnimateFrame() tea.Cmd {
	return tea.Tick(m.frameRate, func(t time.Time) tea.Msg {
		return FrameTickMsg(t)
	})
}

func (m Model) ScheduleRefresh() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return RefreshTickMsg(t)
	})
}

func (m Model) TrackProgress() tea.Cmd {
	return func() tea.Msg {
		for i := 0; i <= 100; i += 10 {
			time.Sleep(200 * time.Millisecond)
		}
		return nil
	}
}