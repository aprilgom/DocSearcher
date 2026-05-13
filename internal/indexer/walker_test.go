package indexer

import "testing"

func TestIsSupportedDocumentFile(t *testing.T) {
	tests := map[string]bool{
		"report.hwp":    true,
		"report.hwpx":   true,
		"manual.pdf":    true,
		"notes.txt":     false,
		"draft.hwp.tmp": false,
		"~$lock.pdf":    false,
	}

	for path, want := range tests {
		if got := IsSupportedDocumentFile(path); got != want {
			t.Fatalf("IsSupportedDocumentFile(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestNormalizeNoSpaceContent(t *testing.T) {
	content := "한 글\nA\tB\r C"
	want := "한글ABC"

	if got := NormalizeNoSpaceContent(content); got != want {
		t.Fatalf("NormalizeNoSpaceContent() = %q, want %q", got, want)
	}
}
