// Example to generate new self-signed credentials:
//  openssl genrsa -out server.key 2048
//  openssl ecparam -genkey -name secp384r1 -out server.key
//  openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650

package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"time"
)

var tlsCrt = "server.crt"
var tlsKey = "server.key"
var serviceListen = "127.0.0.1:12345"
var reverseProxyListen = "127.0.0.1:56789"
var reverseProxyTo = "https://127.0.0.1:12345"

func main() {
	go service()
	go reverseProxy()
	time.Sleep(time.Second)
	err := exec.Command("curl", "--trace", "trace", "-d", "@main.go", "http://127.0.0.1:56789/").Run()
	if err != nil {
		panic(err)
	}
}

func service() {
	err := http.ListenAndServeTLS(serviceListen, tlsCrt, tlsKey, &serviceHandler{})
	if err != nil {
		panic(err)
	}
}

type serviceHandler struct {
}

func (sh *serviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Write and flush with time delay to force chunked output.
	for i := 0; i < 5; i++ {
		w.Write([]byte("test\r\n"))
		f, ok := w.(http.Flusher)
		if !ok {
			panic("no flush")
		}
		f.Flush()
		time.Sleep(time.Second)
	}
}

func reverseProxy() {
	reverseProxyToURL, err := url.Parse(reverseProxyTo)
	if err != nil {
		panic(err)
	}
	rp := httputil.NewSingleHostReverseProxy(reverseProxyToURL)
	// Ensure chunked doesn't get buffered into non-chunked.
	rp.FlushInterval = time.Millisecond
	rp.Transport = http.DefaultTransport
	// Prime things, could cause a tls error but fills out the default configuration.
	r, err := http.NewRequest("GET", reverseProxyTo, nil)
	if err != nil {
		panic(err)
	}
	rp.Transport.RoundTrip(r)
	t, ok := rp.Transport.(*http.Transport)
	if !ok {
		panic("no transport")
	}
	if t.TLSClientConfig == nil {
		panic("no config")
	}
	// All that priming so we can toggle this hopefully without other configuration side-effects.
	t.TLSClientConfig.InsecureSkipVerify = true
	err = http.ListenAndServe(reverseProxyListen, rp)
	if err != nil {
		panic(err)
	}
}
