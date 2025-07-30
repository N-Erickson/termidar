package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Constants
const (
	radarWidth  = 60
	radarHeight = 30
	maxFrames   = 20  // Increased from 12 to get more animation frames
)

// Styles
var (
	// Color palette
	primaryColor   = lipgloss.Color("86")
	secondaryColor = lipgloss.Color("205")
	accentColor    = lipgloss.Color("213")
	errorColor     = lipgloss.Color("196")
	successColor   = lipgloss.Color("46")
	radarGreen     = lipgloss.Color("40")
	radarYellow    = lipgloss.Color("226")
	radarOrange    = lipgloss.Color("208")
	radarRed       = lipgloss.Color("196")

	// Layout styles
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	// Input styles
	inputContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(secondaryColor).
				Padding(1, 2)

	activeInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Padding(1, 2)

	// Info panel styles
	infoPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("239")).
			Padding(0, 1).
			MarginTop(1)

	locationStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(successColor)

	stationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Radar styles
	radarContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(radarGreen).
				Padding(1).
				MarginTop(1)

	radarFrameStyle = lipgloss.NewStyle().
			Width(radarWidth).
			Height(radarHeight)

	// Status styles
	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// Progress bar style
	progressStyle = lipgloss.NewStyle().
			MarginTop(1)
)

// Model states
type state int

const (
	stateInput state = iota
	stateLoading
	stateDisplaying
	stateError
)

// Radar data types
type radarData struct {
	frames      []radarFrame
	location    string
	station     string
	lastUpdated time.Time
	isRealData  bool // Track if this is real or simulated data
}

type radarFrame struct {
	data      [][]int // Intensity values 0-10
	timestamp time.Time
	product   string // Radar product type (N0R, N0S, etc.)
}

// Model represents the application state
type model struct {
	state        state
	zipInput     textinput.Model
	spinner      spinner.Model
	progress     progress.Model
	radar        radarData
	currentFrame int
	frameTimer   *time.Timer
	width        int
	height       int
	errorMsg     string
	
	// UI state
	showHelp     bool
	isPaused     bool
	frameRate    time.Duration
	
	// Auto-refresh
	lastRefresh  time.Time
	refreshTimer *time.Timer
	autoRefresh  bool
	zipCode      string // Store for refresh
}

// Particle effect for radar
type particle struct {
	x, y   float64
	vx, vy float64
	life   int
	color  lipgloss.Color
}

// Messages
type tickMsg time.Time
type frameTickMsg time.Time
type refreshTickMsg time.Time
type radarLoadedMsg struct {
	radar radarData
}
type errorMsg struct {
	err error
}
type progressMsg float64

// Initialize the model
func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter ZIP code"
	ti.Focus()
	ti.CharLimit = 5
	ti.Width = 20
	ti.Prompt = "📍 "

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(secondaryColor)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return model{
		state:       stateInput,
		zipInput:    ti,
		spinner:     s,
		progress:    p,
		width:       80,
		height:      40,
		frameRate:   300 * time.Millisecond, // Faster for smoother animation
		autoRefresh: true,
	}
}

