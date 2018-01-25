// HTTP handler functions, ready for reuse.

package service

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"net/http"

	"fmt"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenAddr = flag.String(
		"listen_hostport", ":8000", "The hostport to listen to.")
	metricsPort = flag.Int("metrics-port", 8675, "The port to export prometheus metrics on.")
	serverCertFile = flag.String(
		"server_cert", "server.crt", "The server certificate file.")
	serverKeyFile = flag.String(
		"server_key", "server.key", "The server private key file.")
	handlerUrlPath = flag.String(
		"handler_url_path", "/", "The default handler URL path.")
)

// HandleFunc is a shorthand for a HTTP handler function.
type HandlerFunc func(http.ResponseWriter, *http.Request)

// NoCache positively turns off page caching.
func NoCache(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(
			"Cache-Control",
			"no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		handler(w, req)
	}

}

// WithRequestLogging decorates handler with a log statement that prints the
// method and the URL requested.
func WithRequestLogging(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		glog.Infof("Method: %v, URL: %v", req.Method, req.URL)
		handler(w, req)
	}
}

// WithStrictTransport decorates handler to require strict transport security
// when serving HTTPS request.
func WithStrictTransport(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Strict-Transport-Security",
			"max-age=86400; includeSubdomains")
		handler(w, req)
	}
}

func configTLS(clientCert []byte) *tls.Config {
	glog.Infof("Using server certificate file: %v", *serverCertFile)
	glog.Infof("Using server private key file: %v", *serverKeyFile)
	sCert, err := tls.LoadX509KeyPair(*serverCertFile, *serverKeyFile)
	if err != nil {
		glog.Fatal(err)
	}

	config := tls.Config{
		Certificates: []tls.Certificate{sCert},
		MinVersion:   tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	if clientCert != nil {
		clientCertPool := x509.NewCertPool()
		clientCertPool.AppendCertsFromPEM(clientCert)
		config.ClientCAs = clientCertPool
		config.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		glog.Warning("Not verifying client cert")
	}

	return &config
}

// ServeMetrics spins up a standalone metrics HTTP endpoint.
func ServeMetrics() {
	// Expose prometheus metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", *metricsPort), nil)
	if err != nil {
		glog.Fatalf("HTTP ListenAndServe for metrics: %+v", err)
	}
}

// Server configures and a https server from passed-in flags using
// the supplied handler. If clientCert is not nil,
func Server(handler HandlerFunc, clientCert []byte) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(*handlerUrlPath, handler)

	// TODO(filmil): Check how to install a handler that returns 404 for
	// everything else.

	return &http.Server{
		Addr:      *listenAddr,
		Handler:   mux,
		TLSConfig: configTLS(clientCert),
		TLSNextProto: make(map[string]func(
			*http.Server, *tls.Conn, http.Handler), 0),
	}
}
