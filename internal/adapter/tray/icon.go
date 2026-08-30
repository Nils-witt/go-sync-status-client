package tray

import (
	"bytes"
	"go-sync-status-client/internal/domain"
	"image"
	"image/color"
	"image/png"
	"sync"
)

// iconSize is the edge length, in pixels, of the generated tray icons.
const iconSize = 32

var (
	iconCacheMu sync.Mutex
	iconCache   = make(map[domain.SyncState][]byte)
)

// stateIcon returns PNG-encoded bytes for a colored floppy disk representing
// state, suitable for systray.SetIcon. Icons are rendered once per state and
// cached.
func stateIcon(state domain.SyncState) []byte {
	iconCacheMu.Lock()
	defer iconCacheMu.Unlock()

	if icon, ok := iconCache[state]; ok {
		return icon
	}

	icon := renderFloppyIcon(stateColor(state))
	iconCache[state] = icon
	return icon
}

func stateColor(state domain.SyncState) color.RGBA {
	switch state {
	case domain.SyncStateSynced:
		return color.RGBA{R: 0x2e, G: 0xcc, B: 0x71, A: 0xff} // green
	case domain.SyncStateSyncing:
		return color.RGBA{R: 0x34, G: 0x98, B: 0xdb, A: 0xff} // blue
	case domain.SyncStatePaused:
		return color.RGBA{R: 0xf3, G: 0x9c, B: 0x12, A: 0xff} // orange
	case domain.SyncStateError:
		return color.RGBA{R: 0xe7, G: 0x4c, B: 0x3c, A: 0xff} // red
	default:
		return color.RGBA{R: 0x95, G: 0xa5, B: 0xa6, A: 0xff} // gray
	}
}

// floppyGeometry describes the pixel layout of a classic 3.5" floppy disk
// glyph: a square body with its top-right corner folded off, a metal
// shutter near the top, and a label in the lower half.
type floppyGeometry struct {
	left, top, right, bottom int
	cornerRadius, foldSize   int
	shutterMinX, shutterMaxX int
	shutterMinY, shutterMaxY int
	labelMinX, labelMaxX     int
	labelMinY, labelMaxY     int
}

func newFloppyGeometry() floppyGeometry {
	const margin = 2
	return floppyGeometry{
		left: margin, top: margin,
		right: iconSize - 1 - margin, bottom: iconSize - 1 - margin,
		cornerRadius: 3, foldSize: 7,
		shutterMinX: margin + 4, shutterMaxX: iconSize - 1 - margin - 4,
		shutterMinY: margin + 2, shutterMaxY: margin + 8,
		labelMinX: margin + 3, labelMaxX: iconSize - 1 - margin - 3,
		labelMinY: margin + 12, labelMaxY: iconSize - 1 - margin - 3,
	}
}

var (
	shutterColor = color.RGBA{R: 0xec, G: 0xf0, B: 0xf1, A: 0xff} // light metal
	labelColor   = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// renderFloppyIcon draws a floppy disk glyph, with the body filled in c, and
// returns it PNG-encoded.
func renderFloppyIcon(c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	geo := newFloppyGeometry()

	for y := range iconSize {
		for x := range iconSize {
			img.Set(x, y, floppyPixel(geo, c, x, y))
		}
	}

	var buf bytes.Buffer
	// Encoding an in-memory image.RGBA into a bytes.Buffer cannot fail.
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func floppyPixel(geo floppyGeometry, c color.RGBA, x, y int) color.RGBA {
	if !geo.inBody(x, y) {
		return color.RGBA{}
	}
	if x >= geo.shutterMinX && x <= geo.shutterMaxX && y >= geo.shutterMinY && y <= geo.shutterMaxY {
		return shutterColor
	}
	if x >= geo.labelMinX && x <= geo.labelMaxX && y >= geo.labelMinY && y <= geo.labelMaxY {
		return labelColor
	}
	return c
}

// inBody reports whether (x, y) falls inside the disk body: a square with
// rounded corners, except the top-right corner which is cut off diagonally
// to form the classic folded-corner floppy silhouette.
func (g floppyGeometry) inBody(x, y int) bool {
	if x < g.left || x > g.right || y < g.top || y > g.bottom {
		return false
	}

	cutX, cutY := g.right-g.foldSize, g.top+g.foldSize
	if x > cutX && y < cutY && (x-cutX)+(cutY-y) > g.foldSize {
		return false
	}

	return !g.outsideRoundedCorner(x, y, g.left, g.top, 1, 1) &&
		!g.outsideRoundedCorner(x, y, g.left, g.bottom, 1, -1) &&
		!g.outsideRoundedCorner(x, y, g.right, g.bottom, -1, -1)
}

// outsideRoundedCorner reports whether (x, y) lies in the square quadrant
// extending from (cornerX, cornerY) toward (dx, dy) but outside the
// cornerRadius circle inscribed there.
func (g floppyGeometry) outsideRoundedCorner(x, y, cornerX, cornerY, dx, dy int) bool {
	centerX, centerY := cornerX+dx*g.cornerRadius, cornerY+dy*g.cornerRadius
	inQuadrant := (dx > 0 && x < centerX || dx < 0 && x > centerX) &&
		(dy > 0 && y < centerY || dy < 0 && y > centerY)
	if !inQuadrant {
		return false
	}
	ddx, ddy := x-centerX, y-centerY
	return ddx*ddx+ddy*ddy > g.cornerRadius*g.cornerRadius
}
