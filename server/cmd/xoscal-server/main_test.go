package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mchorfa/xoscal/server/internal/config"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

func TestServerHelpersAuthTLSAndStores(t *testing.T) {
	jsonLogger := newLogger("debug", "json")
	textLogger := newLogger("info", "text")
	if jsonLogger == nil || textLogger == nil {
		t.Fatal("newLogger returned nil")
	}

	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "tokens.txt")
	if err := os.WriteFile(tokenFile, []byte("# comment\n  file-token-1 \n\nfile-token-2\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	tokens, err := configuredAuthTokens([]string{" inline-token ", ""}, tokenFile)
	if err != nil {
		t.Fatalf("configuredAuthTokens: %v", err)
	}
	for _, want := range []string{"inline-token", "file-token-1", "file-token-2"} {
		if !tokens[want] {
			t.Fatalf("missing token %q in %v", want, tokens)
		}
	}
	if _, err := configuredAuthTokens(nil, filepath.Join(dir, "missing-tokens.txt")); err == nil {
		t.Fatal("expected error for missing token file")
	}

	// TLS credential loading: disabled, partial error, valid server TLS, valid mTLS.
	if creds, err := loadTLSCredentials("", "", ""); err != nil || creds != nil {
		t.Fatalf("expected nil,nil when TLS paths empty, got %v, %v", creds, err)
	}
	if _, err := loadTLSCredentials("only-cert.pem", "", ""); err == nil {
		t.Fatal("expected error when key path is missing")
	}

	certPath, keyPath := writeSelfSignedKeyPair(t, dir)
	if creds, err := loadTLSCredentials(certPath, keyPath, ""); err != nil || creds == nil {
		t.Fatalf("expected valid server TLS credentials, got %v, %v", creds, err)
	}
	if creds, err := loadTLSCredentials(certPath, keyPath, certPath); err != nil || creds == nil {
		t.Fatalf("expected valid mTLS credentials, got %v, %v", creds, err)
	}
	badCA := filepath.Join(dir, "bad-ca.pem")
	if err := os.WriteFile(badCA, []byte("not-a-pem"), 0o600); err != nil {
		t.Fatalf("write bad CA: %v", err)
	}
	if _, err := loadTLSCredentials(certPath, keyPath, badCA); err == nil {
		t.Fatal("expected error for invalid client CA PEM")
	}

	// Store retry helpers against real SQLite DSN.
	dsn := filepath.Join(dir, "server.db")
	s, err := openStoreWithRetry(dsn, dbutil.DefaultPool(), 1, textLogger)
	if err != nil {
		t.Fatalf("openStoreWithRetry: %v", err)
	}
	_ = s.Close()

	ks, err := openKGWithRetry(dsn, dbutil.DefaultPool(), 1, textLogger)
	if err != nil {
		t.Fatalf("openKGWithRetry: %v", err)
	}
	_ = ks.Close()

	vs, err := openVectorWithRetry(config.Vector{Backend: "sqlite", URI: dsn}, dbutil.DefaultPool(), 1, textLogger)
	if err != nil {
		t.Fatalf("openVectorWithRetry: %v", err)
	}
	_ = vs.Close()
}

func writeSelfSignedKeyPair(t *testing.T, dir string) (string, string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certPath := filepath.Join(dir, "tls.crt")
	keyPath := filepath.Join(dir, "tls.key")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return certPath, keyPath
}
