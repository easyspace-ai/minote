// Package docreaderclient calls the WeKnora docreader gRPC service (docker compose: docreader:50051).
package docreaderclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/easyspace-ai/minote/pkg/docreaderpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultDialTimeout = 15 * time.Second
	defaultRPCTimeout  = 10 * time.Minute
	defaultMaxMsgBytes = 55 << 20 // > 50 MiB default docreader limit
)

// ReadMarkdown returns markdown from docreader when DOCREADER_ADDR is set and the call succeeds.
// If DOCREADER_ADDR is empty, returns ("", nil) so callers can fall back to other converters.
func ReadMarkdown(ctx context.Context, fileName, fileType string, fileContent []byte) (string, error) {
	addr := strings.TrimSpace(os.Getenv("DOCREADER_ADDR"))
	if addr == "" {
		return "", nil
	}
	transport := strings.ToLower(strings.TrimSpace(os.Getenv("DOCREADER_TRANSPORT")))
	if transport == "" {
		transport = "grpc"
	}
	if transport != "grpc" {
		return "", fmt.Errorf("docreader: unsupported DOCREADER_TRANSPORT %q (only grpc)", transport)
	}
	if len(fileContent) == 0 {
		return "", fmt.Errorf("docreader: empty file content")
	}

	dialTimeout := defaultDialTimeout
	if s := strings.TrimSpace(os.Getenv("DOCREADER_DIAL_TIMEOUT")); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			dialTimeout = d
		}
	}
	rpcTimeout := defaultRPCTimeout
	if s := strings.TrimSpace(os.Getenv("DOCREADER_RPC_TIMEOUT")); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			rpcTimeout = d
		}
	}
	maxMsg := defaultMaxMsgBytes
	if s := strings.TrimSpace(os.Getenv("DOCREADER_MAX_MSG_BYTES")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			maxMsg = n
		}
	}

	dialCtx, cancelDial := context.WithTimeout(ctx, dialTimeout)
	defer cancelDial()

	var opts []grpc.DialOption
	opts = append(opts,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMsg),
			grpc.MaxCallSendMsgSize(maxMsg),
		),
	)
	if strings.EqualFold(strings.TrimSpace(os.Getenv("DOCREADER_TLS")), "true") {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.DialContext(dialCtx, addr, opts...)
	if err != nil {
		return "", fmt.Errorf("docreader dial %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	rpcCtx, cancelRPC := context.WithTimeout(ctx, rpcTimeout)
	defer cancelRPC()

	client := pb.NewDocReaderClient(conn)
	req := &pb.ReadRequest{
		FileContent: fileContent,
		FileName:    fileName,
		FileType:    fileType,
	}
	resp, err := client.Read(rpcCtx, req)
	if err != nil {
		return "", fmt.Errorf("docreader Read: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("docreader: nil response")
	}
	if errMsg := strings.TrimSpace(resp.GetError()); errMsg != "" {
		return "", fmt.Errorf("docreader: %s", errMsg)
	}
	md := strings.TrimSpace(resp.GetMarkdownContent())
	if md == "" {
		return "", fmt.Errorf("docreader: empty markdown_content")
	}
	return md, nil
}

// ConnectionStatus reports whether docreader gRPC is reachable (for health checks). Returns "skipped" if unset.
func ConnectionStatus(ctx context.Context) string {
	addr := strings.TrimSpace(os.Getenv("DOCREADER_ADDR"))
	if addr == "" {
		return "skipped"
	}
	transport := strings.ToLower(strings.TrimSpace(os.Getenv("DOCREADER_TRANSPORT")))
	if transport != "" && transport != "grpc" {
		return "error: unsupported transport"
	}
	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var opts []grpc.DialOption
	if strings.EqualFold(strings.TrimSpace(os.Getenv("DOCREADER_TLS")), "true") {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.DialContext(dialCtx, addr, opts...)
	if err != nil {
		return "error: " + err.Error()
	}
	_ = conn.Close()
	return "ok"
}
