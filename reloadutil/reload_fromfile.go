package reloadutil

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"sync"
)

// CertificateGetter satisfies ReloadFunc and its GetCertificate method
// satisfies the tls.GetCertificate function signature. Currently it does not
// allow changing paths after the fact.
type CertificateGetter struct {
	sync.RWMutex

	cert *tls.Certificate

	certFile   string
	keyFile    string
	passphrase string
}

func NewCertificateGetter(certFile, keyFile, passphrase string) *CertificateGetter {
	return &CertificateGetter{
		certFile:   certFile,
		keyFile:    keyFile,
		passphrase: passphrase,
	}
}

func (cg *CertificateGetter) Reload() error {
	certPEMBlock, err := ioutil.ReadFile(cg.certFile)
	if err != nil {
		return err
	}
	keyPEMBlock, err := ioutil.ReadFile(cg.keyFile)
	if err != nil {
		return err
	}

	cert, err := parsePEM(certPEMBlock, keyPEMBlock, []byte(cg.passphrase))
	if err != nil {
		return err
	}

	cg.Lock()
	defer cg.Unlock()

	cg.cert = cert
	return nil
}

func (cg *CertificateGetter) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cg.RLock()
	defer cg.RUnlock()

	if cg.cert == nil {
		return nil, fmt.Errorf("nil certificate")
	}

	return cg.cert, nil
}
