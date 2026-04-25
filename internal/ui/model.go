package ui

import (
	"fmt"
	"math"
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
	geoOverlay          [][]string // cached geographic overlay (computed once per location)
	sweepAngle          int        // radar sweep animation angle during loading
	zoomLevel           float64    // zoom multiplier (1.0 = default, >1 = zoomed in)
	mouseX, mouseY      int        // mouse position for hover inspection
	hoverIntensity      int        // precipitation intensity at hover point (-1 = no hover)
	forecastIdx         int        // current forecast period being displayed
	forecastTick        int        // counter to slow forecast rotation vs frame rate
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
		animationActive:  false,
		zoomLevel:        1.0,
		hoverIntensity:   -1,
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
				rw, rh := config.EffectiveRadarSize(m.width, m.height)
				cmds = append(cmds,
					m.spinner.Tick,
					radar.LoadData(m.zipCode, rw, rh),
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
				rw, rh := config.EffectiveRadarSize(m.width, m.height)
				cmds = append(cmds,
					m.spinner.Tick,
					radar.LoadData(m.zipCode, rw, rh),
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
		case "z":
			if m.state == StateDisplaying && m.zoomLevel < 4.0 {
				m.zoomLevel *= 1.5
				m.computeGeoOverlay()
			}
		case "x":
			if m.state == StateDisplaying && m.zoomLevel > 0.4 {
				m.zoomLevel /= 1.5
				m.computeGeoOverlay()
			}
		}

	case tea.MouseMsg:
		if m.state == StateDisplaying && msg.Action == tea.MouseActionMotion {
			m.mouseX = msg.X
			m.mouseY = msg.Y
			// Approximate radar grid offset (accounting for borders, padding, chrome)
			// Radar container has border(1) + padding(1) on each side = 4 chars offset
			rw, rh := m.radarSize()
			gridX := msg.X - 4
			gridY := msg.Y - 10 // approximate offset for header + info panel
			m.hoverIntensity = -1
			if gridX >= 0 && gridX < rw && gridY >= 0 && gridY < rh &&
				len(m.radar.Frames) > 0 && m.currentFrame < len(m.radar.Frames) {
				frame := m.radar.Frames[m.currentFrame]
				if gridY < len(frame.Data) && gridX < len(frame.Data[gridY]) {
					m.hoverIntensity = frame.Data[gridY][gridX]
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == StateDisplaying && len(m.radar.Frames) > 0 {
			m.computeGeoOverlay()
		}

	case spinner.TickMsg:
		if m.state == StateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			m.sweepAngle = (m.sweepAngle + 30) % 360
			cmds = append(cmds, cmd)
		}

	case ProgressMsg:
		if m.state == StateLoading {
			pct := float64(msg)
			if pct > 0.9 {
				pct = 0.9 // cap at 90% until data actually loads
			}
			cmd := m.progress.SetPercent(pct)
			cmds = append(cmds, cmd)
			if pct < 0.9 {
				cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
					return ProgressMsg(pct + 0.15)
				}))
			}
		}

	case radar.LoadedMsg:
		m.radar = msg.Radar
		m.computeGeoOverlay()

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
			rw, rh := config.EffectiveRadarSize(m.width, m.height)
			cmds = append(cmds, radar.LoadData(m.zipCode, rw, rh))
			cmds = append(cmds, m.ScheduleRefresh())
		}

	case FrameTickMsg:
		if m.state == StateDisplaying && m.animationActive && !m.isPaused && len(m.radar.Frames) > 0 {
			m.currentFrame = (m.currentFrame + 1) % len(m.radar.Frames)
			// Rotate forecast every 6 frame ticks (~2s at default speed)
			m.forecastTick++
			if m.forecastTick >= 6 && len(m.radar.Forecast) > 0 {
				m.forecastTick = 0
				m.forecastIdx = (m.forecastIdx + 1) % len(m.radar.Forecast)
			}
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
		content = m.renderDisplaying(header)

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

	// Mini radar sweep animation
	sweep := m.renderRadarSweep()

	status := lipgloss.NewStyle().Foreground(config.RadarGreen).Render(messages[messageIdx])

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		sweep,
		"",
		status,
		progress,
	)
}

