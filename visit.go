package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
)

func visit(ctx context.Context, url *url.URL) {

	fmt.Printf("Visiting %v\n", url.String())

	span, _ := opentracing.StartSpanFromContext(ctx, url.String())
	span.SetTag("url", url.String())

	req := newRequest("GET", url, "")

	var dnsSpan opentracing.Span
	var tcpConnectionSpan opentracing.Span
	var tlsHandshakeSpan opentracing.Span
	var serverProcessingSpan opentracing.Span
	var contentTransferSpan opentracing.Span

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			dnsSpan = span.Tracer().StartSpan(
				"dns-lookup",
				opentracing.ChildOf(span.Context()),
			)
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			dnsSpan.LogFields(
				otlog.String("event", "namelookup"),
			)
			dnsSpan.Finish()
		},
		ConnectStart: func(_, _ string) {
			tcpConnectionSpan = span.Tracer().StartSpan(
				"tcp-connection",
				opentracing.ChildOf(span.Context()),
			)
		},
		ConnectDone: func(net, addr string, err error) {
			tcpConnectionSpan.SetTag("url", url.String())
			tcpConnectionSpan.SetTag("scheme", url.Scheme)
			tcpConnectionSpan.SetTag("host", url.Host)
			tcpConnectionSpan.SetTag("raw-query", url.Path)

			tcpConnectionSpan.LogFields(
				otlog.String("event", "connect"),
			)
			tcpConnectionSpan.Finish()

			// TLS handshake is starting after connect
			tlsHandshakeSpan = span.Tracer().StartSpan(
				"tls-handshake",
				opentracing.ChildOf(span.Context()),
			)

		},
		GotConn: func(_ httptrace.GotConnInfo) {
			// TLS handshake is done

			tlsHandshakeSpan.LogFields(
				otlog.String("event", "pretransfer"),
			)
			tlsHandshakeSpan.Finish()

			serverProcessingSpan = span.Tracer().StartSpan(
				"server-processing",
				opentracing.ChildOf(span.Context()),
			)
		},
		GotFirstResponseByte: func() {
			serverProcessingSpan.Finish()
			contentTransferSpan = span.Tracer().StartSpan(
				"content-transfer",
				opentracing.ChildOf(span.Context()),
			)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// always refuse to follow redirects, visit does that
			// manually if required.
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to read response: %v", err)
	}

	span.SetTag("response-code", resp.Status)

	defer resp.Body.Close()

	contentTransferSpan.Finish()
	span.Finish()

	if resp.StatusCode != 200 {
		fmt.Printf("Received status code %v for %v, exiting\n", resp.Status, url.String())
		return
	}

	parseHTML(ctx, url, resp.Body)
}