// Init initializes the model
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state == stateDisplaying || m.state == stateError {
				m = m.resetToInput()
				return m, textinput.Blink
			}
		case "enter":
			if m.state == stateInput && len(m.zipInput.Value()) == 5 {
				m.state = stateLoading
				m.zipCode = m.zipInput.Value()
				cmds = append(cmds, 
					m.spinner.Tick,
					loadRadarData(m.zipCode),
					m.trackProgress(),
				)
			}
		case "?", "h":
			m.showHelp = !m.showHelp
		case " ":
			if m.state == stateDisplaying {
				m.isPaused = !m.isPaused
			}
		case "r":
			if m.state == stateDisplaying && m.zipCode != "" {
				// Manual refresh
				m.state = stateLoading
				cmds = append(cmds,
					m.spinner.Tick,
					loadRadarData(m.zipCode),
					m.trackProgress(),
				)
			}
		case "left", "a":
			if m.state == stateDisplaying && len(m.radar.frames) > 0 {
				m.currentFrame = (m.currentFrame - 1 + len(m.radar.frames)) % len(m.radar.frames)
			}
		case "right", "d":
			if m.state == stateDisplaying && len(m.radar.frames) > 0 {
				m.currentFrame = (m.currentFrame + 1) % len(m.radar.frames)
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
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case progressMsg:
		if m.state == stateLoading {
			cmd := m.progress.SetPercent(float64(msg))
			cmds = append(cmds, cmd)
		}

	case radarLoadedMsg:
		m.radar = msg.radar
		m.state = stateDisplaying
		m.currentFrame = 0
		m.isPaused = false
		m.lastRefresh = time.Now()
		cmds = append(cmds, m.animateFrame())
		
		// Start auto-refresh timer (refresh every 5 minutes)
		if m.autoRefresh {
			cmds = append(cmds, m.scheduleRefresh())
		}

	case refreshTickMsg:
		if m.state == stateDisplaying && m.autoRefresh && m.zipCode != "" {
			// Auto-refresh the radar data
			cmds = append(cmds, loadRadarData(m.zipCode))
			// Schedule next refresh
			cmds = append(cmds, m.scheduleRefresh())
		}

	case frameTickMsg:
		if m.state == stateDisplaying && !m.isPaused && len(m.radar.frames) > 0 {
			m.currentFrame = (m.currentFrame + 1) % len(m.radar.frames)
			cmds = append(cmds, m.animateFrame())
		}

	case errorMsg:
		m.state = stateError
		m.errorMsg = msg.err.Error()

	}

	// Handle text input updates
	if m.state == stateInput {
		var cmd tea.Cmd
		m.zipInput, cmd = m.zipInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m model) View() string {
	var content string

	// Header
	header := titleStyle.Render("🌦️  Weather Radar Terminal v2.0")
	
	switch m.state {
	case stateInput:
		inputBox := m.renderInputBox()
		help := m.renderHelp()
		content = lipgloss.JoinVertical(lipgloss.Left, header, inputBox, help)

	case stateLoading:
		loadingView := m.renderLoading()
		content = lipgloss.JoinVertical(lipgloss.Left, header, loadingView)

	case stateDisplaying:
		radarView := m.renderRadar()
		controls := m.renderControls()
		content = lipgloss.JoinVertical(lipgloss.Left, header, radarView, controls)

	case stateError:
		errorView := m.renderError()
		content = lipgloss.JoinVertical(lipgloss.Left, header, errorView)
	}

	return appStyle.Render(content)
}

// Render functions
func (m model) renderInputBox() string {
	style := inputContainerStyle
	if m.zipInput.Focused() {
		style = activeInputStyle
	}

	prompt := "Enter a US ZIP code to view weather radar:"
	input := m.zipInput.View()
	
	box := style.Render(
		lipgloss.JoinVertical(lipgloss.Left, prompt, "", input),
	)

	examples := subtitleStyle.Render("Try: 10001 (NYC), 60601 (Chicago), 98101 (Seattle), 33101 (Miami)")
	
	return lipgloss.JoinVertical(lipgloss.Left, box, examples)
}

func (m model) renderLoading() string {
	spinner := m.spinner.View()
	progress := progressStyle.Render(m.progress.View())
	
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
		subtitleStyle.Render("Please wait..."),
	)
}

func (m model) renderRadar() string {
	if len(m.radar.frames) == 0 {
		return "No radar data available"
	}

	// Info panel
	info := m.renderInfoPanel()
	
	// Radar display
	radarDisplay := m.renderRadarFrame()
	
	return lipgloss.JoinVertical(lipgloss.Left, info, radarDisplay)
}

