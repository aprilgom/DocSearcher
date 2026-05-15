# Data Exposure Rules

- Logs may include `server_path` when needed for server-side diagnostics.
- Browser responses, root metadata responses, copy actions, and desktop open
  payloads must not include `server_path`.
- Error messages shown to desktop users may include resolved UNC, SMB URL, or
  local mounted paths for the file being opened.
- Error messages must not include credentials, environment variables, unrelated
  local paths, or Samba administrative configuration.
