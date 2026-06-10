package tui

// Shared keybinding constants for the bak-cli TUI. These are used across
// screens to maintain consistent keyboard shortcuts.
const (
	// KeyQuit exits the application (q).
	KeyQuit rune = 'q'
	// KeyDown navigates down in lists and menus (j).
	KeyDown rune = 'j'
	// KeyUp navigates up in lists and menus (k).
	KeyUp rune = 'k'
	// KeyEnter selects the current item or advances the wizard (carriage return).
	KeyEnter rune = '\r'
	// KeyEsc navigates back or exits (escape, ASCII 27).
	KeyEsc rune = 27
)
