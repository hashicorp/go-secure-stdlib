package reloadutil

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"sync"
)

type ValueCertificateGetter struct {
	sync.RWMutex

	c *tls.Certificate

	certFile   string // TBD: Should we also add support for this to just be passed as value like we did for key?
	key        []byte
	passphrase []byte
}

var _ CertificateGetterIf = &ValueCertificateGetter{}

func NewValueCertificateGetter(certFile string, key, passphrase []byte) (*ValueCertificateGetter, error) {
	certPEMBlock, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	cert, err := parsePEM(certPEMBlock, key, passphrase)
	if err != nil {
		return nil, err
	}

	return &ValueCertificateGetter{
		certFile:   certFile,
		key:        key,
		passphrase: passphrase,
		c:          cert,
	}, nil
}

func (vcg *ValueCertificateGetter) Reload() error {
	return fmt.Errorf("reload called on value certificate getter")
}

func (vcg *ValueCertificateGetter) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	vcg.RLock()
	defer vcg.RUnlock()

	if vcg.c == nil {
		return nil, fmt.Errorf("nil certificate")
	}

	return vcg.c, nil
}
