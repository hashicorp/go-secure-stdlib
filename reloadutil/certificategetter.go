package reloadutil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// ReloadFunc are functions that are called when a reload is requested
type ReloadFunc func() error

type CertificateGetterIf interface {
	Reload() error
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

func parsePEM(certPEMBlock, keyPEMBlock, passphrase []byte) (*tls.Certificate, error) {
	k := make([]byte, len(keyPEMBlock))
	copy(k, keyPEMBlock)

	// Check for encrypted pem block
	keyBlock, _ := pem.Decode(k)
	if keyBlock == nil {
		return nil, errors.New("decoded PEM is blank")
	}

	if x509.IsEncryptedPEMBlock(keyBlock) {
		var err error
		keyBlock.Bytes, err = x509.DecryptPEMBlock(keyBlock, passphrase)
		if err != nil {
			return nil, fmt.Errorf("Decrypting PEM block failed: %w", err)
		}
		k = pem.EncodeToMemory(keyBlock)
	}

	cert, err := tls.X509KeyPair(certPEMBlock, k)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}
