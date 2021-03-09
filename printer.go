package grpc_loginterceptor

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type loggerWithFields interface {
	WithFields(fields logrus.Fields) *logrus.Entry
}

type printer interface {
	printf(ctx context.Context, timestamp time.Time, info *grpc.UnaryServerInfo, format string, args ...interface{})
}

type logWriterPrinter struct {
	logWriter io.Writer
}

func newLogWriterPrinter(logWriter io.Writer) *logWriterPrinter {
	return &logWriterPrinter{logWriter: logWriter}
}

func (l *logWriterPrinter) printf(ctx context.Context, timestamp time.Time, info *grpc.UnaryServerInfo, format string, args ...interface{}) {
	requestId := ctx.Value(DefaultRequestIDKey).(string)
	fmt.Fprintf(l.writer(), "[gRPC] %v | %15s | %s | %s | %s",
		timestamp.Format("2006/01/02 - 15:04:05"),
		clientIP(ctx),
		requestId,
		info.FullMethod,
		fmt.Sprintf(format, args...),
	)
}

func (l *logWriterPrinter) writer() io.Writer {
	if l.logWriter != nil {
		return l.logWriter
	}
	return os.Stdout
}

type structedLoggerPrinter struct {
	loggerWithFields
}

func newStructedLoggerPrinter(l loggerWithFields) *structedLoggerPrinter {
	return &structedLoggerPrinter{
		loggerWithFields: l,
	}
}

func (s *structedLoggerPrinter) printf(ctx context.Context, timestamp time.Time, info *grpc.UnaryServerInfo, format string, args ...interface{}) {
	requestId := ctx.Value(DefaultRequestIDKey).(string)
	entry := s.loggerWithFields.WithFields(logrus.Fields{
		"Time":      timestamp.Format("2006/01/02 - 15:04:05"),
		"ClientIP":  clientIP(ctx),
		"RequestID": requestId,
		"Method":    info.FullMethod,
	})
	entry.Infof(format, args...)
}

func clientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if ok {
		return p.Addr.String()
	}
	return ""
}
