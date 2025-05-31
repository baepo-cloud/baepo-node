package types

import (
	"crypto/tls"
	"crypto/x509"
)

type (
	RegistrationService interface {
		AuthorityCertificate() *x509.Certificate

		TLSCertificate() *tls.Certificate
	}
)
