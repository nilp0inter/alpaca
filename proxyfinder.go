// Copyright 2019, 2021, 2022 The Alpaca Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)


func getProxyFromContext(req *http.Request) (*url.URL, error) {
	if value := req.Context().Value(contextKeyProxy); value != nil {
		proxy := value.(*url.URL)
		return proxy, nil
	}
	return nil, nil
}

type ProxyFinder struct {
	runner  *PACRunner
	fetcher *pacFetcher
	wrapper *PACWrapper
	blocked *blocklist
	sync.Mutex
}

func NewProxyFinder(pacurl string, wrapper *PACWrapper) *ProxyFinder {
	pf := &ProxyFinder{wrapper: wrapper, blocked: newBlocklist()}
	pf.runner = new(PACRunner)
	pf.fetcher = newPACFetcher(pacurl)
	pf.checkForUpdates()
	return pf
}

func (pf *ProxyFinder) WrapHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		pf.checkForUpdates()
		proxies, err := pf.findProxiesForRequest(req)
		if err != nil {
			log.Printf("[%d] %v", req.Context().Value(contextKeyID), err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx := context.WithValue(req.Context(), contextKeyProxies, proxies)
		// For backwards compatibility, also add the first usable proxy to the context.
		// This is used for non-CONNECT requests.
		for _, proxy := range proxies {
			if proxy == nil { // DIRECT
				break
			}
			if !pf.blocked.contains(proxy.Host) {
				ctx = context.WithValue(ctx, contextKeyProxy, proxy)
				break
			}
		}
		req = req.WithContext(ctx)
		next.ServeHTTP(w, req)
	})
}

func (pf *ProxyFinder) checkForUpdates() {
	pf.Lock()
	defer pf.Unlock()
	pacjs := pf.fetcher.download()
	if pacjs == nil {
		if !pf.fetcher.isConnected() {
			pf.blocked = newBlocklist()
			pf.wrapper.Wrap(nil)
		}
		return
	}
	pf.blocked = newBlocklist()
	if err := pf.runner.Update(pacjs); err != nil {
		log.Printf("Error running PAC JS: %q", err)
	} else {
		pf.wrapper.Wrap(pacjs)
	}
}

func (pf *ProxyFinder) findProxiesForRequest(req *http.Request) ([]*url.URL, error) {
	id := req.Context().Value(contextKeyID)
	if pf.fetcher == nil {
		log.Printf(`[%d] %s %s via "DIRECT"`, id, req.Method, req.URL)
		return []*url.URL{nil}, nil
	}
	if !pf.fetcher.isConnected() {
		log.Printf(`[%d] %s %s via "DIRECT" (not connected to PAC server)`,
			id, req.Method, req.URL)
		return []*url.URL{nil}, nil
	}
	str, err := pf.runner.FindProxyForURL(*req.URL)
	if err != nil {
		return nil, err
	}
	var proxies []*url.URL
	for _, elem := range strings.Split(str, ";") {
		fields := strings.Fields(strings.TrimSpace(elem))
		var scheme string
		var defaultPort string
		if len(fields) == 0 {
			continue
		} else if fields[0] == "DIRECT" {
			log.Printf("[%d] %s %s via %q", id, req.Method, req.URL, elem)
			proxies = append(proxies, nil)
		} else if fields[0] == "PROXY" || fields[0] == "HTTP" {
			scheme = "http"
			defaultPort = "80"
		} else if fields[0] == "HTTPS" {
			scheme = "https"
			defaultPort = "443"
		} else {
			log.Printf("[%d] Couldn't parse proxy: %q", id, elem)
			continue
		}
		if scheme != "" {
			proxy := &url.URL{Scheme: scheme, Host: fields[1]}
			if proxy.Port() == "" {
				proxy.Host = net.JoinHostPort(proxy.Host, defaultPort)
			}
			log.Printf("[%d] %s %s via %q", id, req.Method, req.URL, elem)
			proxies = append(proxies, proxy)
		}
	}
	if len(proxies) == 0 {
		return nil, errors.New("no proxies available")
	}
	return proxies, nil
}

func (pf *ProxyFinder) blockProxy(proxy string) {
	pf.blocked.add(proxy)
}
