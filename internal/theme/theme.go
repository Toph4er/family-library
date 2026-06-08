package theme

import (
	"fmt"
	"html/template"
)

// Theme represents a complete visual theme for the library application.
type Theme struct {
	ID              string
	Name            string
	Description     string
	Primary         string // --color-primary
	Secondary       string // --color-secondary
	Accent          string // --color-accent
	Background      string // --color-background
	Surface         string // --color-surface
	Text            string // --color-text
	TextLight       string // --color-text-light
	Success         string // --color-success
	Warning         string // --color-warning
	Error           string // --color-error
	FontHeading     string // --font-heading (Google Fonts name + fallback)
	FontBody        string // --font-body
	FontAccent      string // --font-accent
	BackgroundSVG   string // SVG data URI for body::before pattern
	SelectionBG     string // ::selection background
	ScrollbarThumb  string // scrollbar thumb color
	VineDividerGrad string // gradient for vine-divider pseudo-elements
	FooterTagline   string // footer decorative text
	FooterSVG       string // small inline SVG for footer
}

// WoodlandFairytale returns the default woodland-themed theme.
func WoodlandFairytale() Theme {
	return Theme{
		ID:              "woodland",
		Name:            "Woodland Fairytale",
		Description:     "A magical forest of stories",
		Primary:         "#2d5016",
		Secondary:       "#8b4513",
		Accent:          "#d4a574",
		Background:      "#f5f0e8",
		Surface:         "#faf8f5",
		Text:            "#1a2f0a",
		TextLight:       "#4a5d23",
		Success:         "#457528",
		Warning:         "#a84a2e",
		Error:           "#8b2500",
		FontHeading:     "'Playfair Display', serif",
		FontBody:        "'Inter', sans-serif",
		FontAccent:      "'Caveat', cursive",
		BackgroundSVG:   "data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M30 5 L35 15 L30 12 L25 15 Z' fill='rgba(45,80,22,0.03)'/%3E%3Cpath d='M30 25 L33 30 L30 28 L27 30 Z' fill='rgba(45,80,22,0.03)'/%3E%3Cpath d='M30 45 L34 52 L30 50 L26 52 Z' fill='rgba(45,80,22,0.03)'/%3E%3C/svg%3E",
		SelectionBG:     "rgba(45, 80, 22, 0.2)",
		ScrollbarThumb:  "rgba(139, 69, 19, 0.3)",
		VineDividerGrad: "linear-gradient(to right, transparent, rgba(139, 69, 19, 0.3), transparent)",
		FooterTagline:   "Where stories grow like ancient trees",
		FooterSVG:       "<svg width='16' height='16' viewBox='0 0 24 24' fill='currentColor'><path d='M17 8C8 10 5.9 16.17 3.82 21.34l1.89.66.95-2.3c.48.17.98.3 1.34.3C19 20 22 3 22 3c-1 2-8 2.25-13 3.25S2 11.5 2 13.5s1.75 3.75 1.75 3.75C7 8 17 8 17 8z'/></svg>",
	}
}

