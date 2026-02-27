package http

import (
	json "encoding/json"
	nethttp "net/http"

	"github.com/nojyerac/go-lib/transport/http"
	"github.com/nojyerac/semaphore/data"
)

func RegisterRoutes(src data.Source, srv http.Server) {
	srv.HandleFunc("GET /flags", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		flags, err := src.GetFlags(r.Context())
		if err != nil {
			nethttp.Error(w, "failed to get flags", nethttp.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(flags); err != nil {
			nethttp.Error(w, "failed to encode flags", nethttp.StatusInternalServerError)
			return
		}
	})
}
