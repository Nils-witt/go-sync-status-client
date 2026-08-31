//go:build !windows

package tray

// encodeIconBytes returns pngBytes unchanged. macOS and Linux (via
// getlantern/systray's NSImage/gdk-pixbuf backends) load tray icons directly
// from PNG data.
func encodeIconBytes(pngBytes []byte) []byte {
	return pngBytes
}
