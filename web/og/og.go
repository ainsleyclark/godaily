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
// Issue composites dynamic text (issue number, date, article list) onto a
// pre-designed template PNG, producing a unique card per digest.
package og

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"

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

	articleRow = 36 * scale // line pitch for article list rows

	// Empirical @2x pixel bounds of the template's text area (below wordmark, above footer).
	zoneTopPx    = 180.0
	zoneBottomPx = 1055.0
)

// colour palette matching the design tokens.
var (
	colAccent = color.NRGBA{R: 42, G: 168, B: 216, A: 255}  // #2aa8d8
	colText   = color.NRGBA{R: 13, G: 34, B: 54, A: 255}    // #0d2236
	colText3  = color.NRGBA{R: 122, G: 150, B: 170, A: 255} // #7a96aa
)

// Generator renders Open Graph images for GoDaily pages.
type Generator struct {
	sansBlack   *truetype.Font
	sansRegular *truetype.Font
	monoMedium  *truetype.Font
	issueTPL    []byte // raw bytes of the issue template PNG
}

// New loads the embedded fonts and issue template, ready to generate images.
func New() (*Generator, error) {
	g := &Generator{}

	var err error
	if g.sansBlack, err = loadFont("assets/fonts/DMSans_36pt-Black.ttf"); err != nil {
		return nil, errors.Wrap(err, "loading DMSans_36pt-Black")
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
// the headline and top article titles onto the issue template PNG. The content
// block is vertically centred in the template's text area.
func (g *Generator) Issue(issue news.Issue) ([]byte, error) {
	base, err := png.Decode(bytes.NewReader(g.issueTPL))
	if err != nil {
		return nil, errors.Wrap(err, "decoding issue template")
	}

	dc := gg.NewContextForImage(base)

	// Headline: "#Issue N" on line 1, ordinal date on line 2 (when available).
	line1 := fmt.Sprintf("- Issue %d", issue.ID)
	var line2 string
	if !issue.SentAt.IsZero() {
		d := issue.SentAt.Day()
		line2 = fmt.Sprintf("%d%s %s", d, ordinalSuffix(d), issue.SentAt.Format("January 2006"))
	}

	// Measure one line height for layout maths.
	g.setFont(dc, g.sansBlack, 50*scale)
	_, lineH := dc.MeasureString(line1)

	const linePitch = 1.15 // inter-line spacing multiplier

	// Headline block height covers either 1 or 2 lines.
	headlineBlockH := lineH
	if line2 != "" {
		headlineBlockH = lineH * (1 + linePitch)
	}

	// Article list dimensions.
	const listGap = 42 * scale
	itemCount := min(3, len(issue.Items))
	hasMore := len(issue.Items) > itemCount
	totalRows := itemCount
	if hasMore {
		totalRows++
	}
	listH := float64(totalRows) * float64(articleRow)

	// Total block height (visual top of headline → visual bottom of last row).
	blockH := headlineBlockH + listGap + listH

	// Vertically centre block in the template's text area.
	// Visual block top ≈ headlineBaseline − lineH×0.75 (ascent of line 1).
	// Centre = visual top + blockH/2  →  headlineBaseline = centre + 0.75·lineH − blockH/2
	const centerY = (zoneTopPx + zoneBottomPx) / 2.0
	headlineBaseline := centerY + lineH*0.75 - blockH/2.0

	// Draw headline lines.
	dc.SetColor(colText)
	dc.DrawString(line1, padL, headlineBaseline)
	if line2 != "" {
		dc.DrawString(line2, padL, headlineBaseline+lineH*linePitch)
	}

	// Article list starts below the last headline line's descender.
	lastHeadlineBaseline := headlineBaseline
	if line2 != "" {
		lastHeadlineBaseline = headlineBaseline + lineH*linePitch
	}
	y := lastHeadlineBaseline + lineH*0.25 + listGap

	for i := range itemCount {
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

	if remaining := len(issue.Items) - itemCount; remaining > 0 {
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

// ordinalSuffix returns "st", "nd", "rd", or "th" for the given day number.
func ordinalSuffix(n int) string {
	switch {
	case n%100 >= 11 && n%100 <= 13:
		return "th"
	case n%10 == 1:
		return "st"
	case n%10 == 2:
		return "nd"
	case n%10 == 3:
		return "rd"
	default:
		return "th"
	}
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
