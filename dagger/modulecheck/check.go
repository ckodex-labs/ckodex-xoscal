// Package modulecheck holds the byte comparison executed by module admission.
package modulecheck

// Script compares root manifests after actual tidy, then writes admission.
const Script = `set -eu
go mod tidy
cmp "$1" go.mod
cmp "$2" go.sum
printf 'module-tidy-ok\n' > "$3"
`