// Space returns the space-themed theme.
func Space() Theme {
	return Theme{
		ID:              "space",
		Name:            "Space Explorer",
		Description:     "Blast off into a galaxy of stories",
		Primary:         "#5b5bd6",
		Secondary:       "#7b68ee",
		Accent:          "#ffd700",
		Background:      "#0a0e27",
		Surface:         "#151b3d",
		Text:            "#ffffff",
		TextLight:       "#c8d6f0",
		Success:         "#2dd4a8",
		Warning:         "#f0a030",
		Error:           "#e04040",
		FontHeading:     "'Orbitron', sans-serif",
		FontBody:        "'Inter', sans-serif",
		FontAccent:      "'Space Mono', monospace",
		BackgroundSVG:   "data:image/svg+xml,%3Csvg%20width%3D%2780%27%20height%3D%2780%27%20viewBox%3D%270%200%2080%2080%27%20xmlns%3D%27http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%27%3E\n  %3Ccircle%20cx%3D%2711%27%20cy%3D%2711%27%20r%3D%270.35%27%20fill%3D%27rgba%28255%2C255%2C255%2C0.08%29%27/%3E\n  %3Ccircle%20cx%3D%2729%27%20cy%3D%277%27%20r%3D%270.3%27%20fill%3D%27rgba%28255%2C255%2C255%2C0.06%29%27/%3E\n  %3Ccircle%20cx%3D%2760%27%20cy%3D%2716%27%20r%3D%270.4%27%20fill%3D%27rgba%28240%2C192%2C64%2C0.1%29%27/%3E\n  %3Ccircle%20cx%3D%2773%27%20cy%3D%2727%27%20r%3D%270.2%27%20fill%3D%27rgba%28255%2C255%2C255%2C0.07%29%27/%3E\n  %3Ccircle%20cx%3D%2720%27%20cy%3D%2729%27%20r%3D%270.35%27%20fill%3D%27rgba%28255%2C255%2C255%2C0.05%29%27/%3E\n  %3Ccircle%20cx%3D%2747%27%20cy%3D%2740%27%20r%3D%270.5%27%20fill%3D%27rgba%28240%2C192%2C64%2C0.1%29%27/%3E\n  %3Ccircle%20cx%3D%277%27%20cy%3D%2747%27%20r%3D%270.3%27%20fill%3D%27rgba%28255%2C255%2C255%2C0.06%29%27/%3E\n  %3Ccircle%20cx%3D%2737%27%20cy%3D%2756%27%20r%3D%270.35%27%20fill%3D%27rgba%28255%2C255%2C255%2C0.07%29%27/%3E\n  %3Ccircle%20cx%3D%2764%27%20cy%3D%2764%27%20r%3D%270.4%27%20fill%3D%27rgba%28240%2C192%2C64%2C0.08%29%27/%3E\n  %3Ccircle%20cx%3D%2769%27%20cy%3D%2773%27%20r%3D%270.25%27%20fill%3D%27rgba%28255%2C255%2C255%2C0.06%29%27/%3E\n  %3Ccircle%20cx%3D%2716%27%20cy%3D%2769%27%20r%3D%270.3%27%20fill%3D%27rgba%28255%2C255%2C255%2C0.05%29%27/%3E\n  %3Ccircle%20cx%3D%2751%27%20cy%3D%2773%27%20r%3D%270.2%27%20fill%3D%27rgba%28240%2C192%2C64%2C0.07%29%27/%3E\n%3C/svg%3E",
		SelectionBG:     "rgba(240, 192, 64, 0.25)",
		ScrollbarThumb:  "rgba(74, 63, 138, 0.5)",
		VineDividerGrad: "linear-gradient(to right, transparent, rgba(240, 192, 64, 0.3), transparent)",
		FooterTagline:   "Every book is a new world to explore",
		FooterSVG:       "<svg width='16' height='16' viewBox='0 0 24 24' fill='currentColor'><path d='M12 2l2.4 7.2L22 12l-7.6 2.8L12 22l-2.4-7.2L2 12l7.6-2.8L12 2z'/></svg>",
	}
}

// Dinosaurs returns the dinosaur-themed theme.
func Dinosaurs() Theme {
	return Theme{
		ID:              "dinosaurs",
		Name:            "Dino Discovery",
		Description:     "Roar into a prehistoric library adventure",
		Primary:         "#2d5a27",
		Secondary:       "#8b6914",
		Accent:          "#d4943a",
		Background:      "#f0ead6",
		Surface:         "#f7f3e8",
		Text:            "#1a2e15",
		TextLight:       "#4a6b3a",
		Success:         "#3a8a2a",
		Warning:         "#c47a20",
		Error:           "#a02020",
		FontHeading:     "'Bangers', cursive",
		FontBody:        "'Inter', sans-serif",
		FontAccent:      "'Permanent Marker', cursive",
		BackgroundSVG:   "data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M30 5 L33 8 L31 10 L33 12 L30 11 L27 12 L29 10 L27 8 Z' fill='rgba(45,90,39,0.04)'/%3E%3Cpath d='M30 30 L33 33 L31 35 L33 37 L30 36 L27 37 L29 35 L27 33 Z' fill='rgba(45,90,39,0.04)'/%3E%3C/svg%3E",
		SelectionBG:     "rgba(45, 90, 39, 0.25)",
		ScrollbarThumb:  "rgba(139, 105, 20, 0.4)",
		VineDividerGrad: "linear-gradient(to right, transparent, rgba(139, 105, 20, 0.3), transparent)",
		FooterTagline:   "Stories that have survived millions of years",
		FooterSVG:       "<svg width='16' height='16' viewBox='0 0 24 24' fill='currentColor'><path d='M12 2c-2 0-4 2-4 5 0 1.5.5 2.5 1.5 3.5L8 16c-1 1.5 0 3 1.5 3h5c1.5 0 2.5-1.5 1.5-3l-1.5-5.5c1-1 1.5-2 1.5-3.5 0-3-2-5-4-5h0z'/></svg>",
	}
}

