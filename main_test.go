package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRobotsTxt(t *testing.T) {
	router := newRouter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("%s %s reached the image backend", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	server := httptest.NewServer(router)
	defer server.Close()

	const rules = "User-agent: *\nAllow: /\n"
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequest(method, server.URL+"/robots.txt", nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := server.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if method == http.MethodPost {
				if resp.StatusCode != http.StatusMethodNotAllowed || resp.Header.Get("Allow") != "GET, HEAD" {
					t.Errorf("got status %d, Allow %q", resp.StatusCode, resp.Header.Get("Allow"))
				}
				return
			}
			if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/plain; charset=utf-8" || resp.ContentLength != int64(len(rules)) {
				t.Errorf("got status %d, headers %v", resp.StatusCode, resp.Header)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			want := rules
			if method == http.MethodHead {
				want = ""
			}
			if string(body) != want {
				t.Errorf("body = %q, want %q", body, want)
			}
		})
	}
}
