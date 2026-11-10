package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMainVersionCommand(t *testing.T) {
	origArgs := os.Args
	origStdout := os.Stdout
	t.Cleanup(func() {
		os.Args = origArgs
		os.Stdout = origStdout
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	os.Args = []string{"xoscal", "version"}

	main()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	_ = r.Close()

	if got := strings.TrimSpace(buf.String()); got != "xoscal version "+version {
		t.Fatalf("unexpected version output %q", got)
	}
}
