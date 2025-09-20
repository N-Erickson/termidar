package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/N-Erickson/termidar/internal/ui"
)


func main() {
	model := ui.InitialModel()
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}

	// Clean up resources
	if err := model.Close(); err != nil {
		fmt.Printf("Cleanup error: %v", err)
	}
}