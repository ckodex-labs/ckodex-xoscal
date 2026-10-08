package bundle

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

func canonicalMember(name string) error {
	if !fs.ValidPath(name) || name == "." || path.Clean(name) != name || !norm.NFC.IsNormalString(name) {
		return fmt.Errorf("non-canonical bundle member: %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if strings.ContainsAny(part, `\<>:"|?*`) || strings.TrimRight(part, ". ") != part {
			return fmt.Errorf("non-portable bundle member: %q", name)
		}
		for _, char := range part {
			if char < 32 || char == 127 {
				return fmt.Errorf("invalid bundle member: %q", name)
			}
		}
		if reservedPortableBase(part) {
			return fmt.Errorf("reserved bundle member: %q", name)
		}
	}
	return nil
}

func reservedPortableBase(name string) bool {
	if index := strings.IndexAny(name, ".:"); index >= 0 {
		name = name[:index]
	}
	name = strings.ToUpper(strings.TrimRight(name, " "))
	switch name {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	if !strings.HasPrefix(name, "COM") && !strings.HasPrefix(name, "LPT") {
		return false
	}
	suffix := name[3:]
	return (len(suffix) == 1 && suffix[0] >= '1' && suffix[0] <= '9') || suffix == "¹" || suffix == "²" || suffix == "³"
}

func categoryMember(name, category string) error {
	if err := canonicalMember(name); err != nil {
		return err
	}
	if path.Dir(name) != category {
		return fmt.Errorf("bundle member %q must be directly inside %s", name, category)
	}
	return nil
}

func registerMember(members map[string]bool, name string) error {
	if err := canonicalMember(name); err != nil {
		return err
	}
	key := memberKey(name)
	if members[key] {
		return fmt.Errorf("colliding bundle member: %q", name)
	}
	members[key] = true
	return nil
}

func memberKey(name string) string {
	return cases.Fold().String(norm.NFC.String(name))
}

type boundedBundleWriter struct {
	writer    io.Writer
	remaining int64
}

func (writer *boundedBundleWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > writer.remaining {
		return 0, fmt.Errorf("decompressed bundle exceeds configured byte limit")
	}
	n, err := writer.writer.Write(data)
	writer.remaining -= int64(n)
	return n, err
}
