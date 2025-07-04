package logging

import (
	"net/http"
	"time"
)

func Access(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		// оборачиваем ResponseWriter, чтобы узнать статус
		ww := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)

		L.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"remote", r.RemoteAddr,
			"dur_ms", time.Since(start).Milliseconds(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}