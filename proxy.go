// KatWeb by kittyhacker101 - HTTP(S) / Websockets Reverse Proxy
package main

import (
	"crypto/tls"
	"encoding/json"
	"github.com/yhat/wsutil"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/kittyhacker101/KatWeb/logs"
)

// UpdateData contains a struct for parsing returned json from the request
type UpdateData struct {
	Latest string `json:"tag_name"`
}

var (
	upd UpdateData

	tlsp = &tls.Config{
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	proxy = &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			prox, loc := GetProxy(r)
			u, err := url.Parse(prox + strings.TrimPrefix(r.URL.String(), "/"+loc))
			if err == nil {
				r.URL = u
				return
			}
			r.URL = fixProxy(r.URL, loc)
			r.Host = r.URL.Host
		},
		ErrorLog: Logger,
		Transport: &http.Transport{
			TLSClientConfig:     tlsp,
			MaxIdleConns:        4096,
			MaxIdleConnsPerHost: 256,
			IdleConnTimeout:     time.Duration(conf.DatTime*8) * time.Second,
		},
	}

	wsproxy = &wsutil.ReverseProxy{
		Director: func(r *http.Request) {
			prox, loc := GetProxy(r)
			u, err := url.Parse(prox + strings.TrimPrefix(r.URL.String(), "/"+loc))
			if err != nil {
				r.URL = fixProxy(r.URL, loc)
				return
			}

			if r.URL.Scheme == "https" {
				u.Scheme = "wss://"
			} else {
				u.Scheme = "ws://"
			}

			r.URL = u
		},
		ErrorLog:        Logger,
		TLSClientConfig: tlsp,
	}

	// updateClient is the http.Client used for checking the latest version of KatWeb
	updateClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:   tlsp,
		},
		Timeout: 2 * time.Second,
	}

	proxyMap, redirMap sync.Map
)

// fixProxy proxies requests to the local server if the proxy's URL cannot be parsed
func fixProxy(u *url.URL, loc string) *url.URL {
	u = &url.URL{
		Scheme: "http",
		Host:   "localhost",
		Path:   strings.TrimPrefix(u.String(), "/"+loc),
	}
	if conf.HSTS {
		u.Scheme = "https"
		if conf.Adv.HTTPS != 443 {
			u.Host = u.Host + ":" + strconv.Itoa(conf.Adv.HTTPS)
		}
	} else if conf.Adv.HTTP != 80 {
		u.Host = u.Host + ":" + strconv.Itoa(conf.Adv.HTTP)
	}

	return u
}

// GetProxy finds the correct proxy index to use from the conf.Proxy struct
func GetProxy(r *http.Request) (string, string) {
	url, err := url.QueryUnescape(r.URL.EscapedPath())
	if err != nil {
		url = r.URL.EscapedPath()
	}
	urlp := strings.Split(url, "/")

	if val, ok := proxyMap.Load(r.Host); ok {
		return val.(string), r.Host
	}

	if len(urlp) == 0 {
		return "", ""
	}

	if val, ok := proxyMap.Load(urlp[1]); ok {
		return val.(string), urlp[1]
	}

	return "", ""
}

// MakeProxyMap converts the conf.Proxy into a map[string]string
func MakeProxyMap() {
	for i := range conf.Proxy {
		proxyMap.Store(conf.Proxy[i].Loc, conf.Proxy[i].URL)
	}
	for i := range conf.Redir {
		redirMap.Store(conf.Redir[i].Loc, conf.Redir[i].URL)
	}
}

// ProxyRequest reverse-proxies a request, or websocket
func ProxyRequest(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Connection"), "Upgrade") && strings.Contains(r.Header.Get("Upgrade"), "websocket") {
		wsproxy.ServeHTTP(w, r)
	} else {
		proxy.ServeHTTP(w, r)
	}
}

// CheckUpdate checks if you are using the latest version of KatWeb
func CheckUpdate(current string) logs.Entry {
	resp, err := updateClient.Get("https://api.github.com/repos/kittyhacker101/KatWeb/releases/latest")
	if err != nil {
		return logs.Entry{
			&logs.FailedToContactGithubAPIforUpdate{Resp: resp, Err: err}}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return logs.Entry{
			&logs.FailedToReadGithubAPIUpdatesResponseBody{Err: err}}
	}
	if err := json.Unmarshal(body, &upd); err != nil {
		return logs.Entry{
			&logs.FailedToParseGithubAPIUpdatesResponse{Err: err}}
	}
	if upd.Latest == "" {
		return logs.Entry{
			(*logs.EmptyGithubAPIUpdatesResponse)(nil)}
	}

	currenti, err := strconv.ParseFloat(current[3:], 32)
	if err != nil {
		return logs.Entry{&logs.UnableToParseVersionNumber{
			CurrentOrUpdates: logs.Current, What: current, Err: err}}
	}
	latesti, err := strconv.ParseFloat(upd.Latest[3:], 32)
	if err != nil {
		return logs.Entry{&logs.UnableToParseVersionNumber{
			CurrentOrUpdates: logs.Latest, What: upd.Latest, Err: err}}
	}

	return logs.Entry{&logs.CurrentAndLatest{
		Current: float32(currenti),
		Latest:  float32(latesti),
	}}
}
