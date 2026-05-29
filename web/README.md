# Web

The public site (`godaily.dev`): templ views, SCSS, and a small amount of
TypeScript, compiled by esbuild and served by the handlers in `web/handlers`.

## Layout

```
web/
  assets/
    scss/        Styles (compiled to dist/css/app.css)
    js/          TypeScript (compiled to dist/js/app.js)
  views/
    components/  Reusable templ components
    graphics/    Inline SVG graphics (logos, icons)
    layouts/     Page shells (<head>, footer, …)
    pages/       Full pages
    email/       Email templates
  handlers/      HTTP handlers for each route
  generate/      Static-site generator (renders pages to HTML)
```

## Conventions

### SCSS — one component per file

Every UI component gets its own partial under `assets/scss/components/`
(`_search-box.scss`, `_stat-strip.scss`, `_tabs.scss`, …). Do **not** pile
multiple unrelated components into one file. Register each partial in
`assets/scss/app.scss` with `@use`.

The file name should match the component's block class
(`.search-box` → `_search-box.scss`).

### Colours — CSS variables only

Never hard-code colours in SCSS. No hex (`#fff`), no bare keywords (`white`).
Reference a CSS custom property defined in
`assets/scss/abstracts/_variables.scss`:

```scss
// Don't
background: #fff;
color: white;
box-shadow: 0 2px 5px rgba(42, 168, 216, 0.35);

// Do
background: var(--color-white);
color: var(--color-white);
box-shadow: var(--shadow-accent-mark);
```

If you need a colour or shadow that doesn't exist yet, add it to `:root` in
`_variables.scss` first, then reference it. Recurring `rgba()` washes,
overlays, and shadows live there too (`--accent-wash-strong`,
`--overlay-white`, `--shadow-accent-*`).

### SVGs — in `graphics`, not inline

Don't inline `<svg>` markup inside components. Put glyphs in the graphics
packages and reference them:

- `views/graphics/icons` — generic UI glyphs (`icons.Search(16)`,
  `icons.Envelope(11)`). Each icon is a templ component taking a pixel size.
- `views/graphics/logos` — brand marks for news sources.

To add an icon, add a `templ` function to `views/graphics/icons/icons.templ`
and call it from the component (`@icons.Search(16)`).

## Build & generate

```
make templ        # regenerate *_templ.go from *.templ
pnpm -C web build  # compile SCSS + TS + images into web/dist
```

See the repo-root `CLAUDE.md` for Go testing, linting, and codegen rules.
