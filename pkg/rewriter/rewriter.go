package rewriter

import (
	"net/http"
	"regexp"
	"strings"
)

func New(rules RewriteRule) *Rewriter {
	return &Rewriter{rules: rules}
}

func (rw *Rewriter) RewritePath(path string) string {
	if rw.rules.Value == "" || rw.rules.Type == "" {
		return path
	}

	switch rw.rules.Type {
	case PrefixRewrite:
		path = strings.Replace(path, rw.rules.Value, rw.rules.ReplaceVal, 1)
	case RegexRewrite:
		re := regexp.MustCompile(rw.rules.Value)
		if rw.rules.ReplaceVal != "" {
			path = re.ReplaceAllString(path, rw.rules.ReplaceVal)
		} else {
			path = re.ReplaceAllString(path, "/$1")
		}
	}

	return path
}

func (rw *Rewriter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = rw.RewritePath(r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
