package main

import (
	"context"
	"log"
	observerv1 "test-client/gen/observer/v1"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to observer: %v", err)
	}
	defer conn.Close()

	client := observerv1.NewObserverIngestServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.IngestHTTPExchange(ctx, &observerv1.IngestHTTPExchangeRequest{
		ServiceName: "test-api",
		Environment: "dev",

		Method: "GET",
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   "/health",
		Query:  "",

		RequestHeaders: map[string]*observerv1.HeaderValues{
			"Content-Type": {
				Values: []string{"application/json"},
			},
		},
		RequestBody: []byte(``),

		StatusCode: 200,
		ResponseHeaders: map[string]*observerv1.HeaderValues{
			"Content-Type": {
				Values: []string{"application/json"},
			},
		},
		ResponseBody: []byte(`{"status":"ok"}`),

		DurationMs: 12,
		TraceId:    "example-trace-id",
	})
	if err != nil {
		log.Fatalf("failed to ingest HTTP exchange: %v", err)
	}

	log.Printf("observer accepted=%t event_id=%s", resp.GetAccepted(), resp.GetEventId())
}
