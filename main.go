package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/N-Erickson/termidar/internal/ui"
)


func main() {
	p := tea.NewProgram(ui.InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}