// Princesses returns the royal princess-themed theme.
func Princesses() Theme {
	return Theme{
		ID:              "princesses",
		Name:            "Royal Stories",
		Description:     "A castle of tales fit for royalty",
		Primary:         "#6a2c70",
		Secondary:       "#b8860b",
		Accent:          "#e8b4d8",
		Background:      "#fdf5f9",
		Surface:         "#fff8fc",
		Text:            "#2d1530",
		TextLight:       "#6a3a6a",
		Success:         "#4a8a5a",
		Warning:         "#c49a20",
		Error:           "#a02040",
		FontHeading:     "'Cinzel', serif",
		FontBody:        "'Inter', sans-serif",
		FontAccent:      "'Dancing Script', cursive",
		BackgroundSVG:   "data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M30 5 L31 9 L35 10 L31 11 L30 15 L29 11 L25 10 L29 9 Z' fill='rgba(184,134,11,0.08)'/%3E%3Cpath d='M30 35 L31 39 L35 40 L31 41 L30 45 L29 41 L25 40 L29 39 Z' fill='rgba(184,134,11,0.08)'/%3E%3C/svg%3E",
		SelectionBG:     "rgba(106, 44, 112, 0.2)",
		ScrollbarThumb:  "rgba(184, 134, 11, 0.3)",
		VineDividerGrad: "linear-gradient(to right, transparent, rgba(184, 134, 11, 0.3), transparent)",
		FooterTagline:   "Once upon a time, in a library far far away",
		FooterSVG:       "<svg width='16' height='16' viewBox='0 0 24 24' fill='currentColor'><path d='M2 20h20v2H2v-2zm2-3l2-8 3 3 3-7 3 7 3-3 2 8H4z'/></svg>",
	}
}

// AvailableThemes returns all registered themes.
func AvailableThemes() []Theme {
	return []Theme{
		WoodlandFairytale(),
		Space(),
		Dinosaurs(),
		Princesses(),
	}
}

// GetThemeByID returns a theme by its ID, or WoodlandFairytale if not found.
func GetThemeByID(id string) Theme {
	for _, t := range AvailableThemes() {
		if t.ID == id {
			return t
		}
	}
	return WoodlandFairytale()
}

// CSSOverrideBlock returns a <style> block that overrides Tailwind's @theme
// variables and custom component styles for this theme. Used in base.html.
func (t Theme) CSSOverrideBlock() template.HTML {
	// #nosec G203 -- interpolated values are application-controlled theme settings, not user input
	return template.HTML(fmt.Sprintf(`
<style>
:root {
  --color-primary: %s !important;
  --color-secondary: %s !important;
  --color-accent: %s !important;
  --color-background: %s !important;
  --color-surface: %s !important;
  --color-text: %s !important;
  --color-text-light: %s !important;
  --color-success: %s !important;
  --color-warning: %s !important;
  --color-error: %s !important;
  --font-heading: %s !important;
  --font-body: %s !important;
  --font-accent: %s !important;
}
.vine-divider::before, .vine-divider::after {
  background: %s;
}
.mushroom-tag { background: color-mix(in srgb, %s 10%%, transparent); color: %s; }
.tag-pill { background: color-mix(in srgb, %s 10%%, transparent); color: %s; }
.tag-input-container { background: %s; border-color: color-mix(in srgb, %s 20%%, transparent); }
.modal-content { background: %s; }
.toast-success { background: %s; }
.toast-error { background: %s; }
.toast-info { background: %s; }
.skip-link { background: %s; }
body::-webkit-scrollbar-thumb { background: %s; }
body { scrollbar-color: %s transparent; }
body::before { content: ''; background-color: %s !important; background-image: url("%s") !important; z-index: -1; }
::selection { background: %s; color: %s; }

/* Dark theme support */
html, body { background-color: %s !important; color: %s !important; }
input, select, textarea {
	background-color: %s !important;
	color: %s !important;
	border-color: %s !important;
}
.bg-surface { background-color: %s !important; }
</style>
`, t.Primary, t.Secondary, t.Accent, t.Background, t.Surface, t.Text, t.TextLight,
		t.Success, t.Warning, t.Error,
		t.FontHeading, t.FontBody, t.FontAccent,
		t.VineDividerGrad,
		t.Primary, t.Primary,
		t.Primary, t.Primary,
		t.Surface, t.Secondary,
		t.Surface,
		t.Success, t.Error, t.Primary,
		t.Primary,
		t.ScrollbarThumb, t.ScrollbarThumb,
		t.Background, t.BackgroundSVG,
		t.SelectionBG, t.Text,
		t.Background, t.Text,
		t.Surface, t.Text, t.Secondary,
		t.Surface))
}