func (m Model) renderRadarSweep() string {
	size := 9 // 9x9 grid for the mini sweep
	center := size / 2
	sweepStyle := lipgloss.NewStyle().Foreground(config.RadarGreen)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	brightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF66"))

	var lines []string
	for y := 0; y < size; y++ {
		var line string
		for x := 0; x < size; x++ {
			dx := float64(x - center)
			dy := float64(y-center) * 2 // compensate for char aspect ratio
			dist := math.Sqrt(dx*dx + dy*dy)
			angle := math.Atan2(dy, dx) * 180 / math.Pi
			if angle < 0 {
				angle += 360
			}

			if x == center && y == center {
				line += brightStyle.Render("+")
			} else if dist < 1.5 {
				line += sweepStyle.Render("·")
			} else if dist <= float64(center)+0.5 {
				// Check if this point is on the sweep line
				sweepAngle := float64(m.sweepAngle)
				angleDiff := angle - sweepAngle
				if angleDiff < 0 {
					angleDiff += 360
				}
				if angleDiff > 180 {
					angleDiff = 360 - angleDiff
				}

				if angleDiff < 15 {
					line += brightStyle.Render("█")
				} else if angleDiff < 40 {
					line += sweepStyle.Render("▓")
				} else if angleDiff < 70 {
					line += sweepStyle.Render("░")
				} else if int(dist+0.5) == center { // outer ring
					line += dimStyle.Render("·")
				} else {
					line += " "
				}
			} else {
				line += " "
			}
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func (m Model) renderDisplaying(header string) string {
	availH := m.height - 2 // AppStyle padding

	// Always render all chrome — use compact forms when tight on space
	compact := m.height < 30

	// 1. Header (always)
	hdr := header
	used := countLines(hdr) + 1 // +1 for margin

	// 2. Info — full panel or compact single line
	var info string
	if compact {
		info = m.renderInfoCompact()
	} else {
		info = m.renderInfoPanel()
	}
	used += countLines(info)

	// 3. Controls + forecast (always, 2 lines)
	ctrl := m.renderControls()
	used += countLines(ctrl)

	// 4. Radar container chrome: border(2) + padding(2) + frame dots(1) = 5
	containerChrome := 5
	showExtras := false
	if availH-used-containerChrome > 10 {
		showExtras = true
		containerChrome += 2 // scale + legend
	}
	used += containerChrome

	radarH := availH - used
	if radarH < 1 {
		radarH = 1
	}

	radarView := m.renderRadarFrameSized(radarH, showExtras)

	return lipgloss.JoinVertical(lipgloss.Left, hdr, info, radarView, ctrl)
}

// renderInfoCompact renders a single-line info bar for small terminals
func (m Model) renderInfoCompact() string {
	parts := []string{
		config.LocationStyle.Render(m.radar.Location),
		config.StationStyle.Render(m.radar.Station),
	}

	if m.radar.Temperature != 0 {
		tempColor := lipgloss.Color("226")
		if m.radar.Temperature >= 90 {
			tempColor = lipgloss.Color("196")
		} else if m.radar.Temperature >= 70 {
			tempColor = lipgloss.Color("214")
		} else if m.radar.Temperature >= 32 {
			tempColor = lipgloss.Color("87")
		} else if m.radar.Temperature < 32 {
			tempColor = lipgloss.Color("51")
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(tempColor).Bold(true).Render(fmt.Sprintf("%d°F", m.radar.Temperature)))
	}

	if m.radar.Wind.Speed > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Render(fmt.Sprintf("%s%.0fmph", m.radar.Wind.Arrow, m.radar.Wind.Speed)))
	}

	if len(m.radar.Frames) > 0 && m.currentFrame < len(m.radar.Frames) {
		frame := m.radar.Frames[m.currentFrame]
		t := frame.Timestamp.Local().Format("3:04PM")
		parts = append(parts, config.HelpStyle.Render(fmt.Sprintf("F%d/%d %s", m.currentFrame+1, len(m.radar.Frames), t)))
	}

	return config.HelpStyle.Render(strings.Join(parts, "  "))
}

func (m Model) renderRadarFrameSized(radarH int, showExtras bool) string {
	if len(m.radar.Frames) == 0 {
		return "No radar data available"
	}

	frame := m.radar.Frames[m.currentFrame]
	rw := m.width - 8 // AppStyle padding(4) + container border(2) + container padding(2)
	if rw < 5 {
		rw = 5
	}
	rh := radarH

	// Copy the cached geographic overlay
	display := make([][]string, rh)
	for i := range display {
		display[i] = make([]string, rw)
		if i < len(m.geoOverlay) && len(m.geoOverlay[i]) == rw {
			copy(display[i], m.geoOverlay[i])
		} else {
			for j := range display[i] {
				display[i][j] = " "
			}
		}
	}

	// Draw precipitation data
	if frame.Data != nil {
		m.DrawPrecipitation(display, frame.Data, rw, rh)
	}

	// Add frame indicator dots
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
		Width(rw).
		Align(lipgloss.Center).
		Render(frameIndicator.String())

	if showExtras {
		radarStr += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("239")).
			Width(rw).
			Align(lipgloss.Center).
			Render("───── = 50 miles")

		radarStr += "\n" + lipgloss.NewStyle().
			Width(rw).
			Align(lipgloss.Center).
			Render(m.renderLegend())
	}

	return config.RadarContainerStyle.Render(radarStr)
}