func (m model) renderInfoPanel() string {
	location := locationStyle.Render(fmt.Sprintf("📍 %s", m.radar.location))
	station := stationStyle.Render(fmt.Sprintf("📡 Station: %s", m.radar.station))
	
	// Show frame timestamp to see movement
	var frameInfo string
	if len(m.radar.frames) > 0 && m.currentFrame < len(m.radar.frames) {
		frame := m.radar.frames[m.currentFrame]
		timeAgo := time.Since(frame.timestamp).Round(time.Minute)
		frameInfo = fmt.Sprintf("Frame %d/%d (%s ago)", 
			m.currentFrame+1, len(m.radar.frames), timeAgo)
		
		// Add product type if available
		if frame.product != "" {
			frameInfo += fmt.Sprintf(" [%s]", frame.product)
		}
	} else {
		frameInfo = fmt.Sprintf("Frame %d/%d", m.currentFrame+1, len(m.radar.frames))
	}
	
	if m.isPaused {
		frameInfo += " (PAUSED)"
	}
	
	// Add data source indicator with more detail
	var dataSource string
	if m.radar.isRealData {
		dataSource = lipgloss.NewStyle().
			Foreground(successColor).
			Render(" ✓ Live Radar")
		// Add source info
		if len(m.radar.frames) > 0 && m.radar.frames[0].product != "" {
			dataSource += lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render(fmt.Sprintf(" (NEXRAD %s)", m.radar.frames[0].product))
		}
	} else {
		dataSource = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Render(" ⚠️  Simulated Data")
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
	
	info := lipgloss.JoinHorizontal(lipgloss.Top,
		location,
		strings.Repeat(" ", 4),
		station,
		strings.Repeat(" ", 4),
		frameInfo,
		dataSource,
		refreshInfo,
	)
	
	return infoPanelStyle.Render(info)
}

func (m model) renderRadarFrame() string {
	frame := m.radar.frames[m.currentFrame]
	
	// Create the radar display grid
	display := make([][]string, radarHeight)
	for i := range display {
		display[i] = make([]string, radarWidth)
		for j := range display[i] {
			display[i][j] = " "
		}
	}
	
	// Get center coordinates from the radar station
	centerX, centerY := radarWidth/2, radarHeight/2
	
	// Draw geographic boundaries FIRST (so radar data appears on top)
	m.drawGeographicBoundaries(display, centerX, centerY)
	
	// Draw simple distance markers instead of animated radar circles
	m.drawDistanceMarkers(display, centerX, centerY)
	
	// Draw precipitation data - this is the actual radar data
	if frame.data != nil {
		m.drawPrecipitation(display, frame.data)
	}
	
	// Add scale indicator
	scaleInfo := "───── = 50 miles"
	
	// Add frame indicator dots at bottom
	var frameIndicator strings.Builder
	for i := 0; i < len(m.radar.frames); i++ {
		if i == m.currentFrame {
			frameIndicator.WriteString("●")
		} else {
			frameIndicator.WriteString("·")
		}
		if i < len(m.radar.frames)-1 {
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
		Width(radarWidth).
		Align(lipgloss.Center).
		Render(frameIndicator.String())
	radarStr += "\n" + lipgloss.NewStyle().
		Foreground(lipgloss.Color("239")).
		Width(radarWidth).
		Align(lipgloss.Center).
		Render(scaleInfo)
	
	return radarContainerStyle.Render(radarStr)
}

func (m model) drawDistanceMarkers(display [][]string, centerX, centerY int) {
	// Draw simple distance circles (not animated)
	markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	
	// 50 mile ring
	radius := 12
	for angle := 0.0; angle < 360.0; angle += 10 {
		x := int(float64(centerX) + float64(radius)*math.Cos(angle*math.Pi/180))
		y := int(float64(centerY) + float64(radius)*math.Sin(angle*math.Pi/180))
		
		if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
			if display[y][x] == " " {
				display[y][x] = markerStyle.Render("·")
			}
		}
	}
	
	// 100 mile ring
	radius = 22
	for angle := 0.0; angle < 360.0; angle += 15 {
		x := int(float64(centerX) + float64(radius)*math.Cos(angle*math.Pi/180))
		y := int(float64(centerY) + float64(radius)*math.Sin(angle*math.Pi/180)*0.5) // Adjust for terminal aspect ratio
		
		if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
			if display[y][x] == " " {
				display[y][x] = markerStyle.Render("·")
			}
		}
	}
}

func (m model) drawPrecipitation(display [][]string, data [][]int) {
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
	
	for y := 0; y < len(data) && y < radarHeight; y++ {
		for x := 0; x < len(data[y]) && x < radarWidth; x++ {
			intensity := data[y][x]
			if intensity > 0 && intensity < len(chars) {
				char := chars[intensity]
				color := colors[intensity]
				display[y][x] = lipgloss.NewStyle().Foreground(color).Render(char)
			}
		}
	}
}

func (m model) drawGeographicBoundaries(display [][]string, centerX, centerY int) {
	// This draws approximate state/geographic boundaries based on the radar station
	// The boundaries are simplified for terminal display
	
	boundaryChar := "·"
	boundaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	
	switch m.radar.station {
	case "KOKX": // New York
		// Draw Long Island coastline
		for x := 10; x < 40; x++ {
			y := centerY + 5 + int(math.Sin(float64(x)/5)*2)
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				display[y][x] = boundaryStyle.Render("═")
			}
		}
		// Connecticut border
		for y := 0; y < centerY; y++ {
			x := centerX - 10
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				display[y][x] = boundaryStyle.Render("│")
			}
		}
		
	case "KLOT": // Chicago
		// Lake Michigan shoreline
		for y := 0; y < radarHeight; y++ {
			x := centerX + 15 - int(math.Sin(float64(y)/3)*3)
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				display[y][x] = boundaryStyle.Render("│")
			}
		}
		// State lines
		for x := 0; x < centerX-5; x++ {
			y := centerY + 8
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				display[y][x] = boundaryStyle.Render("─")
			}
		}
		
	case "KAMX": // Miami
		// Florida coastline
		for y := 0; y < radarHeight-5; y++ {
			x := centerX + 12 + int(math.Sin(float64(y)/4)*2)
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				display[y][x] = boundaryStyle.Render(")")
			}
		}
		// Draw the Keys
		for x := centerX-10; x < centerX+5; x++ {
			y := radarHeight - 3
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				if x%3 == 0 {
					display[y][x] = boundaryStyle.Render("·")
				}
			}
		}
		
	case "KATX": // Seattle
		// Puget Sound
		for y := 5; y < radarHeight-5; y++ {
			x := centerX - int(math.Sin(float64(y)/5)*3)
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				display[y][x] = boundaryStyle.Render("≈")
			}
		}
		// Cascade Mountains (to the east)
		for y := 0; y < radarHeight; y++ {
			x := centerX + 18
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				display[y][x] = boundaryStyle.Render("^")
			}
		}
		
	case "KFWS": // Dallas
		// Draw rough Texas borders
		for x := 0; x < 10; x++ {
			y := centerY - 10
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
				display[y][x] = boundaryStyle.Render("─")
			}
		}
		// Red River
		for x := centerX-15; x < centerX+15; x++ {
			y := 3
			if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) && x%2 == 0 {
				display[y][x] = boundaryStyle.Render("~")
			}
		}
		
	default:
		// Generic state boundary indicators
		for i := 0; i < 4; i++ {
			angle := float64(i) * math.Pi / 2
			for r := 15; r < 20; r++ {
				x := centerX + int(float64(r)*math.Cos(angle))
				y := centerY + int(float64(r)*math.Sin(angle))
				if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
					display[y][x] = boundaryStyle.Render(boundaryChar)
				}
			}
		}
	}
	
	// Add city marker for the center
	if centerY >= 0 && centerY < len(display) && centerX >= 0 && centerX < len(display[0]) {
		display[centerY][centerX] = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true).
			Render("★")
	}
}


