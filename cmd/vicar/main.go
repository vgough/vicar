package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

var ErrHTTP2Required = fmt.Errorf("HTTP2 required")

var cli struct {
	Listen string         `flag:"listen" default:":8080" help:"listen address for incoming requests"`
	In     map[string]int `flag:"in" help:"Mapping of input service to local port"`
	Out    map[int]string `flag:"out" type:":*url.URL" help:"Mapping of local port to output URL"`
}

func main() {
	// Debug logger going to discard.
	ll := log.With().Logger()

	kctx := kong.Parse(&cli, kong.Bind(ll))
	if len(cli.In) == 0 && len(cli.Out) == 0 {
		kctx.Errorf("at least one of --in or --out must be specified")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eg, ctx := errgroup.WithContext(ctx)

	if len(cli.In) > 0 {
		h2s := &http2.Server{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello, %v, http: %v", r.URL.Path, r.TLS == nil)
		})

		server := &http.Server{
			Addr:    cli.Listen,
			Handler: h2c.NewHandler(handler, h2s),
		}

		server.ListenAndServe()
		eg.Go(func() error {
			return server.ListenAndServe()
		})
		go func() {
			<-ctx.Done()
			server.Close()
		}()

		for svc, port := range cli.In {
			log.Info().Str("service", svc).Int("port", port).Msg("registering service listener")
		}
	}

	err := eg.Wait()
	log.Info().Err(err).Msg("shutting down")
}

func incomingRequest(w http.ResponseWriter, r *http.Request) {
	log.Info().Str("method", r.Method).Str("url", r.URL.String()).Msg("incoming request")
	if !r.ProtoAtLeast(2, 0) {
		w.WriteHeader(http.StatusHTTPVersionNotSupported)
		w.Write([]byte("HTTP2 required"))
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}

	eg, ctx := errgroup.WithContext(r.Context())
	eg.Go(func() error {
	})
}
