package git

import (
	"testing"
)

func TestParsePRFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantRepo string
		wantPR   int
		wantErr  bool
	}{
		{
			name:     "standard github PR URL",
			url:      "https://github.com/aschreifels/cwt/pull/42",
			wantRepo: "aschreifels/cwt",
			wantPR:   42,
		},
		{
			name:     "URL with trailing slash",
			url:      "https://github.com/aschreifels/cwt/pull/42/",
			wantRepo: "aschreifels/cwt",
			wantPR:   42,
		},
		{
			name:     "URL with files tab",
			url:      "https://github.com/org/repo/pull/123",
			wantRepo: "org/repo",
			wantPR:   123,
		},
		{
			name:    "no pull in URL",
			url:     "https://github.com/org/repo",
			wantErr: true,
		},
		{
			name:    "invalid PR number",
			url:     "https://github.com/org/repo/pull/abc",
			wantErr: true,
		},
		{
			name:    "too short URL",
			url:     "https://github.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, pr, err := ParsePRFromURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got repo=%q pr=%d", repo, pr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo: got %q, want %q", repo, tt.wantRepo)
			}
			if pr != tt.wantPR {
				t.Errorf("pr: got %d, want %d", pr, tt.wantPR)
			}
		})
	}
}