func (m model) renderControls() string {
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
	
	controlStr := helpStyle.Render(strings.Join(controls, " • "))
	return controlStr
}

func (m model) renderError() string {
	errorMsg := errorStyle.Render("❌ " + m.errorMsg)
	help := helpStyle.Render("Press ESC to try again or Q to quit")
	
	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		errorMsg,
		"",
		help,
	)
}

func (m model) renderHelp() string {
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
		return helpStyle.Render(strings.Join(help, "\n"))
	}
	
	return helpStyle.Render("Press ? for help")
}

// Helper methods
func (m model) resetToInput() model {
	m.state = stateInput
	m.radar = radarData{}
	m.currentFrame = 0
	m.errorMsg = ""
	m.zipInput.SetValue("")
	m.zipInput.Focus()
	return m
}

// Animation commands
func (m model) animateFrame() tea.Cmd {
	return tea.Tick(m.frameRate, func(t time.Time) tea.Msg {
		return frameTickMsg(t)
	})
}

func (m model) scheduleRefresh() tea.Cmd {
	// Refresh every 5 minutes
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return refreshTickMsg(t)
	})
}

func (m model) trackProgress() tea.Cmd {
	return func() tea.Msg {
		// Simulate progress updates
		for i := 0; i <= 100; i += 10 {
			time.Sleep(200 * time.Millisecond)
			// Send progress update
		}
		return nil
	}
}

