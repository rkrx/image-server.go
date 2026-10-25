package server

import (
	"net/http"
	"strconv"
	"strings"

	"image-server/src/tools"
)

// applyCORS writes CORS response headers when configured. It returns true if the
// request is a preflight that has already been fully answered and the caller
// should stop processing.
func applyCORS(w http.ResponseWriter, r *http.Request, cfg tools.CORSConfig) bool {
	if !cfg.Enabled {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusForbidden)
			return true
		}
		return false
	}
	allowed, allowAll := matchCORSOrigin(cfg.AllowedOrigins, origin)
	if !allowed {
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusForbidden)
			return true
		}
		return false
	}

	if allowAll && !cfg.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		addVary(w, "Origin")
	}
	if cfg.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if len(cfg.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
	}

	if r.Method != http.MethodOptions || r.Header.Get("Access-Control-Request-Method") == "" {
		return false
	}

	methods := cfg.AllowedMethods
	if len(methods) == 0 {
		methods = []string{http.MethodGet, http.MethodHead}
	}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))

	if len(cfg.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
	} else if requested := r.Header.Get("Access-Control-Request-Headers"); requested != "" {
		w.Header().Set("Access-Control-Allow-Headers", requested)
		addVary(w, "Access-Control-Request-Headers")
	}
	addVary(w, "Access-Control-Request-Method")

	if cfg.MaxAgeInSeconds > 0 {
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAgeInSeconds))
	}

	w.WriteHeader(http.StatusNoContent)
	return true
}

func matchCORSOrigin(allowedOrigins []string, origin string) (allowed, wildcard bool) {
	for _, candidate := range allowedOrigins {
		if candidate == "*" {
			return true, true
		}
		if strings.EqualFold(candidate, origin) {
			return true, false
		}
	}
	return false, false
}

func addVary(w http.ResponseWriter, value string) {
	for _, existing := range w.Header().Values("Vary") {
		if strings.EqualFold(existing, value) {
			return
		}
	}
	w.Header().Add("Vary", value)
}
