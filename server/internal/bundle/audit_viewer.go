package bundle

import _ "embed"

// AuditViewerHTML is a zero-dependency, self-contained HTML/CSS/JS application
// that runs completely offline in an air-gapped SCIF without internet connectivity.
//
//go:embed audit_viewer.html
var AuditViewerHTML string
