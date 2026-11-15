package router

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
)

// ErrTLSFromConfigNeedsCertificates is returned when the configuration given
// to TLSFromConfig has neither Certificates nor GetCertificate.
var ErrTLSFromConfigNeedsCertificates = errors.New("router: TLSFromConfig needs Certificates or GetCertificate")

// TLSStrategy tells ListenTLS and ListenWithTLS where the server certificate
// comes from. Build it with TLSFromFiles or TLSFromConfig.
type TLSStrategy func() (serveStrategy, error)

// TLSFromFiles reads the certificate chain and its private key from PEM files.
func TLSFromFiles(cert, key string) TLSStrategy {
	return func() (serveStrategy, error) {
		return serveTLS(cert, key), nil
	}
}

// TLSFromConfig serves with a ready TLS configuration: certificates in memory
// (Certificates) or chosen per connection (GetCertificate), e.g. to renew them
// without restarting or to use autocert.
func TLSFromConfig(config *tls.Config) TLSStrategy {
	return func() (serveStrategy, error) {
		if config == nil || (len(config.Certificates) == 0 && config.GetCertificate == nil) {
			return serveStrategy{}, ErrTLSFromConfigNeedsCertificates
		}
		return serveTLSConfig(config), nil
	}
}

type serveStrategy struct {
	scheme string
	serve  func(server *http.Server, listener net.Listener) error
}

func servePlain() serveStrategy {
	return serveStrategy{
		scheme: "http",
		serve: func(server *http.Server, listener net.Listener) error {
			return server.Serve(listener)
		},
	}
}

func serveTLS(cert, key string) serveStrategy {
	return serveStrategy{
		scheme: "https",
		serve: func(server *http.Server, listener net.Listener) error {
			return server.ServeTLS(listener, cert, key)
		},
	}
}

func serveTLSConfig(config *tls.Config) serveStrategy {
	return serveStrategy{
		scheme: "https",
		serve: func(server *http.Server, listener net.Listener) error {
			server.TLSConfig = config
			return server.ServeTLS(listener, "", "")
		},
	}
}
