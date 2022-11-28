package progressbar

// These util functions were copied from lib/util.go to avoid an import cycle
// (lib -> progressbar -> lib).

// Returns the maximum value of a and b.
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Returns the minimum value of a and b.
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
