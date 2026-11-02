package store

import (
	"strings"
)

// DetectKeyType names a private key's format from its PEM header.
func DetectKeyType(value string) string {
	for _, k := range []struct{ marker, label string }{
		{"BEGIN OPENSSH PRIVATE KEY", "OpenSSH"},
		{"BEGIN RSA PRIVATE KEY", "RSA"},
		{"BEGIN EC PRIVATE KEY", "EC"},
		{"BEGIN ENCRYPTED PRIVATE KEY", "Encrypted PKCS#8"},
		{"BEGIN PRIVATE KEY", "PKCS#8"},
		{"BEGIN PGP PRIVATE KEY", "PGP"},
	} {
		if strings.Contains(value, k.marker) {
			return k.label
		}
	}

	return "Other"
}
