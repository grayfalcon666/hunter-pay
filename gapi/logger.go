package gapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GrpcLogger(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	startTime := time.Now()

	result, err := handler(ctx, req)

	duration := time.Since(startTime)
	statusCode := codes.Unknown
	if st, ok := status.FromError(err); ok {
		statusCode = st.Code()
	}

	logger := slog.With(
		slog.String("protocol", "grpc"),
		slog.String("method", info.FullMethod),
		slog.Int("status_code", int(statusCode)),
		slog.String("status_text", statusCode.String()),
		slog.Duration("duration", duration),
	)

	if err != nil {
		logger.Error("received a gRPC request", slog.String("error", err.Error()))
	} else {
		logger.Info("received a gRPC request")
	}

	return result, err
}

type responseBodyWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseBodyWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func HttpLogger(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		rec := &responseBodyWriter{ResponseWriter: w, statusCode: http.StatusOK}

		nextHandler.ServeHTTP(rec, r)
		duration := time.Since(startTime)
		logger := slog.With(
			slog.String("protocol", "http"),
			slog.String("method", r.Method),
			slog.String("path", r.RequestURI),
			slog.Int("status_code", int(rec.statusCode)),
			slog.String("status_text", strconv.Itoa(rec.statusCode)),
			slog.Duration("duration", duration),
		)

		if rec.statusCode != http.StatusOK {
			logger.Error("received a HTTP request")
		} else {
			logger.Info("received a HTTP request")
		}
	})
}
