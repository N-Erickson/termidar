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
	maxFrames   = 20
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

// Weather alert types
type weatherAlert struct {
	event       string
	severity    string
	urgency     string
	headline    string
	description string
	expires     time.Time
}

// Radar data types
type radarData struct {
	frames      []radarFrame
	location    string
	station     string
	lastUpdated time.Time
	isRealData  bool
	temperature int
	conditions  string
	alerts      []weatherAlert
}

type radarFrame struct {
	data      [][]int
	timestamp time.Time
	product   string
}

// Model represents the application state
type model struct {
	state           state
	zipInput        textinput.Model
	spinner         spinner.Model
	progress        progress.Model
	radar           radarData
	currentFrame    int
	width           int
	height          int
	errorMsg        string
	showHelp        bool
	isPaused        bool
	frameRate       time.Duration
	lastRefresh     time.Time
	autoRefresh     bool
	zipCode         string
	animationActive bool
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
		state:           stateInput,
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
				m.animationActive = false
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
				if !m.isPaused && !m.animationActive {
					m.animationActive = true
					cmds = append(cmds, m.animateFrame())
				}
			}
		case "r":
			if m.state == stateDisplaying && m.zipCode != "" {
				m.animationActive = false
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

		if !m.animationActive {
			m.animationActive = true
			cmds = append(cmds, m.animateFrame())
		}

		if m.autoRefresh {
			cmds = append(cmds, m.scheduleRefresh())
		}

	case refreshTickMsg:
		if m.state == stateDisplaying && m.autoRefresh && m.zipCode != "" {
			cmds = append(cmds, loadRadarData(m.zipCode))
			cmds = append(cmds, m.scheduleRefresh())
		}

	case frameTickMsg:
		if m.state == stateDisplaying && m.animationActive && !m.isPaused && len(m.radar.frames) > 0 {
			m.currentFrame = (m.currentFrame + 1) % len(m.radar.frames)
			cmds = append(cmds, m.animateFrame())
		} else {
			m.animationActive = false
		}

	case errorMsg:
		m.state = stateError
		m.errorMsg = msg.err.Error()
		m.animationActive = false
	}

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

	header := titleStyle.Render("🌦️  Termidar: Terminal Radar")

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

	info := m.renderInfoPanel()
	radarDisplay := m.renderRadarFrame()

	return lipgloss.JoinVertical(lipgloss.Left, info, radarDisplay)
}

