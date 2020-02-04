package daemon

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/kr/mitm"
)

func genCA() (tls.Certificate, error) {
	hostname, _ := os.Hostname()
	certPEM, keyPEM, err := mitm.GenCA(hostname)
	if err != nil {
		return tls.Certificate{}, err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	return cert, err
}
