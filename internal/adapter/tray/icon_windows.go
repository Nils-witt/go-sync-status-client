//go:build windows

package tray

import "encoding/binary"

// encodeIconBytes wraps pngBytes in a minimal single-image .ico container.
// getlantern/systray's Windows backend loads icons via the Win32 LoadImage
// API with LR_LOADFROMFILE, which parses the ICO file format only — handing
// it a bare PNG fails to load (silently, since the error only reaches
// systray's own internal logger) and the tray icon never appears. Since
// Windows Vista, an ICO directory entry may hold a PNG image verbatim, so no
// re-encoding is needed: just prepend the ICONDIR/ICONDIRENTRY header.
func encodeIconBytes(pngBytes []byte) []byte {
	const (
		icoHeaderSize = 6
		icoEntrySize  = 16
	)

	// width/height are encoded as a single byte each, where 0 means 256.
	dim := byte(iconSize)
	if iconSize >= 256 {
		dim = 0
	}

	buf := make([]byte, icoHeaderSize+icoEntrySize+len(pngBytes))

	// ICONDIR
	binary.LittleEndian.PutUint16(buf[0:2], 0) // reserved
	binary.LittleEndian.PutUint16(buf[2:4], 1) // type: 1 = icon
	binary.LittleEndian.PutUint16(buf[4:6], 1) // image count

	// ICONDIRENTRY
	entry := buf[icoHeaderSize:]
	entry[0] = dim                                                          // width
	entry[1] = dim                                                          // height
	entry[2] = 0                                                            // color count (0 = not palette-based)
	entry[3] = 0                                                            // reserved
	binary.LittleEndian.PutUint16(entry[4:6], 1)                            // color planes
	binary.LittleEndian.PutUint16(entry[6:8], 32)                           // bits per pixel
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngBytes)))       // image data size
	binary.LittleEndian.PutUint32(entry[12:16], icoHeaderSize+icoEntrySize) // offset to image data

	copy(buf[icoHeaderSize+icoEntrySize:], pngBytes)
	return buf
}
