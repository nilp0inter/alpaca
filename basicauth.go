// Copyright 2025 The Alpaca Authors
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
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

func basicAuthMiddleware(next http.Handler, credentials string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if credentials == "" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Proxy-Authorization")
		if auth == "" {
			w.Header().Set("Proxy-Authenticate", `Basic realm="proxy"`)
			w.WriteHeader(http.StatusProxyAuthRequired)
			return
		}

		const prefix = "Basic "
		if !strings.HasPrefix(auth, prefix) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		encoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		parts := strings.SplitN(string(encoded), ":", 2)
		if len(parts) != 2 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		user, pass := parts[0], parts[1]

		credsParts := strings.SplitN(credentials, ":", 2)
		if len(credsParts) != 2 {
			// This would be a misconfiguration of the server, so we should probably
			// log an error and deny access.
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		expectedUser, expectedPass := credsParts[0], credsParts[1]

		userMatch := (subtle.ConstantTimeCompare([]byte(user), []byte(expectedUser)) == 1)
		passMatch := (subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) == 1)

		if userMatch && passMatch {
			next.ServeHTTP(w, r)
		} else {
			w.Header().Set("Proxy-Authenticate", `Basic realm="proxy"`)
			w.WriteHeader(http.StatusProxyAuthRequired)
		}
	})
}