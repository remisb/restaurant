package mid

import (
	"context"
	"github.com/remisb/restaurant/internal/platform/web"
	"go.opencensus.io/trace"
	"log"
	"net/http"
	"time"
)

func Logger(log *log.Logger) web.Middleware {
	f := func(before web.Handler) web.Handler {

		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
			ctx, span := trace.StartSpan(ctx, "internal.mid.Logger")
			defer span.End()

			// If the context is missing this value, request the service
			// to be shutdown gracefully.
			v, ok := ctx.Value(web.KeyValues).(*web.Values)
			if !ok {
				return web.NewShutdownError("web value missing from context")
			}
			err := before(ctx, w, r, params)

			log.Printf("%s : (%d) : %s %s -> %s (%s)",
			v.TraceID, v.StatusCode,
			r.Method, r.URL.Path,
			r.RemoteAddr, time.Since(v.Now),
			)
			return err
		}
		return h
	}

	return f
}
