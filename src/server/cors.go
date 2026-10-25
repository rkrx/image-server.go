package server

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"image-server/src/tools"
)

type corsOriginMatcher struct {
	allowAnyOrigin bool
	patterns       []*regexp.Regexp
}

// compileCORSOrigins turns configured allowedOrigins into a matcher. Entries
// may contain `*` as a wildcard that matches a single host, label or port
// segment (any character except `.`, `:` and `/`). A standalone `*` entry
// enables the wildcard origin.
func compileCORSOrigins(allowedOrigins []string) corsOriginMatcher {
	matcher := corsOriginMatcher{}
	for _, origin := range allowedOrigins {
		if origin == "*" {
			matcher.allowAnyOrigin = true
			continue
		}
		if origin == "" {
			continue
		}
		matcher.patterns = append(matcher.patterns, compileCORSPattern(origin))
	}
	return matcher
}

func compileCORSPattern(pattern string) *regexp.Regexp {
	var builder strings.Builder
	builder.WriteString("(?i)^")
	for i, part := range strings.Split(pattern, "*") {
		if i > 0 {
			builder.WriteString(`[^./:]*`)
		}
		builder.WriteString(regexp.QuoteMeta(part))
	}
	builder.WriteString("$")
	return regexp.MustCompile(builder.String())
}

func (m corsOriginMatcher) match(origin string) (allowed, wildcard bool) {
	for _, re := range m.patterns {
		if re.MatchString(origin) {
			return true, false
		}
	}
	if m.allowAnyOrigin {
		return true, true
	}
	return false, false
}

// applyCORS writes CORS response headers when configured. It returns true if the
// request is a preflight that has already been fully answered and the caller
// should stop processing.
func applyCORS(w http.ResponseWriter, r *http.Request, cfg tools.CORSConfig, matcher corsOriginMatcher) bool {
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
	allowed, allowAll := matcher.match(origin)
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

func addVary(w http.ResponseWriter, value string) {
	for _, existing := range w.Header().Values("Vary") {
		if strings.EqualFold(existing, value) {
			return
		}
	}
	w.Header().Add("Vary", value)
}
