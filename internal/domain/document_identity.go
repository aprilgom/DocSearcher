package domain

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RootID string

type RelativePath string

type LogicalDocumentID string

type SMBShare struct {
	Host  string
	Share string
}

type SMBAlias = SMBShare

type DocumentRoot struct {
	ID         RootID
	Name       string
	ServerPath string
	SMB        SMBShare
	SMBAliases []SMBAlias
}

var rootIDPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

func ValidateRootID(id RootID) error {
	value := string(id)
	if value == "" {
		return fmt.Errorf("root_id is required")
	}
	if !rootIDPattern.MatchString(value) {
		return fmt.Errorf("root_id %q must use lowercase ASCII letters, numbers, _, and -", value)
	}
	return nil
}

func ValidateRelativePath(relativePath RelativePath) error {
	value := string(relativePath)
	if value == "" {
		return fmt.Errorf("relative_path is required")
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("relative_path must be valid UTF-8")
	}
	if strings.HasPrefix(value, "/") || path.IsAbs(value) {
		return fmt.Errorf("relative_path must be relative")
	}
	if hasWindowsDrivePrefix(value) {
		return fmt.Errorf("relative_path must not contain a Windows drive prefix")
	}
	if strings.ContainsAny(value, `:\*?"<>|`) {
		return fmt.Errorf("relative_path contains Windows-unsafe characters")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("relative_path must not contain control characters")
		}
	}

	segments := strings.Split(value, "/")
	for _, segment := range segments {
		if segment == "" {
			return fmt.Errorf("relative_path must not contain empty segments")
		}
		if segment == "." || segment == ".." {
			return fmt.Errorf("relative_path must not contain dot segments")
		}
		if strings.HasSuffix(segment, " ") || strings.HasSuffix(segment, ".") {
			return fmt.Errorf("relative_path segment %q must not end in space or dot", segment)
		}
		if isReservedWindowsDeviceName(segment) {
			return fmt.Errorf("relative_path segment %q uses a reserved Windows device name", segment)
		}
	}

	return nil
}

func NewLogicalDocumentID(rootID RootID, relativePath RelativePath) (LogicalDocumentID, error) {
	if err := ValidateRootID(rootID); err != nil {
		return "", err
	}
	if err := ValidateRelativePath(relativePath); err != nil {
		return "", err
	}
	return LogicalDocumentID(string(rootID) + ":" + string(relativePath)), nil
}

func ParseLogicalDocumentID(id LogicalDocumentID) (RootID, RelativePath, error) {
	root, relativePath, ok := strings.Cut(string(id), ":")
	if !ok {
		return "", "", fmt.Errorf("document_id must contain root_id and relative_path")
	}
	rootID := RootID(root)
	rel := RelativePath(relativePath)
	if err := ValidateRootID(rootID); err != nil {
		return "", "", err
	}
	if err := ValidateRelativePath(rel); err != nil {
		return "", "", err
	}
	return rootID, rel, nil
}

func NewSMBShare(host string, share string) (SMBShare, error) {
	host = strings.TrimSpace(host)
	share = strings.TrimSpace(share)
	if err := validateSMBHost(host); err != nil {
		return SMBShare{}, err
	}
	if err := validateSMBShare(share); err != nil {
		return SMBShare{}, err
	}
	return SMBShare{Host: host, Share: share}, nil
}

func hasWindowsDrivePrefix(value string) bool {
	if len(value) < 2 || value[1] != ':' {
		return false
	}
	r := value[0]
	return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}

func isReservedWindowsDeviceName(segment string) bool {
	base := segment
	if before, _, ok := strings.Cut(segment, "."); ok {
		base = before
	}
	base = strings.ToUpper(base)
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) {
		return base[3] >= '1' && base[3] <= '9'
	}
	return false
}

func validateSMBHost(host string) error {
	if host == "" {
		return fmt.Errorf("smb_host is required")
	}
	for _, r := range host {
		if r > unicode.MaxASCII {
			return fmt.Errorf("smb_host must be ASCII")
		}
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("smb_host must not contain whitespace or control characters")
		}
	}
	if strings.ContainsAny(host, `%:/\@?#`) {
		return fmt.Errorf("smb_host contains unsupported syntax")
	}
	return nil
}

func validateSMBShare(share string) error {
	if share == "" {
		return fmt.Errorf("smb_share is required")
	}
	for _, r := range share {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("smb_share must not contain whitespace or control characters")
		}
	}
	if strings.ContainsAny(share, `\/?#:*"<>|`) {
		return fmt.Errorf("smb_share contains unsupported syntax")
	}
	return nil
}
