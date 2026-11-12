package router

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Rafael24595/go-assert/assert"
	"github.com/Rafael24595/go-web/router/result"
)

func started(t *testing.T, r *Router, listen func() error) (net.Addr, chan error) {
	t.Helper()

	addrs := make(chan net.Addr, 2)

	r.OnListen(func(addr net.Addr) {
		addrs <- addr
	})

	result := make(chan error, 1)

	go func() {
		result <- listen()
	}()

	select {
	case addr := <-addrs:
		return addr, result
	case err := <-result:
		t.Fatalf("listen returned before listening: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("the server did not start")
	}

	return nil, nil
}

func returns(t *testing.T, result chan error) error {
	t.Helper()

	select {
	case err := <-result:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("listen did not return")
		return nil
	}
}

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Nil(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "go-web test"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	assert.Nil(t, err)

	leaf, err := x509.ParseCertificate(der)
	assert.Nil(t, err)

	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
		Leaf:        leaf,
	}
}

func writePEM(t *testing.T, cert tls.Certificate) (string, string) {
	t.Helper()

	key, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	assert.True(t, ok)

	keyDer, err := x509.MarshalECPrivateKey(key)
	assert.Nil(t, err)

	dir := t.TempDir()

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer})

	assert.Nil(t, os.WriteFile(certPath, certPEM, 0o600))
	assert.Nil(t, os.WriteFile(keyPath, keyPEM, 0o600))

	return certPath, keyPath
}

func inMemory(cert tls.Certificate) TLSStrategy {
	return TLSFromConfig(&tls.Config{
		Certificates: []tls.Certificate{cert},
	})
}

func httpsClient(t *testing.T, cert tls.Certificate) *http.Client {
	t.Helper()

	pool := x509.NewCertPool()
	pool.AddCert(cert.Leaf)

	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{RootCAs: pool},
		ForceAttemptHTTP2: true,
	}

	t.Cleanup(transport.CloseIdleConnections)

	return &http.Client{Transport: transport}
}

func TestListen_StopsWhenTheContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	router := NewRouterWithMux(http.NewServeMux()).
		Context(ctx).
		Route(http.MethodGet, text("ok"), "/x")

	addr, result := started(t, router, func() error {
		return router.Listen("127.0.0.1:0")
	})

	hostport := addr.String()
	_, port, err := net.SplitHostPort(hostport)

	assert.Nil(t, err)
	assert.NotEqual(t, "0", port)

	res, body := get(t, "http://"+hostport+"/x")

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "ok", body)

	cancel()

	assert.Nil(t, returns(t, result))

	_, err = http.Get("http://" + hostport + "/x")
	assert.NotNil(t, err)
}

func TestListen_WaitsForRequestsInFlight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	inside := make(chan struct{})
	finished := false

	handler := func(w http.ResponseWriter, req *http.Request, c *Context) result.Result {
		close(inside)
		time.Sleep(200 * time.Millisecond)
		finished = true
		return result.TextOk("done")
	}

	router := NewRouterWithMux(http.NewServeMux()).
		Context(ctx).
		Route(http.MethodGet, handler, "/slow")

	addr, result := started(t, router, func() error {
		return router.Listen("127.0.0.1:0")
	})

	var wg sync.WaitGroup

	wg.Go(func() {
		hostport := addr.String()
		res, body := get(t, "http://"+hostport+"/slow")

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "done", body)
	})

	<-inside
	cancel()

	assert.Nil(t, returns(t, result))
	assert.True(t, finished)

	wg.Wait()
}

func TestListen_ShutdownTimeoutExceeded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	inside := make(chan struct{})
	release := make(chan struct{})

	defer close(release)

	handler := func(w http.ResponseWriter, req *http.Request, c *Context) result.Result {
		close(inside)
		<-release
		return result.TextOk("late")
	}

	router := NewRouterWithMux(http.NewServeMux()).
		Context(ctx).
		ShutdownTimeout(50*time.Millisecond).
		Route(http.MethodGet, handler, "/stuck")

	addr, result := started(t, router, func() error {
		return router.Listen("127.0.0.1:0")
	})

	hostport := addr.String()
	go http.Get("http://" + hostport + "/stuck")

	<-inside
	cancel()

	err := returns(t, result)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestListen_AddressInUse(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)

	defer taken.Close()

	called := false
	router := NewRouterWithMux(http.NewServeMux()).
		OnListen(func(net.Addr) {
			called = true
		})

	err = router.Listen(taken.Addr().String())

	assert.NotNil(t, err)
	assert.False(t, called)
}

