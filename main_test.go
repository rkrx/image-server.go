package main

import (
	"bytes"
	"fmt"
	"image"
	"image-server/src/server"
	"image-server/src/tools"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMissingImageReturnsNotFound(t *testing.T) {
	for _, path := range []string{"/missing.png", "/missing-100%25.png", "/missing%3Fpart.png"} {
		t.Run(path, func(t *testing.T) {
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.RequestURI != path {
					t.Errorf("origin received %q, want %q", r.RequestURI, path)
				}
				http.NotFound(w, r)
			}))
			defer origin.Close()

			router := newRouter(server.NewBackend(tools.Config{
				ImageCacheDir: t.TempDir(),
				Server:        tools.ServerType{OriginalBackend: origin.URL},
			}))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusNotFound {
				t.Errorf("got HTTP %d, want 404", response.Code)
			}
		})
	}
}

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

func TestScriptProfileRendersDefaultAndNamedSource(t *testing.T) {
	origin := func(path string, pixel color.NRGBA) string {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != path {
				http.NotFound(w, r)
				return
			}
			img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
			img.SetNRGBA(0, 0, pixel)
			if err := png.Encode(w, img); err != nil {
				t.Error(err)
			}
		}))
		t.Cleanup(server.Close)
		return server.URL
	}
	red, blue := color.NRGBA{R: 255, A: 255}, color.NRGBA{B: 255, A: 255}
	dir := t.TempDir()
	script, err := os.ReadFile("doc/examples/side-by-side.js")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "side-by-side.js"), script, 0600))
	config := fmt.Sprintf(`secret="test"
imageCacheDir=%q
imageOriginalDir=%q
[server]
originalBackend=%q
[[profile]]
name="side-by-side"
script="side-by-side.js"
[[profile]]
name="other"
originalBackend=%q
`, dir, dir, origin("/main item.png", red), origin("/second item.png", blue))
	filename := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(filename, []byte(config), 0600))
	conf, err := tools.LoadConfig(filename)
	require.NoError(t, err)
	router := newRouter(server.NewBackend(conf))
	target := "/main%20item.png?profile=side-by-side&right=%2Fsecond%20item.png&rightProfile=other&type=png"
	request := func(target string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
		return w
	}
	require.Equal(t, http.StatusForbidden, request(target).Code)
	sign := func(uri string) string { return uri + "&hash=" + tools.Md5(uri + conf.Secret)[:8] }
	signed := sign(target)
	require.Equal(t, http.StatusForbidden, request(strings.Replace(signed, "second", "third", 1)).Code)
	require.Equal(t, http.StatusForbidden, request(strings.Replace(signed, "type=png", "type=jpeg", 1)).Code)
	for _, extra := range []string{"&right=x", "&canvas=white", "&put=[]", "&putb=W10"} {
		require.Equal(t, http.StatusBadRequest, request(sign(target+extra)).Code)
	}
	w := request(signed)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	img, err := png.Decode(bytes.NewReader(w.Body.Bytes()))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 800, 400), img.Bounds())
	require.Equal(t, red, color.NRGBAModel.Convert(img.At(200, 200)))
	require.Equal(t, blue, color.NRGBAModel.Convert(img.At(600, 200)))
	require.Equal(t, http.StatusNotFound, request(sign(strings.Replace(target, "second", "missing", 1))).Code)
	require.Equal(t, w.Body.Bytes(), request(signed).Body.Bytes())

	sameOrigin := "/main%20item.png?profile=side-by-side&right=%2Fmain%20item.png&type=png"
	w = request(sign(sameOrigin))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	img, err = png.Decode(bytes.NewReader(w.Body.Bytes()))
	require.NoError(t, err)
	require.Equal(t, red, color.NRGBAModel.Convert(img.At(600, 200)))
	require.Equal(t, http.StatusBadRequest, request(sign("/main.png?profile=side-by-side")).Code)
}
