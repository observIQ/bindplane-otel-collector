// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpevents

import "net/http"

type authMiddleware interface {
	auth(next http.Handler) http.Handler
	name() string
}

type authToken struct {
	tokenHeader string
	tokens      []string
}

func (a authToken) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET requests without a body do not require auth
		if r.Method == http.MethodGet && r.Body == http.NoBody {
			return
		}

		token := r.Header.Get(a.tokenHeader)

		for _, validToken := range a.tokens {
			if validToken == token {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusForbidden)
	})
}

func (a authToken) name() string {
	return "token-auth"
}

type authBasic struct {
	username string
	password string
}

func (a authBasic) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET requests without a body do not require auth
		if r.Method == http.MethodGet && r.Body == http.NoBody {
			return
		}

		u, p, ok := r.BasicAuth()
		if ok {
			if u == a.username && p == a.password {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusForbidden)
	})
}

func (a authBasic) name() string {
	return "basic-auth"
}
