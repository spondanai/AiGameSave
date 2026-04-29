package clipboard

import (
	"fmt"

	atottoclip "github.com/atotto/clipboard"
)

// WriteToClipboard writes the given string to the system clipboard.
// Supported OS: Linux, Windows, macOS (via atotto/clipboard).
func WriteToClipboard(text string) error {
	if atottoclip.Unsupported {
		return fmt.Errorf("clipboard not supported on this OS")
	}
	return atottoclip.WriteAll(text)
}
