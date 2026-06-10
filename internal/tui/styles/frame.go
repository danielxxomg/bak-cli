package styles

// Frame wraps content in a DoubleBorder using FrameStyle, constrained to the
// given width. It returns the full bordered string including the box-drawing
// characters.
//
// If width is less than 4, the result may be malformed. Callers should guard
// against excessively narrow terminals before calling Frame.
func Frame(content string, width int) string {
	return FrameStyle.Width(width).Render(content)
}