func (m Model) renderForecastCrawl() string {
	if len(m.radar.Forecast) == 0 {
		return ""
	}

	idx := m.forecastIdx % len(m.radar.Forecast)
	p := m.radar.Forecast[idx]

	labelStyle := lipgloss.NewStyle().Foreground(config.PrimaryColor).Bold(true)
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("239"))

	tColor := lipgloss.Color("226")
	if p.Temperature >= 90 {
		tColor = lipgloss.Color("196")
	} else if p.Temperature >= 70 {
		tColor = lipgloss.Color("214")
	} else if p.Temperature >= 50 {
		tColor = lipgloss.Color("226")
	} else if p.Temperature >= 32 {
		tColor = lipgloss.Color("87")
	} else {
		tColor = lipgloss.Color("51")
	}

	temp := lipgloss.NewStyle().Foreground(tColor).Bold(true).Render(fmt.Sprintf("%d°%s", p.Temperature, p.TempUnit))

	// Position dots
	var dots strings.Builder
	for i := 0; i < len(m.radar.Forecast); i++ {
		if i == idx {
			dots.WriteString("●")
		} else {
			dots.WriteString("·")
		}
	}

	return labelStyle.Render("Forecast ") +
		dotStyle.Render(dots.String()+" ") +
		nameStyle.Render(p.Name) + " " +
		temp + " " +
		descStyle.Render(p.ShortFcast)
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
		localTime := frame.Timestamp.Local().Format("3:04 PM")
		utcTime := frame.Timestamp.UTC().Format("15:04 UTC")
		timeAgo := time.Since(frame.Timestamp).Round(time.Minute)
		frameInfo = fmt.Sprintf("Frame %d/%d  %s (%s)  %s ago",
			m.currentFrame+1, len(m.radar.Frames), localTime, utcTime, timeAgo)
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

	// Wind display
	windDisplay := ""
	if m.radar.Wind.Speed > 0 {
		windStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
		windDisplay = windStyle.Render(fmt.Sprintf("💨 %s %.0f mph", m.radar.Wind.Arrow, m.radar.Wind.Speed))
	}

	// Build the info panel
	infoItems := []string{location, station}

	if tempDisplay != "" {
		infoItems = append(infoItems, tempDisplay)
	}

	if conditionEmoji != "" {
		infoItems = append(infoItems, conditionEmoji)
	}

	if windDisplay != "" {
		infoItems = append(infoItems, windDisplay)
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

// Pre-computed precipitation styled strings (avoids 1800 style allocations per frame)
var precipRendered [11]string

func init() {
	chars := []string{" ", "·", "∘", "○", "●", "◉", "◆", "◈", "▰", "▱", "█"}
	// True color (24-bit) meteorological color ramp: green → yellow → orange → red → magenta
	fgColors := []lipgloss.Color{
		lipgloss.Color("#000000"), // 0: none
		lipgloss.Color("#00E5CC"), // 1: light drizzle (cyan-teal)
		lipgloss.Color("#00CC66"), // 2: light rain (green)
		lipgloss.Color("#33CC33"), // 3: light-moderate (bright green)
		lipgloss.Color("#66CC00"), // 4: moderate (yellow-green)
		lipgloss.Color("#CCCC00"), // 5: heavy (yellow)
		lipgloss.Color("#FF9900"), // 6: very heavy (orange)
		lipgloss.Color("#FF6600"), // 7: severe (dark orange)
		lipgloss.Color("#FF3300"), // 8: severe (red-orange)
		lipgloss.Color("#FF0033"), // 9: extreme (red)
		lipgloss.Color("#CC00FF"), // 10: extreme (magenta)
	}
	for i := range chars {
		precipRendered[i] = lipgloss.NewStyle().Foreground(fgColors[i]).Render(chars[i])
	}
}

func (m Model) DrawPrecipitation(display [][]string, data [][]int, rw, rh int) {
	for y := 0; y < len(data) && y < rh; y++ {
		for x := 0; x < len(data[y]) && x < rw; x++ {
			intensity := data[y][x]
			if intensity > 0 && intensity < len(precipRendered) {
				display[y][x] = precipRendered[intensity]
			}
		}
	}
}

func (m Model) renderLegend() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	return labelStyle.Render("Light ") +
		precipRendered[1] + precipRendered[2] + precipRendered[3] +
		labelStyle.Render(" Moderate ") +
		precipRendered[4] + precipRendered[5] + precipRendered[6] +
		labelStyle.Render(" Heavy ") +
		precipRendered[7] + precipRendered[8] +
		labelStyle.Render(" Severe ") +
		precipRendered[9] + precipRendered[10]
}

