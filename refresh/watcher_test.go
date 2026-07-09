package refresh

import "testing"

func TestWatcher_isWatchedFile(t *testing.T) {
	tests := []struct {
		name               string
		includedExtensions []string
		includedPatterns   []string
		path               string
		want               bool
	}{
		// Extensions: backwards compatible exact match on the last extension.
		{
			name:               "extension matches simple .go file",
			includedExtensions: []string{".go"},
			path:               "cmd/main.go",
			want:               true,
		},
		{
			name:               "extension matches multi-dot file by last extension",
			includedExtensions: []string{".go"},
			path:               "api/service.pb.go",
			want:               true,
		},
		{
			name:               "extension does not match on partial name",
			includedExtensions: []string{".go"},
			path:               "tools/cargo",
			want:               false,
		},
		{
			name:               "extension entry is trimmed",
			includedExtensions: []string{" .go "},
			path:               "main.go",
			want:               true,
		},

		// Patterns: glob match on the file name. The motivating .env family.
		{
			name:             "pattern matches bare .env",
			includedPatterns: []string{".env*"},
			path:             ".env",
			want:             true,
		},
		{
			name:             "pattern matches .env.development",
			includedPatterns: []string{".env*"},
			path:             "config/.env.development",
			want:             true,
		},
		{
			name:             "pattern matches .env.local",
			includedPatterns: []string{".env*"},
			path:             ".env.local",
			want:             true,
		},
		{
			name:             "pattern matches .env.development.local",
			includedPatterns: []string{".env*"},
			path:             ".env.development.local",
			want:             true,
		},
		{
			name:             "pattern does not match unrelated .local file",
			includedPatterns: []string{".env*"},
			path:             "config.local",
			want:             false,
		},
		{
			name:             "pattern does not match unrelated .development file",
			includedPatterns: []string{".env*"},
			path:             "notes.development",
			want:             false,
		},
		{
			name:             "suffix-style pattern matches",
			includedPatterns: []string{"*_templ.go"},
			path:             "views/page_templ.go",
			want:             true,
		},

		// Combined extensions and patterns.
		{
			name:               "matches via extension when patterns also set",
			includedExtensions: []string{".go"},
			includedPatterns:   []string{".env*"},
			path:               "main.go",
			want:               true,
		},
		{
			name:               "matches via pattern when extensions also set",
			includedExtensions: []string{".go"},
			includedPatterns:   []string{".env*"},
			path:               ".env.development",
			want:               true,
		},
		{
			name:               "matches neither extension nor pattern",
			includedExtensions: []string{".go"},
			includedPatterns:   []string{".env*"},
			path:               "README.md",
			want:               false,
		},

		// Robustness.
		{
			name:             "empty pattern entry does not match everything",
			includedPatterns: []string{"", "  "},
			path:             "anything.txt",
			want:             false,
		},
		{
			name:             "invalid pattern does not match and does not panic",
			includedPatterns: []string{"["},
			path:             "main.go",
			want:             false,
		},
		{
			name: "no extensions or patterns matches nothing",
			path: "main.go",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := Watcher{
				includedExtensions: tt.includedExtensions,
				includedPatterns:   tt.includedPatterns,
			}
			if got := w.isWatchedFile(tt.path); got != tt.want {
				t.Errorf("isWatchedFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
