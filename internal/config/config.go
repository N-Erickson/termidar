package config

import (
	"github.com/charmbracelet/lipgloss"
)

// Constants
const (
	RadarWidth  = 60
	RadarHeight = 30
	MaxFrames   = 20
)

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
			Foreground(PrimaryColor).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
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
			BorderForeground(lipgloss.Color("239")).
			Padding(0, 1).
			MarginTop(1)

	LocationStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(SuccessColor)

	StationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Radar styles
	RadarContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
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