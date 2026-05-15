package domain

import "testing"

func TestRootIDValidation(t *testing.T) {
	valid := []string{"documents", "docs_2026", "docs-prod"}
	for _, id := range valid {
		if err := ValidateRootID(RootID(id)); err != nil {
			t.Fatalf("ValidateRootID(%q) returned error: %v", id, err)
		}
	}

	invalid := []string{"", "Documents", "docs/main", "docs:main", "docs main", "docs\\main", "docs."}
	for _, id := range invalid {
		if err := ValidateRootID(RootID(id)); err == nil {
			t.Fatalf("ValidateRootID(%q) returned nil, want error", id)
		}
	}
}

func TestRelativePathValidationRejectsMalformedLogicalPaths(t *testing.T) {
	valid := []string{"shared/2026/sample.hwp", "한글/문서.pdf", "..draft/file.hwp"}
	for _, path := range valid {
		if err := ValidateRelativePath(RelativePath(path)); err != nil {
			t.Fatalf("ValidateRelativePath(%q) returned error: %v", path, err)
		}
	}

	invalid := []string{
		"",
		"/absolute.hwp",
		"C:/docs/a.hwp",
		"../secret.hwp",
		"a/./b.hwp",
		"a//b.hwp",
		"a:b.hwp",
		`a\b.hwp`,
		"a?.hwp",
		"folder./a.hwp",
		"folder /a.hwp",
		"CON.txt",
		"docs/Lpt1.pdf",
		"line\nbreak.hwp",
	}
	for _, path := range invalid {
		if err := ValidateRelativePath(RelativePath(path)); err == nil {
			t.Fatalf("ValidateRelativePath(%q) returned nil, want error", path)
		}
	}
}

func TestLogicalDocumentIDBuildsAndParsesLiteralUTF8ID(t *testing.T) {
	id, err := NewLogicalDocumentID(RootID("documents"), RelativePath("공유/샘플 문서.hwp"))
	if err != nil {
		t.Fatalf("NewLogicalDocumentID returned error: %v", err)
	}
	if id != LogicalDocumentID("documents:공유/샘플 문서.hwp") {
		t.Fatalf("LogicalDocumentID = %q, want literal UTF-8 ID", id)
	}

	rootID, relPath, err := ParseLogicalDocumentID(id)
	if err != nil {
		t.Fatalf("ParseLogicalDocumentID returned error: %v", err)
	}
	if rootID != "documents" || relPath != "공유/샘플 문서.hwp" {
		t.Fatalf("ParseLogicalDocumentID = (%q, %q), want documents and relative path", rootID, relPath)
	}
}

func TestSMBShareMetadataValidationTrimsAndRejectsUnsupportedSyntax(t *testing.T) {
	meta, err := NewSMBShare(" docserver ", " documents ")
	if err != nil {
		t.Fatalf("NewSMBShare returned error: %v", err)
	}
	if meta.Host != "docserver" || meta.Share != "documents" {
		t.Fatalf("NewSMBShare = %#v, want trimmed host/share", meta)
	}

	invalidHosts := []string{"", "docserver:445", "user@docserver", "docs%31", "문서서버", "doc/server", "doc server", "doc?x", "doc#x"}
	for _, host := range invalidHosts {
		if _, err := NewSMBShare(host, "documents"); err == nil {
			t.Fatalf("NewSMBShare(%q, documents) returned nil, want error", host)
		}
	}

	invalidShares := []string{"", "docs/shared", "docs:archive", "docs*archive", "docs?archive", "docs#archive", "docs archive"}
	for _, share := range invalidShares {
		if _, err := NewSMBShare("docserver", share); err == nil {
			t.Fatalf("NewSMBShare(docserver, %q) returned nil, want error", share)
		}
	}
}