// Data loading
func loadRadarData(zipCode string) tea.Cmd {
	return func() tea.Msg {
		// Get location from ZIP
		lat, lon, city, state, err := geocodeZip(zipCode)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to geocode ZIP: %w", err)}
		}

		// Get radar station
		station, err := getNWSRadarStation(lat, lon)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to get radar station: %w", err)}
		}

		// Fetch real radar data
		frames, isRealData, err := fetchRealRadarData(station, lat, lon)
		if err != nil {
			// Fall back to simulated data if real data fails
			frames = generateRadarFrames(station, maxFrames)
			isRealData = false
		}
		
		location := fmt.Sprintf("%s, %s", city, state)
		
		return radarLoadedMsg{
			radar: radarData{
				frames:      frames,
				location:    location,
				station:     station,
				lastUpdated: time.Now(),
				isRealData:  isRealData,
			},
		}
	}
}

func fetchRealRadarData(station string, lat, lon float64) ([]radarFrame, bool, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	frames := []radarFrame{}
	
	// First try RainViewer - it provides more historical frames
	frames, err := fetchFromRainViewer(lat, lon)
	if err == nil && len(frames) > 0 {
		log.Printf("Successfully fetched %d frames from RainViewer", len(frames))
		return frames, true, nil
	}
	
	// Fallback to Iowa State University
	baseTime := time.Now().UTC()
	
	// Try to get the last 2 hours of data (every 5 minutes)
	for i := 0; i < 24; i++ { // 24 frames = 2 hours
		frameTime := baseTime.Add(time.Duration(-i*5) * time.Minute)
		
		// Round to nearest 5 minutes
		minutes := frameTime.Minute()
		minutes = (minutes / 5) * 5
		frameTime = time.Date(frameTime.Year(), frameTime.Month(), frameTime.Day(),
			frameTime.Hour(), minutes, 0, 0, time.UTC)
		
		// Build URL for Iowa State's Mesonet API
		timeStr := frameTime.Format("200601021504")
		radarURL := fmt.Sprintf("https://mesonet.agron.iastate.edu/cgi-bin/wms/nexrad/n0r.cgi?SERVICE=WMS&VERSION=1.1.1&REQUEST=GetMap&FORMAT=image/png&TRANSPARENT=true&LAYERS=nexrad-n0r&WIDTH=%d&HEIGHT=%d&SRS=EPSG:4326&BBOX=%f,%f,%f,%f&TIME=%s",
			radarWidth*4, radarHeight*4, // Higher resolution
			lon-2.5, lat-2.0, lon+2.5, lat+2.0, // Wider area
			timeStr,
		)
		
		resp, err := client.Get(radarURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			continue
		}
		
		// Decode the PNG image
		img, err := png.Decode(resp.Body)
		if err != nil {
			continue
		}
		
		// Convert image to radar data
		data := imageToRadarData(img)
		if data != nil {
			frame := radarFrame{
				data:      data,
				timestamp: frameTime,
				product:   "N0R",
			}
			frames = append(frames, frame)
		}
		
		// Stop if we have enough frames
		if len(frames) >= maxFrames {
			break
		}
	}
	
	if len(frames) == 0 {
		return nil, false, fmt.Errorf("no radar data available")
	}
	
	// Reverse frames so oldest is first
	for i := len(frames)/2 - 1; i >= 0; i-- {
		opp := len(frames) - 1 - i
		frames[i], frames[opp] = frames[opp], frames[i]
	}
	
	log.Printf("Successfully fetched %d frames from Iowa State", len(frames))
	return frames, true, nil
}

