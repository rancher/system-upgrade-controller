package framework

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
)

func ChannelServer(location string) *httptest.Server {
	hostname, err := os.Hostname()
	if err != nil {
		Failf("cannot read hostname: %v", err)
	}
	server := &httptest.Server{
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", location)
			w.WriteHeader(http.StatusFound)
		})},
	}
	server.Listener, err = net.Listen("tcp", net.JoinHostPort(hostname, "0"))
	if err != nil {
		Failf("cannot create tcp listener: %v", err)
	}
	server.Start()
	return server
}
