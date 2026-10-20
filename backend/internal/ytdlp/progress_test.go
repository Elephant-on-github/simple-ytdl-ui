package ytdlp

import "testing"

func TestParseProgress(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Progress
		ok   bool
	}{
		{
			name: "documented sample line",
			line: "dl|downloading|5242880|10485760|1048576.5|3",
			want: Progress{Status: "downloading", Downloaded: 5242880, Total: 10485760, Speed: 1048576.5, ETA: 3},
			ok:   true,
		},
		{
			name: "finished line",
			line: "dl|finished|10485760|10485760|0|0",
			want: Progress{Status: "finished", Downloaded: 10485760, Total: 10485760},
			ok:   true,
		},
		{
			name: "full template shape with estimate",
			line: "dl|downloading|5242880|10485760|10485760|1048576.5|3",
			want: Progress{Status: "downloading", Downloaded: 5242880, Total: 10485760, Speed: 1048576.5, ETA: 3},
			ok:   true,
		},
		{
			name: "unknown sizes",
			line: "dl|downloading|NA|NA|NA|NA|NA",
			want: Progress{Status: "downloading"},
			ok:   true,
		},
		{
			name: "line without the dl prefix",
			line: "[download]  50.0% of 10.00MiB",
			ok:   false,
		},
		{
			name: "empty line",
			line: "",
			ok:   false,
		},
		{
			name: "too few fields",
			line: "dl|downloading|100",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseProgress(tt.line)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v (got %+v)", ok, tt.ok, got)
			}
			if !tt.ok {
				return
			}
			if got != tt.want {
				t.Errorf("ParseProgress = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseProgressTotalFallsBackToEstimate(t *testing.T) {
	// total_bytes is NA but the estimate is known.
	got, ok := ParseProgress("dl|downloading|100|NA|500|10|2")
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if got.Total != 500 {
		t.Errorf("Total = %d, want 500", got.Total)
	}
	if got.Speed != 10 {
		t.Errorf("Speed = %v, want 10", got.Speed)
	}
	if got.ETA != 2 {
		t.Errorf("ETA = %d, want 2", got.ETA)
	}
}
