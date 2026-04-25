package config

import (
	"github.com/charmbracelet/lipgloss"
)

// Constants
const (
	RadarWidth     = 40 // minimum radar width
	RadarHeight    = 15 // minimum radar height
	MaxFrames      = 20
	RadarChromeW   = 10 // AppStyle padding(4) + container border(2) + container padding(2) + margin(2)
	RadarChromeH   = 24 // header + info panel + container chrome + dots/scale/legend + controls + forecast crawl
)

// EffectiveRadarSize computes the radar grid dimensions from terminal size.
// Returns at least RadarWidth x RadarHeight.
func EffectiveRadarSize(termW, termH int) (int, int) {
	w := termW - RadarChromeW
	h := termH - RadarChromeH
	if w < RadarWidth {
		w = RadarWidth
	}
	if h < RadarHeight {
		h = RadarHeight
	}
	return w, h
}

// Styles
var (
	// Color palette
	PrimaryColor   = lipgloss.Color("86")
	SecondaryColor = lipgloss.Color("205")
	AccentColor    = lipgloss.Color("213")
	ErrorColor     = lipgloss.Color("196")
	SuccessColor   = lipgloss.Color("46")
	RadarGreen     = lipgloss.Color("40")
	RadarYellow    = lipgloss.Color("226")
	RadarOrange    = lipgloss.Color("208")
	RadarRed       = lipgloss.Color("196")

	// Layout styles
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(RadarGreen).
			Background(lipgloss.Color("233")).
			Padding(0, 2).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	// Input styles
	InputContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(SecondaryColor).
				Padding(1, 2)

	ActiveInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(AccentColor).
				Padding(1, 2)

	// Info panel styles
	InfoPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginTop(1)

	LocationStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(SuccessColor)

	StationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Custom radar border for instrument look
	RadarBorder = lipgloss.Border{
		Top:         "═",
		Bottom:      "═",
		Left:        "║",
		Right:       "║",
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
	}

	// Radar styles
	RadarContainerStyle = lipgloss.NewStyle().
				Border(RadarBorder).
				BorderForeground(RadarGreen).
				Padding(1).
				MarginTop(1)

	RadarFrameStyle = lipgloss.NewStyle().
			Width(RadarWidth).
			Height(RadarHeight)

	// Status styles
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// Progress bar style
	ProgressStyle = lipgloss.NewStyle().
			MarginTop(1)
)