func fetchFromRainViewer(lat, lon float64) ([]radarFrame, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Get available timestamps
	resp, err := client.Get("https://api.rainviewer.com/public/weather-maps.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var apiData struct {
		Radar struct {
			Past []struct {
				Time int64  `json:"time"`
				Path string `json:"path"`
			} `json:"past"`
			Nowcast []struct {
				Time int64  `json:"time"`
				Path string `json:"path"`
			} `json:"nowcast"`
		} `json:"radar"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		return nil, err
	}
	
	frames := []radarFrame{}
	
	// Get all available past radar frames
	for _, past := range apiData.Radar.Past {
		// Calculate tile coordinates for the location
		zoom := 7 // Higher zoom for more detail
		tileX, tileY := latLonToTile(lat, lon, zoom)
		
		// Build tile URL
		tileURL := fmt.Sprintf("https://tilecache.rainviewer.com%s/512/%d/%d/%d/6/1_1.png",
			past.Path, zoom, tileX, tileY)
		
		resp, err := client.Get(tileURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		img, err := png.Decode(resp.Body)
		if err != nil {
			continue
		}
		
		// Convert to radar data
		data := imageToRadarData(img)
		if data != nil {
			frame := radarFrame{
				data:      data,
				timestamp: time.Unix(past.Time, 0),
				product:   "Composite",
			}
			frames = append(frames, frame)
		}
		
		if len(frames) >= maxFrames {
			break
		}
	}
	
	return frames, nil
}

func latLonToTile(lat, lon float64, zoom int) (int, int) {
	n := math.Pow(2, float64(zoom))
	x := int((lon + 180.0) / 360.0 * n)
	latRad := lat * math.Pi / 180.0
	y := int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n)
	return x, y
}

func imageToRadarData(img image.Image) [][]int {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	// Create radar data array
	data := make([][]int, radarHeight)
	for i := range data {
		data[i] = make([]int, radarWidth)
	}
	
	// Track if we found any precipitation
	foundPrecipitation := false
	
	// Sample the image and convert to intensity values
	for y := 0; y < radarHeight; y++ {
		for x := 0; x < radarWidth; x++ {
			// Map terminal coordinates to image coordinates
			imgX := x * width / radarWidth
			imgY := y * height / radarHeight
			
			// Get pixel color
			c := img.At(imgX, imgY)
			r, g, b, a := c.RGBA()
			
			// Skip transparent pixels
			if a < 128 {
				continue
			}
			
			// Convert color to radar intensity (0-10)
			// Radar color scale typically goes from green -> yellow -> red
			intensity := 0
			
			// Normalize to 0-255 range
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			
			// Detect precipitation colors (based on standard NEXRAD color scales)
			if r8 > 200 && g8 < 100 && b8 < 100 {
				// Red - heavy precipitation
				intensity = 8 + int((r8-200)/28)
				foundPrecipitation = true
			} else if r8 > 200 && g8 > 150 && b8 < 100 {
				// Orange/Yellow - moderate precipitation
				intensity = 5 + int((r8-200)/50)
				foundPrecipitation = true
			} else if g8 > 150 && r8 < 150 && b8 < 100 {
				// Green - light precipitation
				intensity = 1 + int((g8-150)/50)
				foundPrecipitation = true
			} else if b8 > 150 && r8 < 100 && g8 < 150 {
				// Blue - very light precipitation
				intensity = 1
				foundPrecipitation = true
			} else if r8 > 100 && g8 > 100 && b8 < 50 {
				// Yellowish - moderate
				intensity = 4
				foundPrecipitation = true
			}
			
			// Clamp to valid range
			if intensity > 10 {
				intensity = 10
			}
			if intensity < 0 {
				intensity = 0
			}
			
			data[y][x] = intensity
		}
	}
	
	// Log if we found precipitation (helps verify real data)
	if foundPrecipitation {
		log.Printf("Found precipitation in radar image")
	} else {
		log.Printf("No precipitation detected in radar image")
	}
	
	return data
}

func generateRealisticRadarData(station string, frameOffset int) [][]int {
	// This generates more realistic-looking radar data
	// In a real app, this would be replaced with actual radar data parsing
	data := make([][]int, radarHeight)
	for y := range data {
		data[y] = make([]int, radarWidth)
	}
	
	// Create realistic weather patterns based on station location
	stationPatterns := map[string]struct {
		stormX, stormY int
		size           int
		intensity      int
		direction      float64
	}{
		"KOKX": {25, 12, 8, 7, 45},   // NYC - northeast movement
		"KLOT": {20, 15, 10, 8, 90},  // Chicago - eastward movement
		"KAMX": {30, 20, 12, 9, 315}, // Miami - northwest movement
		"KATX": {15, 10, 9, 6, 135},  // Seattle - southeast movement
		"KFWS": {22, 18, 11, 8, 60},  // Dallas - northeast movement
	}
	
	pattern, exists := stationPatterns[station]
	if !exists {
		pattern = stationPatterns["KOKX"] // Default pattern
	}
	
	// Move storm based on frame
	stormX := pattern.stormX + frameOffset*int(math.Cos(pattern.direction*math.Pi/180)*3)
	stormY := pattern.stormY + frameOffset*int(math.Sin(pattern.direction*math.Pi/180)*2)
	
	// Create main storm cell
	for y := 0; y < radarHeight; y++ {
		for x := 0; x < radarWidth; x++ {
			dx := float64(x - stormX)
			dy := float64(y - stormY)
			dist := math.Sqrt(dx*dx + dy*dy)
			
			if dist < float64(pattern.size) {
				// Realistic radar reflectivity pattern
				intensity := pattern.intensity - int(dist*float64(pattern.intensity)/float64(pattern.size))
				
				// Add some noise for realism
				noise := int(math.Sin(float64(x)*0.3)*2 + math.Cos(float64(y)*0.4)*2)
				intensity += noise
				
				if intensity > 0 && intensity <= 10 {
					data[y][x] = intensity
				}
			}
			
			// Add scattered precipitation
			if math.Sin(float64(x+y+frameOffset))*math.Cos(float64(x-y)) > 0.7 {
				if data[y][x] == 0 {
					data[y][x] = 1 + int(math.Abs(math.Sin(float64(x*y))*3))
				}
			}
		}
	}
	
	// Add secondary cells
	for i := 0; i < 2; i++ {
		cellX := (stormX + 10 + i*15) % radarWidth
		cellY := (stormY + 5 + i*8) % radarHeight
		cellSize := 4 + i*2
		
		for y := -cellSize; y <= cellSize; y++ {
			for x := -cellSize; x <= cellSize; x++ {
				px, py := cellX+x, cellY+y
				if px >= 0 && px < radarWidth && py >= 0 && py < radarHeight {
					dist := math.Sqrt(float64(x*x + y*y))
					if dist < float64(cellSize) {
						intensity := 5 - int(dist)
						if intensity > data[py][px] {
							data[py][px] = intensity
						}
					}
				}
			}
		}
	}
	
	return data
}

func generateRadarFrames(station string, count int) []radarFrame {
	frames := make([]radarFrame, count)
	
	for i := 0; i < count; i++ {
		// Generate simulated precipitation data
		data := make([][]int, radarHeight)
		for y := range data {
			data[y] = make([]int, radarWidth)
		}
		
		// Create some storm cells
		numCells := 2 + i%3
		for c := 0; c < numCells; c++ {
			// Random storm position that moves
			centerX := 10 + (i*3+c*15)%radarWidth
			centerY := 5 + (i*2+c*10)%radarHeight
			intensity := 5 + c*2
			
			// Draw storm cell
			for dy := -5; dy <= 5; dy++ {
				for dx := -5; dx <= 5; dx++ {
					x, y := centerX+dx, centerY+dy
					if x >= 0 && x < radarWidth && y >= 0 && y < radarHeight {
						dist := math.Sqrt(float64(dx*dx + dy*dy))
						if dist < 5 {
							data[y][x] = intensity - int(dist)
							if data[y][x] < 0 {
								data[y][x] = 0
							}
							if data[y][x] > 10 {
								data[y][x] = 10
							}
						}
					}
				}
			}
		}
		
		frames[i] = radarFrame{
			data:      data,
			timestamp: time.Now().Add(time.Duration(i*10) * time.Minute),
		}
	}
	
	return frames
}

// API functions (keeping the existing ones)
func geocodeZip(zipCode string) (float64, float64, string, string, error) {
	// First try using a free ZIP code API
	url := fmt.Sprintf("https://api.zippopotam.us/us/%s", zipCode)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return getHardcodedLocation(zipCode)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return getHardcodedLocation(zipCode)
	}

	var result struct {
		PostCode    string `json:"post code"`
		Country     string `json:"country"`
		CountryCode string `json:"country abbreviation"`
		Places      []struct {
			PlaceName  string  `json:"place name"`
			State      string  `json:"state"`
			StateCode  string  `json:"state abbreviation"`
			Latitude   string  `json:"latitude"`
			Longitude  string  `json:"longitude"`
		} `json:"places"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return getHardcodedLocation(zipCode)
	}

	if len(result.Places) == 0 {
		return getHardcodedLocation(zipCode)
	}

	place := result.Places[0]
	
	lat, err := strconv.ParseFloat(place.Latitude, 64)
	if err != nil {
		return getHardcodedLocation(zipCode)
	}
	
	lon, err := strconv.ParseFloat(place.Longitude, 64)
	if err != nil {
		return getHardcodedLocation(zipCode)
	}

	return lat, lon, place.PlaceName, place.StateCode, nil
}

