package pages

import "testing"

func TestSafeFTS5Query(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"apostrophe", "We're going", `"We're"* "going"*`},
		{"contraction", "don't", `"don't"*`},
		{"embedded double quote", `it's "quoted"`, `"it's"* """quoted"""*`},
		{"dash operator", "a - b", `"a"* "-"* "b"*`},
		{"column filter", "col1:col2", `"col1:col2"*`},
		{"prefix star", "*", `"*"*`},
		{"unclosed paren", "(unclosed", `"(unclosed"*`},
		{"AND keyword", "a AND b", `"a"* "AND"* "b"*`},
		{"NEAR operator", "NEAR(a b)", `"NEAR(a"* "b)"*`},
		{"extra whitespace", "  hello    world  ", `"hello"* "world"*`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeFTS5Query(tt.in); got != tt.want {
				t.Fatalf("SafeFTS5Query(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
