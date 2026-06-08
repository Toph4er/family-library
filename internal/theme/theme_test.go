package theme

import (
	"strings"
	"testing"
)

func TestAvailableThemes(t *testing.T) {
	themes := AvailableThemes()

	if len(themes) != 4 {
		t.Fatalf("expected 4 themes, got %d", len(themes))
	}

	for i, th := range themes {
		if th.ID == "" {
			t.Errorf("theme[%d] has empty ID", i)
		}
		if th.Name == "" {
			t.Errorf("theme[%d] (%s) has empty Name", i, th.ID)
		}
		if th.Primary == "" {
			t.Errorf("theme[%d] (%s) has empty Primary color", i, th.ID)
		}
	}
}

func TestGetThemeByID_valid(t *testing.T) {
	tests := []struct {
		id         string
		wantName   string
		wantPrimary string
	}{
		{"woodland", "Woodland Fairytale", "#2d5016"},
		{"space", "Space Explorer", "#5b5bd6"},
		{"dinosaurs", "Dino Discovery", "#2d5a27"},
		{"princesses", "Royal Stories", "#6a2c70"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			th := GetThemeByID(tt.id)
			if th.ID != tt.id {
				t.Errorf("GetThemeByID(%q) = ID %q, want %q", tt.id, th.ID, tt.id)
			}
			if th.Name != tt.wantName {
				t.Errorf("GetThemeByID(%q) = Name %q, want %q", tt.id, th.Name, tt.wantName)
			}
			if th.Primary != tt.wantPrimary {
				t.Errorf("GetThemeByID(%q) = Primary %q, want %q", tt.id, th.Primary, tt.wantPrimary)
			}
		})
	}
}

func TestGetThemeByID_invalid(t *testing.T) {
	// Unknown ID falls back to WoodlandFairytale
	th := GetThemeByID("nonexistent")
	if th.ID != "woodland" {
		t.Errorf("GetThemeByID(\"nonexistent\") = ID %q, want \"woodland\"", th.ID)
	}
	if th.Name != "Woodland Fairytale" {
		t.Errorf("GetThemeByID(\"nonexistent\") = Name %q, want \"Woodland Fairytale\"", th.Name)
	}

	// Empty string also falls back to WoodlandFairytale
	th = GetThemeByID("")
	if th.ID != "woodland" {
		t.Errorf("GetThemeByID(\"\") = ID %q, want \"woodland\"", th.ID)
	}
}

func TestCSSOverrideBlock_notEmpty(t *testing.T) {
	themes := AvailableThemes()
	for _, th := range themes {
		css := th.CSSOverrideBlock()
		if len(css) == 0 {
			t.Errorf("CSSOverrideBlock() for theme %q returned empty HTML", th.ID)
		}
		// Verify the CSS contains the theme's primary color hex value
		if !strings.Contains(string(css), th.Primary) {
			t.Errorf("CSSOverrideBlock() for theme %q does not contain primary color %q", th.ID, th.Primary)
		}
	}
}

func TestCSSOverrideBlock_noStyleTags(t *testing.T) {
	themes := AvailableThemes()
	for _, th := range themes {
		css := string(th.CSSOverrideBlock())
		// The raw template.HTML string includes <style> tags — verify content is present
		if !strings.Contains(css, ":root") {
			t.Errorf("CSSOverrideBlock() for theme %q does not contain :root", th.ID)
		}
		if !strings.Contains(css, "--color-primary") {
			t.Errorf("CSSOverrideBlock() for theme %q does not contain --color-primary", th.ID)
		}
	}
}
