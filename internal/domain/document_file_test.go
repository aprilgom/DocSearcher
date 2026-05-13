package domain

import "testing"

func TestIsSupportedDocumentPath(t *testing.T) {
	tests := map[string]bool{
		"report.hwp":    true,
		"report.hwpx":   true,
		"REPORT.HWPX":   true,
		"manual.pdf":    true,
		"notes.txt":     false,
		"draft.hwp.tmp": false,
		"~$lock.pdf":    false,
	}

	for path, want := range tests {
		if got := IsSupportedDocumentPath(path); got != want {
			t.Fatalf("IsSupportedDocumentPath(%q) = %v, want %v", path, got, want)
		}
	}
}
