package middleware

import (
	"bytes"
	"context"
	"fmt"
	gootel "github.com/erajayatech/go-opentelemetry/v2"
	"github.com/gin-gonic/gin"
	"go-upload-chunk/server/internal/utils"
	ioOtel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

type CustomWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func NewCustomWriter(rsp gin.ResponseWriter, body *bytes.Buffer) *CustomWriter {
	return &CustomWriter{rsp, body}
}

func (c *CustomWriter) Write(b []byte) (int, error) {
	c.body.Write(b)
	return c.ResponseWriter.Write(b)
}

func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract trace parent from header to context
		ctx := ioOtel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// create new span
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.Request.RequestURI)
		ctx, span := gootel.NewSpan(ctx, spanName, "")
		defer span.End()

		traceID := span.SpanContext().TraceID()
		c.Set("traceID", traceID)
		ctx = context.WithValue(ctx, "traceID", traceID)

		// get buffer from Pool
		buf := utils.ByteBufferPool.Get().(*bytes.Buffer)
		defer func() {
			buf.Reset()
			utils.ByteBufferPool.Put(buf)
		}()

		customWriter := NewCustomWriter(c.Writer, buf)
		c.Writer = customWriter

		// continue to next handler
		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if customWriter.body != nil {
			span.SetAttributes(attribute.String("http.response.body", customWriter.body.String()))
		}
	}
}
