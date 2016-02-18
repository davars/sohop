package sohop

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

func parseCert(certPem []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPem)
	if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
		return nil, fmt.Errorf("not a certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil

}

func CertValidity(certPem []byte) (notBefore, notAfter time.Time, err error) {
	cert, err := parseCert(certPem)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return cert.NotBefore, cert.NotAfter, nil
}
