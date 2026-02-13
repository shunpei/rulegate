package http

import (
	"net/http"
)

// NewServeMux wires up all routes and middleware.
func NewServeMux(h *Handler, rateLimiter *IPRateLimiter, allowOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("POST /ask", h.Ask)

	// Stack middleware: outermost first.
	var handler http.Handler = mux
	handler = rateLimiter.Middleware(handler)
	handler = Logging(handler)
	handler = CORS(allowOrigin)(handler)
	handler = RequestID(handler)
	handler = Recovery(handler)

	return handler
}
