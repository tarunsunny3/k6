package progressbar

import "github.com/fatih/color"

//nolint:gochecknoglobals
var (
	colorFaint   = color.New(color.Faint)
	statusColors = map[Status]*color.Color{
		Interrupted: color.New(color.FgRed),
		Done:        color.New(color.FgGreen),
		Waiting:     colorFaint,
	}
)

const (
	// DefaultWidth of the progress bar
	defaultWidth = 40
	// threshold below which progress should be rendered as
	// percentages instead of filling bars
	minWidth = 8
	// Max length of left-side progress bar text before trimming is forced
	maxLeftLength = 30
	// Amount of padding in chars between rendered progress
	// bar text and right-side terminal window edge.
	termPadding = 1
)

// Status of the progress bar
type Status rune

// Progress bar status symbols
const (
	Running     Status = ' '
	Waiting     Status = '•'
	Stopping    Status = '↓'
	Interrupted Status = '✗'
	Done        Status = '✓'
)
