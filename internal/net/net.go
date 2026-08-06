// Package net provides the AI streaming/proxy HTTP client the frontend's
// `proxyFetch` uses to talk to OpenAI-compatible providers. It also
// implements the simple LM Studio / Ollama health-check used to surface
// "local model reachable" UI state.
package net

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"terax/internal/types"
)

// PingArgs is a thin alias used by the bound method.
type PingArgs = types.LmPingArgs

// AiHTTPRequestArgs is the request body for AiHTTPRequest.
type AiHTTPRequestArgs = types.AiHttpRequestArgs

// AiHTTPStreamArgs is the request body for AiHTTPStream.
type AiHTTPStreamArgs = types.AiHttpStreamArgs

// LmPing pings a local model endpoint with a short timeout. Returns the
// HTTP status code (so the frontend can render Connected / Failed badges).
func LmPing(ctx context.Context, args PingArgs) (int, error) {
	if args.BaseURL == "" {
		return 0, errors.New("empty url")
	}
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", args.BaseURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// AiHTTPRequest performs a single non-streaming HTTP request.
func AiHTTPRequest(ctx context.Context, args types.AiHttpRequestArgs) (map[string]interface{}, error) {
	client := newClient(args.AllowPrivateNetwork)
	req, err := buildRequest(ctx, args)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10 MB cap
	if err != nil {
		return nil, err
	}
	headers := map[string]string{}
	for k, vs := range resp.Header {
		if len(vs) > 0 {
			headers[k] = vs[0]
		}
	}
	return map[string]interface{}{
		"status":  resp.StatusCode,
		"headers": headers,
		"body":    string(body),
	}, nil
}

// AiHTTPStream streams the response body back to the frontend in chunks.
func AiHTTPStream(ctx context.Context, args types.AiHttpStreamArgs) error {
	client := newClient(args.AllowPrivateNetwork)
	req, err := buildRequest(ctx, types.AiHttpRequestArgs{
		URL:                 args.URL,
		Method:              args.Method,
		Headers:             args.Headers,
		Body:                args.Body,
		AllowPrivateNetwork: args.AllowPrivateNetwork,
	})
	if err != nil {
		wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{Kind: "error", Message: err.Error()})
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{Kind: "error", Message: err.Error()})
		return err
	}
	defer resp.Body.Close()

	headers := map[string]string{}
	for k, vs := range resp.Header {
		if len(vs) > 0 {
			headers[k] = vs[0]
		}
	}
	wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{
		Kind:    "headers",
		Status:  resp.StatusCode,
		Headers: headers,
	})

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1 MB cap on error body
		wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{
			Kind:    "error",
			Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		})
		wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{Kind: "end"})
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	buf := make([]byte, 8*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{Kind: "chunk", Bytes: chunk})
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{Kind: "end"})
				return nil
			}
			wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{Kind: "error", Message: err.Error()})
			wailsruntime.EventsEmit(ctx, args.OnEventEvent, types.AiStreamEvent{Kind: "end"})
			return err
		}
	}
}

func buildRequest(ctx context.Context, args types.AiHttpRequestArgs) (*http.Request, error) {
	method := strings.ToUpper(args.Method)
	if method == "" {
		method = "GET"
	}
	var body io.Reader
	if len(args.Body) > 0 {
		body = bytesReader(args.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, args.URL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range args.Headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

// newClient returns an http.Client with a custom dialer that can either
// allow or reject private-network hosts (SSRF mitigation).
func newClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if !allowPrivate && (isPrivateHost(host) || isLocalHost(host)) {
				return nil, errors.New("private network access denied")
			}
			return dialer.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{Transport: transport, Timeout: 0}
}

func isPrivateHost(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLoopback()
}

func isLocalHost(host string) bool {
	return host == "localhost"
}

// bytesReader wraps a byte slice as io.Reader. We avoid bytes.NewReader so
// the helper can be inlined where convenient.
type bytesReaderImpl struct {
	b []byte
	i int
}

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func bytesReader(b []byte) io.Reader { return &bytesReaderImpl{b: b} }

// JSONPatch keeps the dependency graph honest.
var _ = json.Marshal