func (m Model) renderControls() string {
	controls := []string{
		"[Space] Play/Pause",
		"[←/→] Previous/Next",
		"[R] Refresh",
		"[+/-] Speed",
		"[Z/X] Zoom",
		"[ESC] New location",
		"[Q] Quit",
	}

	if m.showHelp {
		controls = append(controls, "",
			fmt.Sprintf("Frame rate: %s", m.frameRate),
			fmt.Sprintf("Auto-refresh: Every 5 minutes"),
			fmt.Sprintf("Zoom: %.1fx", m.zoomLevel),
		)
	}

	controlStr := config.HelpStyle.Render(strings.Join(controls, " • "))

	// Hover tooltip
	if m.hoverIntensity >= 0 {
		labels := []string{"Clear", "Light Drizzle", "Light Rain", "Light-Mod", "Moderate",
			"Heavy", "Very Heavy", "Severe", "Severe", "Extreme", "Extreme"}
		label := "Clear"
		if m.hoverIntensity < len(labels) {
			label = labels[m.hoverIntensity]
		}
		hoverStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF66")).Bold(true)
		controlStr += "  " + hoverStyle.Render(fmt.Sprintf("▸ %s (%d/10)", label, m.hoverIntensity))
	}

	// Forecast crawl line
	forecastCrawl := m.renderForecastCrawl()
	if forecastCrawl != "" {
		controlStr += "\n" + forecastCrawl
	}

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

// computeGeoOverlay builds the geographic overlay grid once for the current location
func (m *Model) computeGeoOverlay() {
	rw, rh := m.radarSize()
	overlay := make([][]string, rh)
	for i := range overlay {
		overlay[i] = make([]string, rw)
		for j := range overlay[i] {
			overlay[i][j] = " "
		}
	}
	centerX, centerY := rw/2, rh/2
	geography.DrawGeographicBoundaries(overlay, centerX, centerY, m.radar.Lat, m.radar.Lon, m.zoomLevel)
	geography.DrawDistanceMarkers(overlay, centerX, centerY, m.zoomLevel)
	m.geoOverlay = overlay
}

// radarSize returns the effective radar grid dimensions based on current terminal size
func (m Model) radarSize() (int, int) {
	return config.EffectiveRadarSize(m.width, m.height)
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
	m.zoomLevel = 1.0
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
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return ProgressMsg(m.progress.Percent() + 0.15)
	})
}