package cmd

import (
	"errors"
	"testing"
)

func TestNormalizeTimeSpent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "hours", input: "2h", want: "2h"},
		{name: "compact", input: "1h30m", want: "1h 30m"},
		{name: "all units", input: "1W 2D 3H 4M", want: "1w 2d 3h 4m"},
		{name: "trims spaces", input: " 30m ", want: "30m"},
		{name: "text", input: "tomorrow", wantErr: true},
		{name: "decimal", input: "1.5h", wantErr: true},
		{name: "zero", input: "0h", wantErr: true},
		{name: "duplicate", input: "1h 2h", wantErr: true},
		{name: "wrong order", input: "30m 1h", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeTimeSpent(tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidTimeSpent) {
					t.Fatalf("normalizeTimeSpent() error = %v, want ErrInvalidTimeSpent", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("normalizeTimeSpent() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeTimeSpent() = %q, want %q", got, tt.want)
			}
		})
	}
}
