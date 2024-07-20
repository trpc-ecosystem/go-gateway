package fhttp

import (
	"crypto/tls"
	"crypto/x509"
)

// TLSOption is a function type used to set one attribute of *tls.Config.
type TLSOption func(o *tls.Config)

var (
	customTLSOptions = make([]TLSOption, 0)
)

// RegisterTLSConfig registers one or more TLSOption into the gateway.
func RegisterTLSConfig(opts ...TLSOption) {
	customTLSOptions = append(customTLSOptions, opts...)
}

// SetCustomTLSOptions applies all registered TLSOption to *tls.Config.
func SetCustomTLSOptions(tlsConfig *tls.Config) {
	for _, o := range customTLSOptions {
		o(tlsConfig)
	}
}

// WithTLSPeerVerifier creates a TLSOption used to set the VerifyPeerCertificate attribute in *tls.Config.
func WithTLSPeerVerifier(f func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error) TLSOption {
	return func(o *tls.Config) {
		o.VerifyPeerCertificate = f
	}
}
