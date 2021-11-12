package logging

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/felixge/httpsnoop"
	"go.uber.org/zap"
)

type loggerKeyType int

const (
	loggerKey loggerKeyType = iota
)

func NewMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctxID := fmt.Sprintf("%p", ctx)
			ctxLogger := logger.WithOptions(zap.Fields(zap.String("ctx", ctxID)))
			ctx = WithLogger(ctx, ctxLogger)
			var mtx httpsnoop.Metrics
			defer func(logger *zap.Logger) {
				err := recover()
				if err == nil {
					logger.Info(
						"DONE",
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("referer", r.Header.Get("Referer")),
						zap.Int("status", mtx.Code),
						zap.Int64("size", mtx.Written),
						zap.Duration("elapsed", mtx.Duration),
					)
				} else {
					f := zap.Stack("trace")
					logger.Error(
						"DONE",
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("referer", r.Header.Get("Referer")),
						zap.String("error", fmt.Sprint(err)),
						f,
					)
					fmt.Fprintf(os.Stderr, "Context: %s\nError: %s\nStacktrace:\n%s", ctxID, err, f.String)
					http.Error(w, "500 Internal Server Error\n"+ctxID, http.StatusInternalServerError)
				}
			}(ctxLogger.WithOptions(zap.WithCaller(false)))
			mtx = httpsnoop.CaptureMetrics(next, w, r.WithContext(ctx))
		})
	}
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func Logger(ctx context.Context) *zap.Logger {
	return ctx.Value(loggerKey).(*zap.Logger)
}
