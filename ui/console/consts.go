package console

const (
	banner = "" +
		`          /\      |‾‾| /‾‾/   /‾‾/   \n` +
		`     /\  /  \     |  |/  /   /  /    \n` +
		`    /  \/    \    |     (   /   ‾‾\  \n` +
		`   /          \   |  |\  \ |  (‾)  | \n` +
		`  / __________ \  |__| \__\ \_____/ .io`

	// Max length of left-side progress bar text before trimming is forced
	maxLeftLength = 30
	// Amount of padding in chars between rendered progress
	// bar text and right-side terminal window edge.
	termPadding      = 1
	defaultTermWidth = 80
)
