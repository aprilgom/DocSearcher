package domain

import "testing"

func TestNewIndexedDocumentNormalizesSearchText(t *testing.T) {
	doc := NewDocument("report.hwpx", "한 글\nA\tB\r C")

	indexed := NewIndexedDocument(doc)

	if indexed.ID != "report.hwpx" {
		t.Fatalf("ID = %q, want %q", indexed.ID, "report.hwpx")
	}
	if indexed.ContentNoSpace != "한글ABC" {
		t.Fatalf("ContentNoSpace = %q, want %q", indexed.ContentNoSpace, "한글ABC")
	}
}

func TestSearchModeFromFlags(t *testing.T) {
	tests := []struct {
		name         string
		exact        bool
		ignoreSpaces bool
		want         SearchMode
	}{
		{
			name:         "default_query",
			exact:        false,
			ignoreSpaces: false,
			want:         SearchModeQuery,
		},
		{
			name:         "exact",
			exact:        true,
			ignoreSpaces: false,
			want:         SearchModeExact,
		},
		{
			name:         "ignore_spaces",
			exact:        false,
			ignoreSpaces: true,
			want:         SearchModeIgnoreSpaces,
		},
		{
			name:         "ignore_spaces_precedence",
			exact:        true,
			ignoreSpaces: true,
			want:         SearchModeIgnoreSpaces,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := SearchModeFromFlags(tt.exact, tt.ignoreSpaces)

			if mode != tt.want {
				t.Fatalf("SearchModeFromFlags(%v, %v) = %v, want %v", tt.exact, tt.ignoreSpaces, mode, tt.want)
			}
		})
	}
}

func TestPersonNameSearchPolicyDefinesIndexRules(t *testing.T) {
	policy := PersonNameSearchPolicy()

	if policy.MinQueryLength != 2 {
		t.Fatalf("MinQueryLength = %d, want %d", policy.MinQueryLength, 2)
	}
	if policy.PartialMatchMinGram != 2 {
		t.Fatalf("PartialMatchMinGram = %d, want %d", policy.PartialMatchMinGram, 2)
	}
	if policy.PartialMatchMaxGram != 5 {
		t.Fatalf("PartialMatchMaxGram = %d, want %d", policy.PartialMatchMaxGram, 5)
	}
	if !policy.IgnoreWhitespace {
		t.Fatal("IgnoreWhitespace = false, want true")
	}
}

func TestIndexSchemaDefinesSearchFields(t *testing.T) {
	schema := DefaultIndexSchema()

	if schema.ContentField != "content" {
		t.Fatalf("ContentField = %q, want %q", schema.ContentField, "content")
	}
	if schema.ContentNoSpaceField != "content_nospace" {
		t.Fatalf("ContentNoSpaceField = %q, want %q", schema.ContentNoSpaceField, "content_nospace")
	}
	if schema.PathField != "path" {
		t.Fatalf("PathField = %q, want %q", schema.PathField, "path")
	}
}

func TestSearchRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "shorter_than_policy",
			query:   "김",
			wantErr: true,
		},
		{
			name:    "trimmed_shorter_than_policy",
			query:   " 김 ",
			wantErr: true,
		},
		{
			name:    "policy_length",
			query:   "김철",
			wantErr: false,
		},
		{
			name:    "longer_than_policy",
			query:   "김철수",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SearchRequest{Query: tt.query, Mode: SearchModeQuery}

			err := req.Validate(PersonNameSearchPolicy())

			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate returned nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
		})
	}
}
