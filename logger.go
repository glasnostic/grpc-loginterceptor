package grpc_loginterceptor

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/rs/xid"
	"google.golang.org/grpc"
)

var (
	DefaultRequestIDPrefix = "req-"
	DefaultRequestIDKey    = "gl-request-id"
	DefaultRequestBeginKey = "gl-request-begin"
)

type accessLoggerInterceptor struct {
	printer
}

func NewAccessLoggerInterceptor(logWriter io.Writer) grpc.UnaryServerInterceptor {
	return (&accessLoggerInterceptor{printer: newLogWriterPrinter(logWriter)}).Intercept
}

func NewAccessLoggerInterceptorWithLogger(l loggerWithFields) grpc.UnaryServerInterceptor {
	return (&accessLoggerInterceptor{printer: newStructedLoggerPrinter(l)}).Intercept
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
