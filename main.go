package main

import (
	"context"
	"log"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/yurishkuro/opentracing-tutorial/go/lib/tracing"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Missing remote URL")
	}

	// Init tracing
	tracer, closer := tracing.Init("get-web")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	url := parseURL(os.Args[1])

	span := tracer.StartSpan("get-http")
	defer span.Finish()

	ctx := opentracing.ContextWithSpan(context.Background(), span)

	visit(ctx, url)
}
