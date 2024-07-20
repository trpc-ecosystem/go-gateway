package fhttp

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterTlsConfig(t *testing.T) {
	RegisterTLSConfig(WithTLSPeerVerifier(func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		t.Log("this is a test for tls peer verifier who returns nil")
		return nil
	}))
	tlsConfig := &tls.Config{}
	SetCustomTLSOptions(tlsConfig)
	assert.Nil(t, tlsConfig.VerifyPeerCertificate(nil, nil))
	t.Log("TlsPeerVerifier test pass for nil return")

	errStr := "this is a test for tls peer verifier"
	RegisterTLSConfig(WithTLSPeerVerifier(func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		t.Log("this is a test for tls peer verifier who returns an error")
		return errors.New(errStr)
	}))
	SetCustomTLSOptions(tlsConfig)
	assert.NotNil(t, tlsConfig.VerifyPeerCertificate(nil, nil))
	assert.Equal(t, tlsConfig.VerifyPeerCertificate(nil, nil).Error(), errStr)
	t.Log("TlsPeerVerifier test pass for error return")
}