func TestListen_ContextAlreadyDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	router := NewRouterWithMux(http.NewServeMux()).
		Context(ctx)

	result := make(chan error, 1)
	go func() {
		result <- router.Listen("127.0.0.1:0")
	}()

	assert.Nil(t, returns(t, result))
}

func TestListenWithTLS_StopsBothServers(t *testing.T) {
	cert := selfSignedCert(t)

	ctx, cancel := context.WithCancel(context.Background())

	router := NewRouterWithMux(http.NewServeMux()).
		Context(ctx).
		Route(http.MethodGet, text("secure"), "/x")

	var mu sync.Mutex
	var addrs []net.Addr

	ready := make(chan struct{}, 2)

	router.OnListen(func(addr net.Addr) {
		mu.Lock()
		addrs = append(addrs, addr)
		mu.Unlock()
		ready <- struct{}{}
	})

	result := make(chan error, 1)
	go func() {
		tls := inMemory(cert)
		result <- router.ListenWithTLS("127.0.0.1:0", "127.0.0.1:0", tls)
	}()

	for range 2 {
		select {
		case <-ready:
		case <-time.After(2 * time.Second):
			t.Fatal("both servers must start")
		}
	}

	cancel()
	assert.Nil(t, returns(t, result))

	mu.Lock()
	defer mu.Unlock()

	for _, addr := range addrs {
		_, err := net.DialTimeout("tcp", addr.String(), 200*time.Millisecond)
		assert.NotNil(t, err)
	}
}

func TestListenWithTLS_FailingTLSStopsTheHTTPServer(t *testing.T) {
	router := NewRouterWithMux(http.NewServeMux())

	result := make(chan error, 1)
	go func() {
		tls := TLSFromFiles("missing.crt", "missing.key")
		result <- router.ListenWithTLS("127.0.0.1:0", "127.0.0.1:0", tls)
	}()

	err := returns(t, result)
	assert.NotNil(t, err)
}

func TestListenWithTLS_FailingHTTPStopsTheTLSServer(t *testing.T) {
	cert := selfSignedCert(t)

	taken, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)

	defer taken.Close()

	router := NewRouterWithMux(http.NewServeMux())

	result := make(chan error, 1)
	go func() {
		tls := inMemory(cert)
		result <- router.ListenWithTLS(taken.Addr().String(), "127.0.0.1:0", tls)
	}()

	err = returns(t, result)
	assert.NotNil(t, err)
}

func TestListenTLS_Strategies(t *testing.T) {
	cert := selfSignedCert(t)

	certFile, keyFile := writePEM(t, cert)

	getCertificate := func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		return &cert, nil
	}

	strategies := map[string]TLSStrategy{
		"files":           TLSFromFiles(certFile, keyFile),
		"certificates":    inMemory(cert),
		"get certificate": TLSFromConfig(&tls.Config{GetCertificate: getCertificate}),
	}

	for name, strategy := range strategies {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			router := NewRouterWithMux(http.NewServeMux()).
				Context(ctx).
				Route(http.MethodGet, text("secure"), "/x")

			addr, result := started(t, router, func() error {
				return router.ListenTLS("127.0.0.1:0", strategy)
			})

			res, err := httpsClient(t, cert).Get("https://" + addr.String() + "/x")
			assert.Nil(t, err)

			defer res.Body.Close()

			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, 2, res.ProtoMajor)

			cancel()

			assert.Nil(t, returns(t, result))
		})
	}
}

func TestListenTLS_InvalidStrategies(t *testing.T) {
	strategies := []struct {
		name     string
		strategy TLSStrategy
		err      error
	}{
		{
			name:     "nil",
			strategy: nil,
			err:      ErrTLSStrategyRequired,
		},
		{
			name:     "nil config",
			strategy: TLSFromConfig(nil),
			err:      ErrTLSFromConfigNeedsCertificates,
		},
		{
			name:     "config without certificates",
			strategy: TLSFromConfig(&tls.Config{}),
			err:      ErrTLSFromConfigNeedsCertificates,
		},
	}

	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {
			called := false

			router := NewRouterWithMux(http.NewServeMux()).
				OnListen(func(net.Addr) {
					called = true
				})

			err := router.ListenTLS("127.0.0.1:0", tt.strategy)
			assert.Equal(t, tt.err, err)

			err = router.ListenWithTLS("127.0.0.1:0", "127.0.0.1:0", tt.strategy)
			assert.Equal(t, tt.err, err)

			assert.False(t, called)
		})
	}
}