func getHardcodedLocation(zipCode string) (float64, float64, string, string, error) {
	locations := map[string]struct {
		lat   float64
		lon   float64
		city  string
		state string
	}{
		// Major cities
		"10001": {40.7505, -73.9965, "New York", "NY"},
		"10002": {40.7157, -73.9859, "New York", "NY"},
		"10003": {40.7317, -73.9885, "New York", "NY"},
		"90210": {34.0901, -118.4065, "Beverly Hills", "CA"},
		"60601": {41.8856, -87.6228, "Chicago", "IL"},
		"60602": {41.8826, -87.6290, "Chicago", "IL"},
		"33101": {25.7751, -80.1947, "Miami", "FL"},
		"98101": {47.6080, -122.3351, "Seattle", "WA"},
		"75201": {32.7831, -96.8067, "Dallas", "TX"},
		"85001": {33.4484, -112.0740, "Phoenix", "AZ"},
		"80202": {39.7392, -104.9903, "Denver", "CO"},
		"02108": {42.3601, -71.0589, "Boston", "MA"},
		"94102": {37.7749, -122.4194, "San Francisco", "CA"},
		"30301": {33.7490, -84.3880, "Atlanta", "GA"},
		"77001": {29.7604, -95.3698, "Houston", "TX"},
		"19019": {39.9526, -75.1652, "Philadelphia", "PA"},
		"48201": {42.3314, -83.0458, "Detroit", "MI"},
		"55401": {44.9778, -93.2650, "Minneapolis", "MN"},
		"97201": {45.5152, -122.6784, "Portland", "OR"},
		"89101": {36.1699, -115.1398, "Las Vegas", "NV"},
		"70112": {29.9511, -90.0715, "New Orleans", "LA"},
	}

	if loc, ok := locations[zipCode]; ok {
		return loc.lat, loc.lon, loc.city, loc.state, nil
	}

	return 0, 0, "", "", fmt.Errorf("unknown ZIP code: %s", zipCode)
}

func getNWSRadarStation(lat, lon float64) (string, error) {
	// Simplified - return nearest major radar station
	stations := []struct {
		id   string
		lat  float64
		lon  float64
	}{
		{"KOKX", 40.8653, -72.8639}, // New York
		{"KLOT", 41.6045, -88.0847}, // Chicago
		{"KAMX", 25.6111, -80.4128}, // Miami
		{"KATX", 48.1945, -122.4958}, // Seattle
		{"KFWS", 32.5731, -97.3031}, // Dallas
	}

	minDist := 999999.0
	nearest := "KOKX"
	
	for _, s := range stations {
		dist := math.Sqrt(math.Pow(lat-s.lat, 2) + math.Pow(lon-s.lon, 2))
		if dist < minDist {
			minDist = dist
			nearest = s.id
		}
	}

	return nearest, nil
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}