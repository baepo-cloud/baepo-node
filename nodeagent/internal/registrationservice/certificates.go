package registrationservice

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func (s *Service) AuthorityCertificate() *x509.Certificate {
	return s.authorityCert
}

func (s *Service) TLSCertificate() *tls.Certificate {
	return s.tlsCert
}

func parseCertificate(value []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(value)
	if block == nil || block.Type != "CA CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return cert, nil
}
