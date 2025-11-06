package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Define styles using Lip Gloss
var (
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)

// Helper functions for styled output
func Success(msg string) string {
	return SuccessStyle.Render("✓ " + msg)
}

func Error(msg string) string {
	return ErrorStyle.Render("❌ " + msg)
}

func Warning(msg string) string {
	return WarningStyle.Render("⚠️  " + msg)
}

func Info(msg string) string {
	return InfoStyle.Render(msg)
}
