package server_test

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/sarathsp06/sparrow/server"
)

// Example shows an embedding program serving Sparrow behind its own
// middleware. It uses only the exported API, so it also proves that a module
// outside Sparrow can compile against it.
func Example() {
	ctx := context.Background()

	srv, err := server.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// The embedder's own middleware wraps Sparrow's handler.
	withServedBy := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Served-By", "my-app")
			next.ServeHTTP(w, r)
		})
	}
	httpServer := &http.Server{
		Addr:              srv.Addr(),
		Handler:           withServedBy(srv.Handler()),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = httpServer.ListenAndServe() }()

	// On shutdown: drain, stop accepting requests, then stop Sparrow.
	shutdownCtx, cancel := context.WithTimeout(ctx, srv.ShutdownTimeout())
	defer cancel()
	srv.Drain()
	_ = httpServer.Shutdown(shutdownCtx)
	_ = srv.Shutdown(shutdownCtx)
}
