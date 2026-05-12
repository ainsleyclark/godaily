// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package og generates 1200×630 Open Graph images for GoDaily pages.
// Home returns a fully static pre-designed PNG.
// Issue composites dynamic text (kicker, headline, article list) onto a
// pre-designed template PNG, producing a unique card per digest.
package og

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/news"
	webpkg "github.com/ainsleyclark/godaily/web"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/pkg/errors"
)

// The issue template is a @2x export (≈2388×1256), so all coordinates and font
// sizes are doubled relative to the 1× OG spec (1200×630).
const (
	scale = 2 // @2x export factor

	padL = 64 * scale // left-column text padding

	// Vertical positions of each content band.
	kickerY    = 138 * scale
	headlineY  = 196 * scale
	articleY   = 340 * scale
	articleRow = 36 * scale // line pitch for article list rows

	// Max wrap width for the headline.
	wrapW = 556 * scale
)

// colour palette matching the design tokens.
var (
	colAccent     = color.NRGBA{R: 42, G: 168, B: 216, A: 255}  // #2aa8d8
	colAccentDark = color.NRGBA{R: 26, G: 127, B: 168, A: 255}  // #1a7fa8
	colText       = color.NRGBA{R: 13, G: 34, B: 54, A: 255}    // #0d2236
	colText3      = color.NRGBA{R: 122, G: 150, B: 170, A: 255} // #7a96aa
)

// Generator renders Open Graph images for GoDaily pages.
type Generator struct {
	sansExtraBold *truetype.Font
	sansRegular   *truetype.Font
	monoMedium    *truetype.Font
	issueTPL      []byte // raw bytes of the issue template PNG
}

// New loads the embedded fonts and issue template, ready to generate images.
func New() (*Generator, error) {
	g := &Generator{}

	var err error
	if g.sansExtraBold, err = loadFont("assets/fonts/DMSans-ExtraBold.ttf"); err != nil {
		return nil, errors.Wrap(err, "loading DMSans-ExtraBold")
	}
	if g.sansRegular, err = loadFont("assets/fonts/DMSans-Regular.ttf"); err != nil {
		return nil, errors.Wrap(err, "loading DMSans-Regular")
	}
	if g.monoMedium, err = loadFont("assets/fonts/DMMono-Medium.ttf"); err != nil {
		return nil, errors.Wrap(err, "loading DMMono-Medium")
	}

	g.issueTPL, err = webpkg.Assets.ReadFile("assets/images/og-issue.png")
	if err != nil {
		return nil, errors.Wrap(err, "loading issue template")
	}

	return g, nil
}

// Home returns the static homepage OG image bytes unchanged.
func (g *Generator) Home() ([]byte, error) {
	data, err := webpkg.Assets.ReadFile("assets/images/og-home.png")
	return data, errors.Wrap(err, "reading home OG image")
}

// Issue renders a 1200×630 (@2x) OG card for the given digest by compositing
// the kicker, headline, and top article titles onto the issue template PNG.
func (g *Generator) Issue(issue news.Issue) ([]byte, error) {
	base, err := png.Decode(bytes.NewReader(g.issueTPL))
	if err != nil {
		return nil, errors.Wrap(err, "decoding issue template")
	}

	dc := gg.NewContextForImage(base)

	// Kicker: WEEKDAY · DATE (no issue number).
	if !issue.SentAt.IsZero() {
		g.setFont(dc, g.monoMedium, 14*scale)
		dc.SetColor(colAccentDark)
		dc.DrawString(
			issue.SentAt.Format("Monday")+"  ·  "+issue.SentAt.Format("January 2, 2006"),
			padL, kickerY,
		)
	}

	// Headline — break at " – " so "GoDaily – May 11, 2026" spans two lines.
	g.setFont(dc, g.sansExtraBold, 50*scale)
	dc.SetColor(colText)
	subject := strings.Replace(truncate(issue.Subject, 75), " – ", "\n", 1)
	dc.DrawStringWrapped(subject, padL, headlineY, 0, 0, wrapW, 1.1, gg.AlignLeft)

	// Article list: up to 3 items, then a "+N more" line.
	y := float64(articleY)
	maxItems := min(3, len(issue.Items))
	for i := range maxItems {
		num := fmt.Sprintf("%02d", i+1)
		title := truncate(issue.Items[i].Title, 58)

		g.setFont(dc, g.monoMedium, 13*scale)
		dc.SetColor(colAccent)
		dc.DrawString(num, padL, y)

		g.setFont(dc, g.sansRegular, 17*scale)
		dc.SetColor(colText)
		dc.DrawString(title, padL+40*scale, y)
		y += float64(articleRow)
	}

	if remaining := len(issue.Items) - maxItems; remaining > 0 {
		g.setFont(dc, g.sansRegular, 17*scale)
		dc.SetColor(colText3)
		dc.DrawString(fmt.Sprintf("+%d more in this issue…", remaining), padL, y)
	}

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, errors.Wrap(err, "encoding PNG")
	}
	return buf.Bytes(), nil
}

// setFont applies the truetype font at the given point size.
// DPI is fixed at 72 so point size equals pixel size on the @2x canvas.
func (g *Generator) setFont(dc *gg.Context, f *truetype.Font, size float64) {
	dc.SetFontFace(truetype.NewFace(f, &truetype.Options{
		Size: size,
		DPI:  72,
	}))
}

// truncate shortens s to max runes, appending "…" if trimmed.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// loadFont reads a TTF from the web package's embedded FS and parses it.
func loadFont(path string) (*truetype.Font, error) {
	data, err := webpkg.Assets.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading font file")
	}
	f, err := truetype.Parse(data)
	if err != nil {
		return nil, errors.Wrap(err, "parsing font")
	}
	return f, nil
}
