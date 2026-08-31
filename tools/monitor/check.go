package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	pb "loadmon/protocol"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func checkGRPC(cfg *Config) error {
	addr := cfg.GRPC.address()
	timeout := time.Duration(cfg.GRPC.TimeoutSec) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("gRPC connect %s: %w", addr, err)
	}
	defer conn.Close()

	client := pb.NewLoadmonClient(conn)
	_, err = client.GetHWDatacenterNames(ctx, &pb.Empty{})
	if err != nil {
		return fmt.Errorf("gRPC GetHWDatacenterNames %s: %w", addr, err)
	}
	return nil
}

func checkHTTP(cfg *Config) error {
	timeout := time.Duration(cfg.HTTP.TimeoutSec) * time.Second
	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(cfg.HTTP.URL)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", cfg.HTTP.URL, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != cfg.HTTP.ExpectStatus {
		return fmt.Errorf("HTTP GET %s: статус %d, ожидался %d", cfg.HTTP.URL, resp.StatusCode, cfg.HTTP.ExpectStatus)
	}
	return nil
}
