// HTTP handler functions, ready for reuse.

package service

import (
	"crypto/tls"
	"net/http"

	"github.com/golang/glog"
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

// Server configures and runs a TLS-enabled server from passed-in flags using
// the supplied handler.  Listenaddr is the address to listen to (e.g. "localhost:8080"),
// handlerUrlPath is the URL path to respond to (e.g. "/handler").
func Server(listenAddr, handlerUrlPath string, handler HandlerFunc) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(handlerUrlPath, handler)
	// TODO(filmil): Check how to install a handler that returns 404 for
	// everything else.

	cfg := &tls.Config{
		// TODO(fmil): Figure out how to not skip verify.
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
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
	return &http.Server{
		Addr:      listenAddr,
		Handler:   mux,
		TLSConfig: cfg,
		TLSNextProto: make(map[string]func(
			*http.Server, *tls.Conn, http.Handler), 0),
	}
}