func (m model) renderInfoPanel() string {
	location := locationStyle.Render(fmt.Sprintf("📍 %s", m.radar.location))
	station := stationStyle.Render(fmt.Sprintf("📡 Station: %s", m.radar.station))

	// Check for severe weather alerts
	alertDisplay := ""
	if len(m.radar.alerts) > 0 {
		emoji, color, text := getAlertDisplay(m.radar.alerts)
		if emoji != "" {
			alertStyle := lipgloss.NewStyle().
				Foreground(color).
				Bold(true)

			for _, alert := range m.radar.alerts {
				if alert.severity == "Extreme" {
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
	if m.radar.temperature != 0 {
		tempDisplay = fmt.Sprintf("%d°F", m.radar.temperature)
		tempColor := lipgloss.Color("87")
		if m.radar.temperature >= 90 {
			tempColor = lipgloss.Color("196")
		} else if m.radar.temperature >= 70 {
			tempColor = lipgloss.Color("214")
		} else if m.radar.temperature >= 50 {
			tempColor = lipgloss.Color("226")
		} else if m.radar.temperature >= 32 {
			tempColor = lipgloss.Color("87")
		} else {
			tempColor = lipgloss.Color("51")
		}
		tempDisplay = lipgloss.NewStyle().Foreground(tempColor).Bold(true).Render(tempDisplay)
	}

	// Weather condition emoji
	conditionEmoji := getWeatherEmoji(m.radar.conditions)

	// Show frame timestamp info
	var frameInfo string
	if len(m.radar.frames) > 0 && m.currentFrame < len(m.radar.frames) {
		frame := m.radar.frames[m.currentFrame]
		timeAgo := time.Since(frame.timestamp).Round(time.Minute)
		frameInfo = fmt.Sprintf("Frame %d/%d (%s ago)",
			m.currentFrame+1, len(m.radar.frames), timeAgo)
	} else {
		frameInfo = fmt.Sprintf("Frame %d/%d", m.currentFrame+1, len(m.radar.frames))
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
	lines = append(lines, helpStyle.Render(frameInfo+refreshInfo))

	return infoPanelStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
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

	// Draw simple distance markers
	m.drawDistanceMarkers(display, centerX, centerY)

	// Draw precipitation data
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
		y := int(float64(centerY) + float64(radius)*math.Sin(angle*math.Pi/180)*0.5)

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
	// Get lat/lon to determine what features to draw
	lat, lon, _, _, err := geocodeZip(m.zipCode)
	if err != nil {
		// If geocoding fails, just draw the center marker
		if centerY >= 0 && centerY < len(display) && centerX >= 0 && centerX < len(display[0]) {
			display[centerY][centerX] = lipgloss.NewStyle().
				Foreground(lipgloss.Color("226")).
				Bold(true).
				Render("★")
		}
		return
	}

	boundaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	waterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	mountainStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("94"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Scale: approximately 1 character = 4-5 miles
	milesPerCharX := 250.0 / float64(radarWidth)
	milesPerCharY := 150.0 / float64(radarHeight)

	// Helper functions first
	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	abs := func(a int) int {
		if a < 0 {
			return -a
		}
		return a
	}

	// Safe bounds checking function
	inBounds := func(x, y int) bool {
		return y >= 0 && y < len(display) && x >= 0 && x < len(display[0])
	}

	// Helper function to convert lat/lon to display coordinates
	latLonToDisplay := func(targetLat, targetLon float64) (int, int) {
		milesPerDegreeLat := 69.0
		milesPerDegreeLon := 69.0 * math.Cos(lat*math.Pi/180)

		deltaLat := targetLat - lat
		deltaLon := targetLon - lon

		milesNorth := deltaLat * milesPerDegreeLat
		milesEast := deltaLon * milesPerDegreeLon

		x := centerX + int(milesEast/milesPerCharX)
		y := centerY - int(milesNorth/milesPerCharY)

		return x, y
	}

	// Safe drawing helper that checks bounds
	safeDrawPoint := func(x, y int, char string, style *lipgloss.Style) {
		if inBounds(x, y) {
			display[y][x] = style.Render(char)
		}
	}

	// Safe line drawing function
	drawLine := func(x1, y1, x2, y2 int, char string, style *lipgloss.Style, skipExisting bool) {
		// Clip line to display bounds
		if (x1 < 0 && x2 < 0) || (x1 >= radarWidth && x2 >= radarWidth) ||
			(y1 < 0 && y2 < 0) || (y1 >= radarHeight && y2 >= radarHeight) {
			return
		}

		// Simple clipping
		x1 = max(0, min(radarWidth-1, x1))
		x2 = max(0, min(radarWidth-1, x2))
		y1 = max(0, min(radarHeight-1, y1))
		y2 = max(0, min(radarHeight-1, y2))

		if x1 == x2 { // Vertical line
			if y1 > y2 {
				y1, y2 = y2, y1
			}
			for y := y1; y <= y2; y++ {
				if skipExisting && inBounds(x1, y) && display[y][x1] != " " {
					continue
				}
				safeDrawPoint(x1, y, char, style)
			}
		} else if y1 == y2 { // Horizontal line
			if x1 > x2 {
				x1, x2 = x2, x1
			}
			for x := x1; x <= x2; x++ {
				if skipExisting && inBounds(x, y1) && display[y1][x] != " " {
					continue
				}
				safeDrawPoint(x, y1, char, style)
			}
		} else { // Diagonal line
			dx := abs(x2 - x1)
			dy := abs(y2 - y1)
			sx := 1
			sy := 1
			if x1 > x2 {
				sx = -1
			}
			if y1 > y2 {
				sy = -1
			}
			err := dx - dy

			x, y := x1, y1
			for {
				if skipExisting && inBounds(x, y) && display[y][x] != " " {
					// Skip this point
				} else {
					safeDrawPoint(x, y, char, style)
				}

				if x == x2 && y == y2 {
					break
				}

				e2 := 2 * err
				if e2 > -dy {
					err -= dy
					x += sx
				}
				if e2 < dx {
					err += dx
					y += sy
				}
			}
		}
	}

	// Draw state borders using actual state boundary data
	drawStateBorders := func() {
		// Define key state boundary points (simplified)
		borders := [][]float64{
			// === WESTERN STATES ===
			// Washington borders
			{49.0, -117.03, 49.0, -123.0}, // WA-Canada border
			{49.0, -123.0, 48.5, -124.7},  // WA Pacific coast
			{48.5, -124.7, 46.0, -124.0},  // WA Pacific coast
			{46.0, -124.0, 46.0, -117.03}, // WA-OR border
			{46.0, -117.03, 49.0, -117.03}, // WA-ID border
		
			// Oregon borders  
			{46.0, -117.03, 46.0, -124.0}, // OR-WA border
			{46.0, -124.0, 42.0, -124.4},  // OR Pacific coast
			{42.0, -124.4, 42.0, -117.02}, // OR-CA border
			{42.0, -117.02, 45.5, -117.02}, // OR-ID border
			{45.5, -117.02, 46.0, -117.03}, // OR-ID border (northeast)
		
			// California borders
			{42.0, -120.0, 42.0, -124.4},  // CA-OR border
			{42.0, -120.0, 39.0, -120.0},  // CA-NV border (north)
			{39.0, -120.0, 35.0, -119.5},  // CA Sierra Nevada line
			{35.0, -119.5, 35.0, -114.6},  // CA-NV border (south)
			{35.0, -114.6, 32.5, -114.5},  // CA-AZ border
			{32.5, -114.5, 32.5, -117.1},  // CA-Mexico border
			{32.5, -117.1, 42.0, -124.4},  // CA Pacific coast (simplified)
		
			// Idaho borders
			{49.0, -117.03, 49.0, -116.05}, // ID-Canada border
			{49.0, -116.05, 44.5, -111.05}, // ID-MT border
			{44.5, -111.05, 42.0, -111.05}, // ID-WY border
			{42.0, -111.05, 42.0, -114.0},  // ID-UT border
			{42.0, -114.0, 42.0, -117.02},  // ID-NV/OR border
			{42.0, -117.02, 49.0, -117.03}, // ID-WA border
		
			// Nevada borders
			{42.0, -120.0, 42.0, -114.0},  // NV-OR/ID border
			{42.0, -114.0, 37.0, -114.0},  // NV-UT border
			{37.0, -114.0, 35.0, -114.6},  // NV-AZ border
			{35.0, -114.6, 35.0, -120.0},  // NV-CA border (south)
			{35.0, -120.0, 39.0, -120.0},  // NV-CA border (west)
			{39.0, -120.0, 42.0, -120.0},  // NV-CA border (north)
		
			// Utah borders
			{42.0, -114.0, 42.0, -111.05}, // UT-ID border
			{42.0, -111.05, 41.0, -111.05}, // UT-WY border
			{41.0, -111.05, 41.0, -109.05}, // UT-WY border (south)
			{41.0, -109.05, 37.0, -109.05}, // UT-CO border
			{37.0, -109.05, 37.0, -114.0},  // UT-AZ border
			{37.0, -114.0, 42.0, -114.0},   // UT-NV border
		
			// Arizona borders
			{37.0, -114.0, 37.0, -109.05},  // AZ-UT border
			{37.0, -109.05, 31.33, -109.05}, // AZ-NM border
			{31.33, -109.05, 31.33, -111.07}, // AZ-Mexico border (east)
			{31.33, -111.07, 31.33, -114.81}, // AZ-Mexico border
			{31.33, -114.81, 32.5, -114.5},   // AZ-CA border (junction)
			{32.5, -114.5, 35.0, -114.6},     // AZ-CA border
			{35.0, -114.6, 37.0, -114.0},     // AZ-NV border
		
			// === MOUNTAIN STATES ===
			// Montana borders
			{49.0, -116.05, 49.0, -104.03}, // MT-Canada border
			{49.0, -104.03, 45.0, -104.03}, // MT-ND border
			{45.0, -104.03, 45.0, -111.05}, // MT-WY border
			{45.0, -111.05, 48.5, -116.05}, // MT-ID border
			{48.5, -116.05, 49.0, -116.05}, // MT-ID border (north)
		
			// Wyoming borders
			{45.0, -111.05, 45.0, -104.05}, // WY-MT border
			{45.0, -104.05, 41.0, -104.05}, // WY-SD/NE border
			{41.0, -104.05, 41.0, -111.05}, // WY-CO/UT border
			{41.0, -111.05, 45.0, -111.05}, // WY-ID border
		
			// Colorado borders
			{41.0, -109.05, 41.0, -102.05}, // CO-WY/NE border
			{41.0, -102.05, 37.0, -102.05}, // CO-NE/KS border
			{37.0, -102.05, 37.0, -109.05}, // CO-KS/OK/NM border
			{37.0, -109.05, 41.0, -109.05}, // CO-UT border
		
			// New Mexico borders
			{37.0, -109.05, 37.0, -103.0},  // NM-CO border
			{37.0, -103.0, 32.0, -103.0},   // NM-OK/TX border
			{32.0, -103.0, 32.0, -106.5},   // NM-TX border
			{32.0, -106.5, 31.78, -106.5},  // NM-TX border (El Paso)
			{31.78, -106.5, 31.78, -108.2}, // NM-Mexico border
			{31.78, -108.2, 31.33, -109.05}, // NM-Mexico border
			{31.33, -109.05, 37.0, -109.05}, // NM-AZ border
		
			// === MIDWEST STATES ===
			// North Dakota borders
			{49.0, -104.03, 49.0, -97.23},  // ND-Canada border
			{49.0, -97.23, 45.94, -96.56},  // ND-MN border
			{45.94, -96.56, 45.94, -104.03}, // ND-SD border
			{45.94, -104.03, 49.0, -104.03}, // ND-MT border
		
			// South Dakota borders
			{45.94, -104.03, 45.94, -96.44}, // SD-ND border
			{45.94, -96.44, 43.5, -96.44},   // SD-MN/IA border
			{43.5, -96.44, 43.0, -96.44},    // SD-IA border
			{43.0, -96.44, 43.0, -104.05},   // SD-NE border
			{43.0, -104.05, 45.94, -104.03}, // SD-WY/MT border
		
			// Nebraska borders
			{43.0, -104.05, 43.0, -96.44},  // NE-SD border
			{43.0, -96.44, 40.0, -95.31},   // NE-IA border
			{40.0, -95.31, 40.0, -102.05},  // NE-KS border
			{40.0, -102.05, 41.0, -102.05}, // NE-CO border
			{41.0, -102.05, 41.0, -104.05}, // NE-WY border
			{41.0, -104.05, 43.0, -104.05}, // NE-WY border
		
			// Kansas borders
			{40.0, -102.05, 40.0, -94.62},  // KS-NE border
			{40.0, -94.62, 39.0, -94.62},   // KS-MO border
			{39.0, -94.62, 37.0, -94.62},   // KS-MO border
			{37.0, -94.62, 37.0, -102.05},  // KS-OK border
			{37.0, -102.05, 40.0, -102.05}, // KS-CO border
		
			// Minnesota borders
			{49.0, -97.23, 49.0, -95.15},   // MN-Canada border
			{49.0, -95.15, 49.0, -89.53},   // MN-Canada border (east)
			{48.0, -89.53, 47.5, -92.3},    // MN-Lake Superior
			{47.5, -92.3, 46.5, -92.3},     // MN-WI border
			{46.5, -92.3, 45.5, -92.3},     // MN-WI border
			{45.5, -92.3, 43.5, -91.22},    // MN-WI border
			{43.5, -91.22, 43.5, -96.44},   // MN-IA border
			{43.5, -96.44, 45.94, -96.56},  // MN-SD border
			{45.94, -96.56, 49.0, -97.23},  // MN-ND border
		
			// Iowa borders
			{43.5, -96.44, 43.5, -91.22},   // IA-MN border
			{43.5, -91.22, 42.5, -90.64},   // IA-WI border
			{42.5, -90.64, 40.38, -91.41},  // IA-IL border
			{40.38, -91.41, 40.58, -95.77}, // IA-MO border
			{40.58, -95.77, 43.0, -96.44},  // IA-NE border
			{43.0, -96.44, 43.5, -96.44},   // IA-SD border
		
			// Missouri borders
			{40.58, -95.77, 40.38, -91.41}, // MO-IA border
			{40.38, -91.41, 36.5, -89.5},   // MO-IL border
			{36.5, -89.5, 36.0, -89.5},     // MO-KY/TN border
			{36.0, -89.5, 36.5, -90.37},    // MO-AR border
			{36.5, -90.37, 36.5, -94.62},   // MO-AR border
			{36.5, -94.62, 37.0, -94.62},   // MO-OK border
			{37.0, -94.62, 39.0, -94.62},   // MO-KS border
			{39.0, -94.62, 40.58, -95.77},  // MO-KS/NE border
		
			// Wisconsin borders
			{46.5, -92.3, 45.5, -92.3},     // WI-MN border
			{45.5, -92.3, 43.5, -91.22},    // WI-MN border
			{43.5, -91.22, 42.5, -90.64},   // WI-IA border
			{42.5, -90.64, 42.5, -87.02},   // WI-IL border
			{42.5, -87.02, 45.0, -87.0},    // WI-Lake Michigan
			{45.0, -87.0, 45.5, -88.0},     // WI-MI border
			{45.5, -88.0, 46.5, -90.0},     // WI-MI border
			{46.5, -90.0, 46.5, -92.3},     // WI-Lake Superior
		
			// Illinois borders
			{42.5, -90.64, 42.5, -87.02},   // IL-WI border
			{42.5, -87.02, 41.76, -87.53},  // IL-Lake Michigan
			{41.76, -87.53, 39.0, -87.5},   // IL-IN border
			{39.0, -87.5, 37.0, -88.1},     // IL-IN border
			{37.0, -88.1, 37.0, -89.15},    // IL-KY border
			{37.0, -89.15, 36.5, -89.5},    // IL-MO border
			{36.5, -89.5, 40.38, -91.41},   // IL-MO border
			{40.38, -91.41, 42.5, -90.64},  // IL-IA border
		
			// Michigan borders
			{45.0, -87.0, 45.5, -88.0},     // MI-WI border
			{45.5, -88.0, 46.5, -90.0},     // MI-WI border
			{46.5, -90.0, 47.5, -89.0},     // MI-Lake Superior
			{42.0, -86.5, 45.0, -87.0},     // MI-Lake Michigan
			{41.76, -84.8, 42.0, -83.0},    // MI-OH border
			{42.0, -83.0, 42.5, -82.4},     // MI-Canada border
		
			// Indiana borders
			{41.76, -87.53, 41.76, -84.8},  // IN-MI border
			{41.76, -84.8, 39.0, -84.8},    // IN-OH border
			{39.0, -84.8, 38.0, -86.0},     // IN-KY border
			{38.0, -86.0, 37.0, -88.1},     // IN-KY border
			{37.0, -88.1, 39.0, -87.5},     // IN-IL border
			{39.0, -87.5, 41.76, -87.53},   // IN-IL border
		
			// Ohio borders
			{41.76, -84.8, 41.97, -80.52},  // OH-MI/PA border
			{41.97, -80.52, 40.64, -80.52}, // OH-PA border
			{40.64, -80.52, 39.0, -81.0},   // OH-WV border
			{39.0, -81.0, 38.5, -82.0},     // OH-WV border
			{38.5, -82.0, 38.5, -84.8},     // OH-KY border
			{38.5, -84.8, 39.0, -84.8},     // OH-KY border
			{39.0, -84.8, 41.76, -84.8},    // OH-IN border
		
			// === SOUTHERN STATES ===
			// Texas borders
			{36.5, -103.0, 36.5, -100.0},   // TX-OK border (panhandle)
			{36.5, -100.0, 34.0, -100.0},   // TX-OK border (Red River)
			{34.0, -100.0, 33.5, -94.04},   // TX-OK border (Red River)
			{33.5, -94.04, 31.17, -94.04},  // TX-LA border
			{31.17, -94.04, 29.5, -93.84},  // TX-LA border (Sabine)
			{29.5, -93.84, 26.0, -97.14},   // TX Gulf Coast
			{26.0, -97.14, 25.84, -97.14},  // TX-Mexico border (Gulf)
			{25.84, -97.14, 31.78, -106.5}, // TX-Mexico border (Rio Grande)
			{31.78, -106.5, 32.0, -106.5},  // TX-NM border
			{32.0, -106.5, 32.0, -103.0},   // TX-NM border
			{32.0, -103.0, 36.5, -103.0},   // TX-NM/OK border
		
			// Oklahoma borders
			{37.0, -103.0, 37.0, -94.62},   // OK-KS border
			{37.0, -94.62, 36.5, -94.62},   // OK-MO border
			{36.5, -94.62, 35.0, -94.43},   // OK-AR border
			{35.0, -94.43, 33.5, -94.04},   // OK-AR border
			{33.5, -94.04, 34.0, -100.0},   // OK-TX border (Red River)
			{34.0, -100.0, 36.5, -100.0},   // OK-TX border
			{36.5, -100.0, 36.5, -103.0},   // OK-TX border (panhandle)
			{36.5, -103.0, 37.0, -103.0},   // OK-NM/CO border
		
			// Arkansas borders
			{36.5, -94.62, 36.5, -90.37},   // AR-MO border
			{36.5, -90.37, 35.0, -90.0},    // AR-TN border
			{35.0, -90.0, 35.0, -91.0},     // AR-MS border
			{35.0, -91.0, 33.0, -91.2},     // AR-LA border
			{33.0, -91.2, 33.0, -94.04},    // AR-LA border
			{33.0, -94.04, 35.0, -94.43},   // AR-TX border
			{35.0, -94.43, 36.5, -94.62},   // AR-OK border
		
			// Louisiana borders
			{33.0, -94.04, 33.0, -91.2},    // LA-AR border
			{33.0, -91.2, 31.0, -91.5},     // LA-MS border
			{31.0, -91.5, 30.0, -89.5},     // LA-MS border
			{30.0, -89.5, 29.0, -89.0},     // LA Gulf Coast
			{29.0, -89.0, 29.5, -93.84},    // LA Gulf Coast
			{29.5, -93.84, 31.17, -94.04},  // LA-TX border
			{31.17, -94.04, 33.0, -94.04},  // LA-TX border
		
			// Mississippi borders
			{35.0, -91.0, 35.0, -88.2},     // MS-TN border
			{35.0, -88.2, 31.0, -88.47},    // MS-AL border
			{31.0, -88.47, 30.0, -89.5},    // MS Gulf Coast
			{30.0, -89.5, 31.0, -91.5},     // MS-LA border
			{31.0, -91.5, 35.0, -91.0},     // MS-LA border
		
			// Alabama borders
			{35.0, -88.2, 35.0, -85.0},     // AL-TN border
			{35.0, -85.0, 32.9, -85.0},     // AL-GA border
			{32.9, -85.0, 31.0, -85.0},     // AL-FL border
			{31.0, -85.0, 30.0, -87.5},     // AL-FL border
			{30.0, -87.5, 30.0, -88.47},    // AL Gulf Coast
			{30.0, -88.47, 31.0, -88.47},   // AL-MS border
			{31.0, -88.47, 35.0, -88.2},    // AL-MS border
		
			// Tennessee borders
			{36.5, -90.37, 36.5, -81.65},   // TN-KY/VA border
			{36.5, -81.65, 35.0, -84.32},   // TN-NC border
			{35.0, -84.32, 35.0, -85.0},    // TN-GA border
			{35.0, -85.0, 35.0, -88.2},     // TN-AL border
			{35.0, -88.2, 35.0, -90.0},     // TN-MS border
			{35.0, -90.0, 36.5, -90.37},    // TN-AR border
		
			// Kentucky borders
			{39.0, -84.8, 38.5, -84.8},     // KY-OH border
			{38.5, -84.8, 38.5, -82.0},     // KY-OH border
			{38.5, -82.0, 37.5, -82.5},     // KY-WV border
			{37.5, -82.5, 36.5, -83.68},    // KY-VA border
			{36.5, -83.68, 36.5, -89.5},    // KY-TN border
			{36.5, -89.5, 37.0, -89.15},    // KY-MO border
			{37.0, -89.15, 37.0, -88.1},    // KY-IL border
			{37.0, -88.1, 38.0, -86.0},     // KY-IN border
			{38.0, -86.0, 39.0, -84.8},     // KY-IN border
		
			// Florida borders
			{31.0, -87.5, 31.0, -85.0},     // FL-AL border
			{31.0, -85.0, 30.5, -84.86},    // FL-GA border
			{30.5, -84.86, 30.0, -82.0},    // FL-GA border
			{30.0, -82.0, 25.0, -80.0},     // FL Atlantic coast
			{25.0, -80.0, 24.5, -81.8},     // FL Keys
			{24.5, -81.8, 30.0, -87.5},     // FL Gulf coast
			{30.0, -87.5, 31.0, -87.5},     // FL-AL border
		
			// Georgia borders
			{35.0, -85.0, 35.0, -83.5},     // GA-TN/NC border
			{35.0, -83.5, 32.0, -81.0},     // GA-SC border
			{32.0, -81.0, 30.5, -81.5},     // GA Atlantic coast
			{30.5, -81.5, 30.0, -82.0},     // GA-FL border
			{30.0, -82.0, 30.5, -84.86},    // GA-FL border
			{30.5, -84.86, 32.9, -85.0},    // GA-AL border
			{32.9, -85.0, 35.0, -85.0},     // GA-AL border
		
			// South Carolina borders
			{35.0, -83.5, 35.2, -80.5},     // SC-NC border
			{35.2, -80.5, 33.5, -79.0},     // SC Atlantic coast
			{33.5, -79.0, 32.0, -81.0},     // SC-GA border
			{32.0, -81.0, 35.0, -83.5},     // SC-GA border
		
			// North Carolina borders
			{36.5, -83.68, 36.5, -75.5},    // NC-VA border
			{36.5, -75.5, 35.5, -75.5},     // NC Atlantic coast
			{35.5, -75.5, 33.5, -79.0},     // NC Atlantic coast
			{33.5, -79.0, 35.2, -80.5},     // NC-SC border
			{35.2, -80.5, 35.0, -84.32},    // NC-TN border
			{35.0, -84.32, 36.5, -83.68},   // NC-TN/VA border
		
			// Virginia borders
			{39.0, -77.52, 39.0, -75.5},    // VA-MD border
			{39.0, -75.5, 38.0, -75.5},     // VA Atlantic coast
			{38.0, -75.5, 36.5, -75.5},     // VA Atlantic coast
			{36.5, -75.5, 36.5, -83.68},    // VA-NC border
			{36.5, -83.68, 37.5, -82.5},    // VA-KY border
			{37.5, -82.5, 39.0, -80.52},    // VA-WV border
			{39.0, -80.52, 39.0, -77.52},   // VA-MD border
		
			// West Virginia borders
			{40.64, -80.52, 39.72, -79.48}, // WV-PA border
			{39.72, -79.48, 39.0, -77.52},  // WV-MD border
			{39.0, -77.52, 39.0, -80.52},   // WV-VA border
			{39.0, -80.52, 37.5, -82.5},    // WV-VA border
			{37.5, -82.5, 38.5, -82.0},     // WV-KY border
			{38.5, -82.0, 39.0, -81.0},     // WV-OH border
			{39.0, -81.0, 40.64, -80.52},   // WV-OH/PA border
		
			// === NORTHEASTERN STATES ===
			// Pennsylvania borders
			{42.0, -80.52, 42.0, -79.76},   // PA-NY border (west)
			{42.0, -79.76, 41.99, -75.35},  // PA-NY border
			{41.99, -75.35, 41.0, -75.1},   // PA-NJ border
			{41.0, -75.1, 39.72, -75.79},   // PA-DE border
			{39.72, -75.79, 39.72, -79.48}, // PA-MD border
			{39.72, -79.48, 40.64, -80.52}, // PA-WV border
			{40.64, -80.52, 42.0, -80.52},  // PA-OH border
		
			// New York borders
			{45.01, -74.75, 45.01, -71.5},  // NY-Canada border
			{45.01, -71.5, 42.73, -71.5},   // NY-VT border
			{42.73, -71.5, 42.0, -73.35},   // NY-MA/CT border
			{42.0, -73.35, 41.0, -73.9},    // NY-CT border
			{41.0, -73.9, 40.7, -74.0},     // NY Atlantic coast
			{40.7, -74.0, 41.0, -75.1},     // NY-NJ border
			{41.0, -75.1, 41.99, -75.35},   // NY-PA border
			{41.99, -75.35, 42.0, -79.76},  // NY-PA border
			{42.0, -79.76, 45.01, -74.75},  // NY-Canada border
		
			// New Jersey borders
			{41.36, -74.7, 41.0, -73.9},    // NJ-NY border
			{41.0, -73.9, 40.7, -74.0},     // NJ-NY border
			{40.7, -74.0, 39.0, -74.5},     // NJ Atlantic coast
			{39.0, -74.5, 38.8, -75.2},     // NJ-DE border
			{38.8, -75.2, 39.72, -75.79},   // NJ-DE/PA border
			{39.72, -75.79, 41.0, -75.1},   // NJ-PA border
			{41.0, -75.1, 41.36, -74.7},    // NJ-NY border
		
			// Delaware borders
			{39.84, -75.79, 39.72, -75.79}, // DE-PA border
			{39.72, -75.79, 38.8, -75.2},   // DE-NJ border
			{38.8, -75.2, 38.45, -75.05},   // DE Atlantic coast
			{38.45, -75.05, 38.45, -75.79}, // DE-MD border
			{38.45, -75.79, 39.84, -75.79}, // DE-MD border
		
			// Maryland borders
			{39.72, -79.48, 39.72, -75.79}, // MD-PA border
			{39.72, -75.79, 38.45, -75.79}, // MD-DE border
			{38.45, -75.79, 38.0, -76.0},   // MD Chesapeake Bay
			{38.0, -76.0, 38.0, -77.0},     // MD-VA border
			{38.0, -77.0, 39.0, -77.52},    // MD-VA/WV border
			{39.0, -77.52, 39.72, -79.48},  // MD-WV border
		
			// Connecticut borders
			{42.05, -73.48, 42.05, -71.8},  // CT-MA border
			{42.05, -71.8, 41.3, -71.85},   // CT-RI border
			{41.3, -71.85, 41.0, -72.0},    // CT Long Island Sound
			{41.0, -72.0, 41.0, -73.9},     // CT-NY border
			{41.0, -73.9, 42.0, -73.35},    // CT-NY border
			{42.0, -73.35, 42.05, -73.48},  // CT-MA border
		
			// Rhode Island borders
			{42.01, -71.38, 42.01, -71.12}, // RI-MA border
			{42.01, -71.12, 41.3, -71.12},  // RI Atlantic coast
			{41.3, -71.12, 41.3, -71.85},   // RI-CT border
			{41.3, -71.85, 42.01, -71.8},   // RI-CT border
			{42.01, -71.8, 42.01, -71.38},  // RI-MA border
		
			// Massachusetts borders
			{42.88, -73.26, 42.75, -71.0},  // MA-NH/VT border
			{42.75, -71.0, 42.88, -70.5},   // MA-NH border
			{42.88, -70.5, 42.0, -70.0},    // MA Atlantic coast
			{42.0, -70.0, 41.5, -71.12},    // MA Atlantic coast
			{41.5, -71.12, 42.01, -71.38},  // MA-RI border
			{42.01, -71.38, 42.05, -71.8},  // MA-CT border
			{42.05, -71.8, 42.05, -73.48},  // MA-CT border
			{42.05, -73.48, 42.88, -73.26}, // MA-NY border
		
			// Vermont borders
			{45.01, -71.5, 45.01, -73.35},  // VT-Canada border
			{45.01, -73.35, 42.73, -73.26}, // VT-NY border
			{42.73, -73.26, 42.73, -72.46}, // VT-MA border
			{42.73, -72.46, 42.73, -71.5},  // VT-NH border
			{42.73, -71.5, 45.01, -71.5},   // VT-NH border
		
			// New Hampshire borders
			{45.3, -71.08, 45.3, -71.0},    // NH-Canada border
			{45.3, -71.0, 42.88, -70.5},    // NH-ME border
			{42.88, -70.5, 42.75, -71.0},   // NH-MA border
			{42.75, -71.0, 42.73, -72.46},  // NH-MA border
			{42.73, -72.46, 45.01, -71.5},  // NH-VT border
			{45.01, -71.5, 45.3, -71.08},   // NH-Canada border
		
			// Maine borders
			{47.46, -69.23, 45.3, -71.08},  // ME-Canada border
			{45.3, -71.08, 45.3, -71.0},    // ME-NH border
			{45.3, -71.0, 42.88, -70.5},    // ME-NH border
			{42.88, -70.5, 43.5, -70.0},    // ME Atlantic coast
			{43.5, -70.0, 45.0, -67.0},     // ME Atlantic coast
			{45.0, -67.0, 47.46, -69.23},   // ME-Canada border
		
			// === NON-CONTIGUOUS STATES ===
			// Alaska (simplified box)
			{71.5, -156.5, 71.5, -141.0},  // AK north border
			{71.5, -141.0, 54.5, -130.0},  // AK east border
			{54.5, -130.0, 54.5, -173.0},  // AK south border
			{54.5, -173.0, 71.5, -156.5},  // AK west border
		
			// Hawaii (simplified boxes for main islands)
			{22.2, -159.8, 22.2, -159.3},  // Kauai
			{22.2, -159.3, 21.8, -159.3},
			{21.8, -159.3, 21.8, -159.8},
			{21.8, -159.8, 22.2, -159.8},
		
			{21.1, -156.3, 21.1, -155.9},  // Maui
			{21.1, -155.9, 20.5, -155.9},
			{20.5, -155.9, 20.5, -156.7},
			{20.5, -156.7, 21.1, -156.3},
		
			{21.7, -158.3, 21.7, -157.6},  // Oahu
			{21.7, -157.6, 21.2, -157.6},
			{21.2, -157.6, 21.2, -158.3},
			{21.2, -158.3, 21.7, -158.3},
		
			{19.7, -156.1, 19.7, -154.8},  // Big Island
			{19.7, -154.8, 18.9, -154.8},
			{18.9, -154.8, 18.9, -156.1},
			{18.9, -156.1, 19.7, -156.1},
		}

		// Draw each border segment
		for _, border := range borders {
			startX, startY := latLonToDisplay(border[0], border[1])
			endX, endY := latLonToDisplay(border[2], border[3])

			// Determine if vertical or horizontal
			if abs(startX-endX) < abs(startY-endY) {
				drawLine(startX, startY, endX, endY, "│", &borderStyle, false)
			} else {
				drawLine(startX, startY, endX, endY, "─", &borderStyle, false)
			}
		}

		// Add state abbreviations
		stateLabels := []struct {
			lat, lon float64
			label    string
		}{
			{44.5, -100.0, "SD"}, {41.5, -99.0, "NE"}, {42.0, -93.5, "IA"},
			{46.0, -94.5, "MN"}, {43.0, -89.5, "WI"}, {40.0, -89.0, "IL"},
			{38.5, -98.5, "KS"}, {39.0, -105.5, "CO"}, {44.0, -107.5, "WY"},
			{47.0, -110.0, "MT"}, {46.5, -100.5, "ND"}, {38.5, -92.5, "MO"},
			{35.0, -97.5, "OK"}, {31.0, -99.0, "TX"}, {40.5, -112.0, "UT"},
			{39.0, -119.5, "NV"}, {37.5, -119.5, "CA"}, {44.0, -120.5, "OR"},
			{47.5, -120.5, "WA"}, {43.5, -114.0, "ID"}, {34.5, -106.0, "NM"},
			{34.5, -112.0, "AZ"}, {42.5, -72.5, "VT"}, {43.5, -71.5, "NH"},
			{42.3, -71.8, "MA"}, {41.7, -71.5, "RI"}, {41.6, -72.7, "CT"},
			{43.0, -75.5, "NY"}, {40.5, -74.5, "NJ"}, {41.0, -77.5, "PA"},
			{39.0, -75.5, "DE"}, {39.0, -76.5, "MD"}, {38.0, -79.5, "VA"},
			{35.5, -79.5, "NC"}, {34.0, -81.0, "SC"}, {33.0, -83.5, "GA"},
			{30.5, -84.5, "FL"}, {32.5, -86.5, "AL"}, {32.5, -90.0, "MS"},
			{31.0, -92.0, "LA"}, {35.0, -86.0, "TN"}, {37.5, -84.5, "KY"},
			{40.0, -82.5, "OH"}, {40.0, -86.0, "IN"}, {42.0, -84.5, "MI"},
			{38.5, -81.0, "WV"}, {35.5, -92.5, "AR"},
		}

		// Draw visible state labels
		for _, state := range stateLabels {
			x, y := latLonToDisplay(state.lat, state.lon)
			if inBounds(x, y) && x+len(state.label)-1 < len(display[0]) {
				for i, ch := range state.label {
					if inBounds(x+i, y) {
						display[y][x+i] = boundaryStyle.Render(string(ch))
					}
				}
			}
		}
	}

	// Draw state borders first
	drawStateBorders()

	// Then draw geographic features on top
	// Draw major rivers
	rivers := []struct {
		name string
		path [][]float64
	}{
		{
			"Mississippi",
			[][]float64{
				{47.5, -94.5}, {46.0, -94.0}, {44.0, -92.0}, {42.0, -90.5},
				{40.0, -90.0}, {38.0, -89.5}, {36.0, -89.5}, {34.0, -90.5},
				{32.0, -91.0}, {30.0, -91.0}, {29.0, -89.5},
			},
		},
		{
			"Missouri",
			[][]float64{
				{46.0, -111.5}, {45.5, -110.0}, {44.5, -108.0}, {43.5, -104.0},
				{42.5, -100.0}, {41.5, -96.0}, {40.0, -95.0}, {39.0, -93.5},
				{38.5, -90.5},
			},
		},
		{
			"Colorado",
			[][]float64{
				{36.0, -114.5}, {35.5, -113.0}, {34.5, -111.0}, {33.5, -109.0},
				{32.5, -107.5}, {31.5, -105.5},
			},
		},
		{
			"Rio Grande",
			[][]float64{
				{37.0, -107.0}, {36.0, -106.0}, {34.0, -106.5}, {32.0, -106.5},
				{30.0, -104.0}, {28.0, -102.0}, {26.0, -99.0}, {25.8, -97.2},
			},
		},
	}

	// Draw rivers
	for _, river := range rivers {
		for i := 0; i < len(river.path)-1; i++ {
			x1, y1 := latLonToDisplay(river.path[i][0], river.path[i][1])
			x2, y2 := latLonToDisplay(river.path[i+1][0], river.path[i+1][1])

			steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
			if steps > 0 {
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					x := int(float64(x1) + t*float64(x2-x1))
					y := int(float64(y1) + t*float64(y2-y1))

					if inBounds(x, y) && display[y][x] == " " {
						display[y][x] = waterStyle.Render("~")
					}
				}
			}
		}
	}

	// Draw mountain ranges
	mountains := []struct {
		name string
		path [][]float64
	}{
		{
			"Rockies",
			[][]float64{
				{49.0, -114.0}, {47.0, -113.5}, {45.0, -112.5}, {43.0, -109.0},
				{41.0, -105.5}, {39.0, -105.5}, {37.0, -105.0}, {35.0, -106.0},
			},
		},
		{
			"Cascades",
			[][]float64{
				{49.0, -121.5}, {47.5, -121.5}, {46.0, -121.7}, {44.0, -122.0},
				{42.0, -122.2}, {40.5, -122.0},
			},
		},
		{
			"Sierra Nevada",
			[][]float64{
				{40.5, -121.0}, {39.0, -120.5}, {37.5, -119.0}, {36.0, -118.0},
			},
		},
		{
			"Appalachians",
			[][]float64{
				{44.0, -71.5}, {42.0, -73.5}, {40.0, -75.5}, {38.0, -78.5},
				{36.0, -81.5}, {34.5, -83.5},
			},
		},
	}

	// Draw mountains
	for _, mountain := range mountains {
		for _, point := range mountain.path {
			x, y := latLonToDisplay(point[0], point[1])
			if inBounds(x, y) && display[y][x] == " " {
				display[y][x] = mountainStyle.Render("^")
			}
			// Add some width to mountain ranges
			if inBounds(x-1, y) && display[y][x-1] == " " {
				display[y][x-1] = mountainStyle.Render("^")
			}
			if inBounds(x+1, y) && display[y][x+1] == " " {
				display[y][x+1] = mountainStyle.Render("^")
			}
		}
	}

	// Draw coastlines
	// Atlantic Coast
	if lon > -85 {
		coastPoints := [][]float64{
			{45.0, -67.0}, {44.0, -68.0}, {42.5, -70.0}, {41.0, -71.0},
			{40.5, -73.5}, {39.0, -74.0}, {37.5, -75.5}, {36.0, -76.0},
			{34.0, -78.0}, {32.0, -80.0}, {30.0, -81.0}, {28.0, -80.5},
			{25.5, -80.0}, {24.5, -81.5},
		}

		for i := 0; i < len(coastPoints)-1; i++ {
			x1, y1 := latLonToDisplay(coastPoints[i][0], coastPoints[i][1])
			x2, y2 := latLonToDisplay(coastPoints[i+1][0], coastPoints[i+1][1])

			steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
			if steps > 0 {
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					x := int(float64(x1) + t*float64(x2-x1))
					y := int(float64(y1) + t*float64(y2-y1))

					if inBounds(x, y) && display[y][x] == " " {
						display[y][x] = waterStyle.Render("≈")
					}
				}
			}
		}
	}

	// Pacific Coast
	if lon < -115 {
		coastPoints := [][]float64{
			{48.5, -124.7}, {47.0, -124.0}, {45.0, -124.0}, {43.0, -124.4},
			{41.0, -124.2}, {39.0, -123.8}, {37.0, -122.5}, {35.0, -121.0},
			{33.5, -118.0}, {32.5, -117.2},
		}

		for i := 0; i < len(coastPoints)-1; i++ {
			x1, y1 := latLonToDisplay(coastPoints[i][0], coastPoints[i][1])
			x2, y2 := latLonToDisplay(coastPoints[i+1][0], coastPoints[i+1][1])

			steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
			if steps > 0 {
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					x := int(float64(x1) + t*float64(x2-x1))
					y := int(float64(y1) + t*float64(y2-y1))

					if inBounds(x, y) && display[y][x] == " " {
						display[y][x] = waterStyle.Render("≈")
					}
				}
			}
		}
	}

	// Gulf Coast
	if lat < 33 && lon > -98 {
		coastPoints := [][]float64{
			{30.0, -87.5}, {29.5, -89.0}, {29.0, -91.0}, {28.5, -93.0},
			{27.5, -95.0}, {26.5, -97.0}, {25.8, -97.2},
		}

		for i := 0; i < len(coastPoints)-1; i++ {
			x1, y1 := latLonToDisplay(coastPoints[i][0], coastPoints[i][1])
			x2, y2 := latLonToDisplay(coastPoints[i+1][0], coastPoints[i+1][1])

			steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
			if steps > 0 {
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					x := int(float64(x1) + t*float64(x2-x1))
					y := int(float64(y1) + t*float64(y2-y1))

					if inBounds(x, y) && display[y][x] == " " {
						display[y][x] = waterStyle.Render("≈")
					}
				}
			}
		}
	}

	// Great Lakes
	if lon > -93 && lon < -75 && lat > 41 && lat < 49 {
		// Lake Superior
		if lat > 46 {
			lakePoints := [][]float64{
				{48.0, -89.5}, {47.5, -91.0}, {46.5, -92.0}, {46.5, -94.0},
				{47.0, -92.5}, {47.5, -90.5}, {48.0, -89.5},
			}
			for _, point := range lakePoints {
				x, y := latLonToDisplay(point[0], point[1])
				if inBounds(x, y) {
					display[y][x] = waterStyle.Render("≈")
				}
			}
		}

		// Lake Michigan
		if lon > -88 && lon < -85 {
			for dlat := -2.0; dlat <= 2.0; dlat += 0.5 {
				x, y := latLonToDisplay(lat+dlat, lon+1.5)
				if inBounds(x, y) {
					display[y][x] = waterStyle.Render("≈")
				}
			}
		}
	}

	// Add city marker for the center (on top of everything)
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
	m.animationActive = false
	return m
}

// Weather helper functions
func getWeatherEmoji(conditions string) string {
	if conditions == "" {
		return ""
	}

	cond := strings.ToLower(conditions)

	switch {
	case strings.Contains(cond, "thunder") || strings.Contains(cond, "storm"):
		return "⛈️"
	case strings.Contains(cond, "snow") || strings.Contains(cond, "blizzard"):
		return "🌨️"
	case strings.Contains(cond, "rain") || strings.Contains(cond, "shower"):
		if strings.Contains(cond, "heavy") {
			return "🌧️"
		}
		return "🌦️"
	case strings.Contains(cond, "drizzle") || strings.Contains(cond, "mist"):
		return "🌫️"
	case strings.Contains(cond, "cloud"):
		if strings.Contains(cond, "partly") || strings.Contains(cond, "few") {
			return "⛅"
		}
		return "☁️"
	case strings.Contains(cond, "clear") || strings.Contains(cond, "sunny"):
		hour := time.Now().Hour()
		if hour >= 6 && hour < 18 {
			return "☀️"
		}
		return "🌙"
	case strings.Contains(cond, "fog"):
		return "🌫️"
	case strings.Contains(cond, "wind"):
		return "💨"
	case strings.Contains(cond, "hail"):
		return "🌨️"
	default:
		return "🌤️"
	}
}

func getAlertDisplay(alerts []weatherAlert) (emoji string, color lipgloss.Color, text string) {
	if len(alerts) == 0 {
		return "", lipgloss.Color(""), ""
	}

	// Find the most severe alert
	var mostSevere weatherAlert
	severityRank := map[string]int{
		"Extreme":  4,
		"Severe":   3,
		"Moderate": 2,
		"Minor":    1,
		"Unknown":  0,
	}

	maxSeverity := -1
	for _, alert := range alerts {
		rank := severityRank[alert.severity]
		if rank > maxSeverity {
			maxSeverity = rank
			mostSevere = alert
		}
	}

	// Determine emoji and color based on event type and severity
	switch {
	case strings.Contains(strings.ToLower(mostSevere.event), "tornado"):
		emoji = "🌪️"
		color = lipgloss.Color("196")
		text = "TORNADO " + strings.ToUpper(getAlertType(mostSevere.event))

	case strings.Contains(strings.ToLower(mostSevere.event), "severe thunderstorm"):
		emoji = "⛈️"
		color = lipgloss.Color("208")
		text = "SEVERE T-STORM " + strings.ToUpper(getAlertType(mostSevere.event))

	case strings.Contains(strings.ToLower(mostSevere.event), "flood"):
		emoji = "🌊"
		color = lipgloss.Color("33")
		text = "FLOOD " + strings.ToUpper(getAlertType(mostSevere.event))

	case strings.Contains(strings.ToLower(mostSevere.event), "winter") ||
		strings.Contains(strings.ToLower(mostSevere.event), "snow") ||
		strings.Contains(strings.ToLower(mostSevere.event), "blizzard"):
		emoji = "❄️"
		color = lipgloss.Color("51")
		text = strings.ToUpper(getAlertType(mostSevere.event))

	case strings.Contains(strings.ToLower(mostSevere.event), "heat"):
		emoji = "🔥"
		color = lipgloss.Color("202")
		text = "HEAT " + strings.ToUpper(getAlertType(mostSevere.event))

	case strings.Contains(strings.ToLower(mostSevere.event), "wind"):
		emoji = "💨"
		color = lipgloss.Color("226")
		text = "WIND " + strings.ToUpper(getAlertType(mostSevere.event))

	default:
		emoji = "⚠️"
		if mostSevere.severity == "Extreme" {
			color = lipgloss.Color("196")
		} else if mostSevere.severity == "Severe" {
			color = lipgloss.Color("208")
		} else {
			color = lipgloss.Color("226")
		}
		text = strings.ToUpper(getAlertType(mostSevere.event))
	}

	return emoji, color, text
}

func getAlertType(event string) string {
	switch {
	case strings.Contains(event, "Warning"):
		return "WARNING"
	case strings.Contains(event, "Watch"):
		return "WATCH"
	case strings.Contains(event, "Advisory"):
		return "ADVISORY"
	default:
		return "ALERT"
	}
}

func fetchWeatherAlerts(lat, lon float64) []weatherAlert {
	client := &http.Client{Timeout: 5 * time.Second}

	alertsURL := fmt.Sprintf("https://api.weather.gov/alerts/active?point=%.4f,%.4f", lat, lon)

	resp, err := client.Get(alertsURL)
	if err != nil {
		log.Printf("Failed to fetch weather alerts: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var alertsData struct {
		Features []struct {
			Properties struct {
				Event       string    `json:"event"`
				Severity    string    `json:"severity"`
				Urgency     string    `json:"urgency"`
				Headline    string    `json:"headline"`
				Description string    `json:"description"`
				Expires     time.Time `json:"expires"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&alertsData); err != nil {
		log.Printf("Failed to decode alerts: %v", err)
		return nil
	}

	var alerts []weatherAlert
	for _, feature := range alertsData.Features {
		alert := weatherAlert{
			event:       feature.Properties.Event,
			severity:    feature.Properties.Severity,
			urgency:     feature.Properties.Urgency,
			headline:    feature.Properties.Headline,
			description: feature.Properties.Description,
			expires:     feature.Properties.Expires,
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// Animation commands
func (m model) animateFrame() tea.Cmd {
	return tea.Tick(m.frameRate, func(t time.Time) tea.Msg {
		return frameTickMsg(t)
	})
}

func (m model) scheduleRefresh() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return refreshTickMsg(t)
	})
}

func (m model) trackProgress() tea.Cmd {
	return func() tea.Msg {
		for i := 0; i <= 100; i += 10 {
			time.Sleep(200 * time.Millisecond)
		}
		return nil
	}
}

// Data loading
func loadRadarData(zipCode string) tea.Cmd {
	return func() tea.Msg {
		lat, lon, city, state, err := geocodeZip(zipCode)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to geocode ZIP: %w", err)}
		}

		station, err := getNWSRadarStation(lat, lon)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to get radar station: %w", err)}
		}

		temperature, conditions := fetchCurrentWeather(lat, lon)
		alerts := fetchWeatherAlerts(lat, lon)

		frames, isRealData, err := fetchRealRadarData(station, lat, lon)
		if err != nil {
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
				temperature: temperature,
				conditions:  conditions,
				alerts:      alerts,
			},
		}
	}
}

func fetchRealRadarData(station string, lat, lon float64) ([]radarFrame, bool, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	frames := []radarFrame{}

	// First try RainViewer
	frames, err := fetchFromRainViewer(lat, lon)
	if err == nil && len(frames) > 0 {
		log.Printf("Successfully fetched %d frames from RainViewer", len(frames))
		return frames, true, nil
	}

	// Fallback to Iowa State University
	baseTime := time.Now().UTC()

	for i := 0; i < 24; i++ {
		frameTime := baseTime.Add(time.Duration(-i*5) * time.Minute)

		minutes := frameTime.Minute()
		minutes = (minutes / 5) * 5
		frameTime = time.Date(frameTime.Year(), frameTime.Month(), frameTime.Day(),
			frameTime.Hour(), minutes, 0, 0, time.UTC)

		timeStr := frameTime.Format("200601021504")
		radarURL := fmt.Sprintf("https://mesonet.agron.iastate.edu/cgi-bin/wms/nexrad/n0r.cgi?SERVICE=WMS&VERSION=1.1.1&REQUEST=GetMap&FORMAT=image/png&TRANSPARENT=true&LAYERS=nexrad-n0r&WIDTH=%d&HEIGHT=%d&SRS=EPSG:4326&BBOX=%f,%f,%f,%f&TIME=%s",
			radarWidth*4, radarHeight*4,
			lon-2.5, lat-2.0, lon+2.5, lat+2.0,
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

		img, err := png.Decode(resp.Body)
		if err != nil {
			continue
		}

		data := imageToRadarData(img)
		if data != nil {
			frame := radarFrame{
				data:      data,
				timestamp: frameTime,
				product:   "N0R",
			}
			frames = append(frames, frame)
		}

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

	for _, past := range apiData.Radar.Past {
		zoom := 7
		tileX, tileY := latLonToTile(lat, lon, zoom)

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

	data := make([][]int, radarHeight)
	for i := range data {
		data[i] = make([]int, radarWidth)
	}

	foundPrecipitation := false

	for y := 0; y < radarHeight; y++ {
		for x := 0; x < radarWidth; x++ {
			imgX := x * width / radarWidth
			imgY := y * height / radarHeight

			c := img.At(imgX, imgY)
			r, g, b, a := c.RGBA()

			if a < 128 {
				continue
			}

			intensity := 0

			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if r8 > 200 && g8 < 100 && b8 < 100 {
				intensity = 8 + int((r8-200)/28)
				foundPrecipitation = true
			} else if r8 > 200 && g8 > 150 && b8 < 100 {
				intensity = 5 + int((r8-200)/50)
				foundPrecipitation = true
			} else if g8 > 150 && r8 < 150 && b8 < 100 {
				intensity = 1 + int((g8-150)/50)
				foundPrecipitation = true
			} else if b8 > 150 && r8 < 100 && g8 < 150 {
				intensity = 1
				foundPrecipitation = true
			} else if r8 > 100 && g8 > 100 && b8 < 50 {
				intensity = 4
				foundPrecipitation = true
			}

			if intensity > 10 {
				intensity = 10
			}
			if intensity < 0 {
				intensity = 0
			}

			data[y][x] = intensity
		}
	}

	if foundPrecipitation {
		log.Printf("Found precipitation in radar image")
	} else {
		log.Printf("No precipitation detected in radar image")
	}

	return data
}

func generateRadarFrames(station string, count int) []radarFrame {
	frames := make([]radarFrame, count)

	for i := 0; i < count; i++ {
		data := make([][]int, radarHeight)
		for y := range data {
			data[y] = make([]int, radarWidth)
		}

		numCells := 2 + i%3
		for c := 0; c < numCells; c++ {
			centerX := 10 + (i*3+c*15)%radarWidth
			centerY := 5 + (i*2+c*10)%radarHeight
			intensity := 5 + c*2

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

func geocodeZip(zipCode string) (float64, float64, string, string, error) {
	url := fmt.Sprintf("https://api.zippopotam.us/us/%s", zipCode)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return geocodeZipAlternative(zipCode)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return geocodeZipAlternative(zipCode)
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
		return geocodeZipAlternative(zipCode)
	}

	if len(result.Places) == 0 {
		return geocodeZipAlternative(zipCode)
	}

	place := result.Places[0]

	lat, err := strconv.ParseFloat(place.Latitude, 64)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("invalid latitude for ZIP %s", zipCode)
	}

	lon, err := strconv.ParseFloat(place.Longitude, 64)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("invalid longitude for ZIP %s", zipCode)
	}

	return lat, lon, place.PlaceName, place.StateCode, nil
}

func geocodeZipAlternative(zipCode string) (float64, float64, string, string, error) {
	url := fmt.Sprintf("https://api.geocod.io/v1.7/geocode?q=%s&api_key=demo", zipCode)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("failed to geocode ZIP %s: %w", zipCode, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, "", "", fmt.Errorf("unable to find location for ZIP %s", zipCode)
	}

	var result struct {
		Results []struct {
			AddressComponents struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"address_components"`
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, "", "", fmt.Errorf("failed to decode geocoding response: %w", err)
	}

	if len(result.Results) == 0 {
		return 0, 0, "", "", fmt.Errorf("no results found for ZIP %s", zipCode)
	}

	r := result.Results[0]
	return r.Location.Lat, r.Location.Lng, r.AddressComponents.City, r.AddressComponents.State, nil
}

func getNWSRadarStation(lat, lon float64) (string, error) {
	stations := []struct {
		id   string
		lat  float64
		lon  float64
	}{
		{"KOKX", 40.8653, -72.8639},  // New York
		{"KLOT", 41.6045, -88.0847},  // Chicago
		{"KAMX", 25.6111, -80.4128},  // Miami
		{"KATX", 48.1945, -122.4958}, // Seattle
		{"KFWS", 32.5731, -97.3031},  // Dallas
		{"KLVX", 37.9753, -85.9439},  // Louisville
		{"KTFX", 47.4595, -111.3855}, // Great Falls
		{"KSGF", 37.2355, -93.4003},  // Springfield
		{"KLAS", 36.0558, -115.1622}, // Las Vegas
		{"KPHX", 33.4301, -112.0128}, // Phoenix
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

func fetchCurrentWeather(lat, lon float64) (int, string) {
	client := &http.Client{Timeout: 5 * time.Second}

	pointURL := fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", lat, lon)

	resp, err := client.Get(pointURL)
	if err != nil {
		log.Printf("Failed to get NWS point data: %v", err)
		return 0, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("NWS point API returned status: %d", resp.StatusCode)
		return 0, ""
	}

	var pointData struct {
		Properties struct {
			ForecastURL    string `json:"forecast"`
			ObservationURL string `json:"observationStations"`
		} `json:"properties"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pointData); err != nil {
		log.Printf("Failed to decode NWS point data: %v", err)
		return 0, ""
	}

	stationsResp, err := client.Get(pointData.Properties.ObservationURL)
	if err != nil {
		log.Printf("Failed to get observation stations: %v", err)
		return 0, ""
	}
	defer stationsResp.Body.Close()

	var stationsData struct {
		Features []struct {
			Properties struct {
				StationIdentifier string `json:"stationIdentifier"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.NewDecoder(stationsResp.Body).Decode(&stationsData); err != nil {
		log.Printf("Failed to decode stations data: %v", err)
		return 0, ""
	}

	if len(stationsData.Features) == 0 {
		log.Printf("No observation stations found")
		return 0, ""
	}

	stationID := stationsData.Features[0].Properties.StationIdentifier
	obsURL := fmt.Sprintf("https://api.weather.gov/stations/%s/observations/latest", stationID)

	obsResp, err := client.Get(obsURL)
	if err != nil {
		log.Printf("Failed to get observations: %v", err)
		return 0, ""
	}
	defer obsResp.Body.Close()

	var obsData struct {
		Properties struct {
			Temperature struct {
				Value    float64 `json:"value"`
				UnitCode string  `json:"unitCode"`
			} `json:"temperature"`
			TextDescription string `json:"textDescription"`
		} `json:"properties"`
	}

	if err := json.NewDecoder(obsResp.Body).Decode(&obsData); err != nil {
		log.Printf("Failed to decode observation data: %v", err)
		return 0, ""
	}

	temp := obsData.Properties.Temperature.Value
	unitCode := obsData.Properties.Temperature.UnitCode
	
	// Log for debugging
	log.Printf("Temperature value: %f, unit: %s", temp, unitCode)
	
	// Check for Celsius in various formats the API might return
	if strings.Contains(strings.ToLower(unitCode), "degc") || 
	   strings.Contains(strings.ToLower(unitCode), "celsius") ||
	   unitCode == "wmoUnit:degC" ||
	   unitCode == "unit:degC" {
		temp = temp*9/5 + 32
		log.Printf("Converted from Celsius to Fahrenheit: %f", temp)
	}

	conditions := obsData.Properties.TextDescription
	if conditions == "" {
		conditions = "Clear"
	}

	return int(temp), conditions
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}