package grpc_loginterceptor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

var (
	DefaultRequestIDPrefix = "req-"
	DefaultRequestIDKey    = "gl-request-id"
	DefaultRequestBeginKey = "gl-request-begin"
)

type accessLoggerInterceptor struct {
	logWriter io.Writer
}

func NewAccessLoggerInterceptor(logWriter io.Writer) grpc.UnaryServerInterceptor {
	return (&accessLoggerInterceptor{logWriter: logWriter}).Intercept
}

func (a *accessLoggerInterceptor) Intercept(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx = a.beforeHandle(ctx, info, req)
	resp, err := handler(ctx, req)
	a.afterHandle(ctx, info, err)
	return resp, err
}

func (a *accessLoggerInterceptor) beforeHandle(ctx context.Context, info *grpc.UnaryServerInfo, req interface{}) context.Context {
	requestBegin := time.Now()
	ctx = context.WithValue(ctx, DefaultRequestBeginKey, requestBegin)

	requestId := xid.NewWithTime(requestBegin)
	ctx = context.WithValue(ctx, DefaultRequestIDKey, DefaultRequestIDPrefix+requestId.String())

	b, err := json.Marshal(req)
	if err != nil {
		a.printf(ctx, requestBegin, info, "failed to marshal the request: %s\n", err)
	} else {
		a.printf(ctx, requestBegin, info, "%s\n", b)
	}

	return ctx
}

func (a *accessLoggerInterceptor) afterHandle(ctx context.Context, info *grpc.UnaryServerInfo, err error) {
	requestEnd := time.Now()
	requestBegin := ctx.Value(DefaultRequestBeginKey).(time.Time)

	var errMessage string
	if err != nil {
		errMessage = err.Error()
	}

	a.printf(ctx, requestEnd, info, "%13v\n%s", requestEnd.Sub(requestBegin), errMessage)
}

func (a *accessLoggerInterceptor) printf(ctx context.Context, timestamp time.Time, info *grpc.UnaryServerInfo, format string, args ...interface{}) {
	requestId := ctx.Value(DefaultRequestIDKey).(string)
	fmt.Fprintf(a.writer(), "[gRPC] %v | %15s | %s | %s | %s",
		timestamp.Format("2006/01/02 - 15:04:05"),
		clientIP(ctx),
		requestId,
		info.FullMethod,
		fmt.Sprintf(format, args...),
	)
}

func (a *accessLoggerInterceptor) writer() io.Writer {
	if a.logWriter != nil {
		return a.logWriter
	}
	return os.Stdout
}

func clientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if ok {
		return p.Addr.String()
	}
	return ""
}
