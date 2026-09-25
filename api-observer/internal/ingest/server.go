// Package ingest handles the ingestion of logs.
package ingest

import (
	"errors"
	"io"

	"api-observer/internal/audit"
	"api-observer/internal/nodes"
	ingestv1 "api-observer/proto/ingest/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	ingestv1.UnimplementedIngestServiceServer

	queue            *audit.Queue
	collectorManager *nodes.NodeManager
}

func NewServer(queue *audit.Queue) *Server {
	return &Server{
		queue: queue,
	}
}

func (s *Server) Ingest(
	stream ingestv1.IngestService_IngestServer,
) error {
	for {
		pbJob, err := stream.Recv()

		if errors.Is(err, io.EOF) {
			// Client finished sending jobs.
			return nil
		}

		if err != nil {
			return err
		}

		if pbJob == nil {
			return status.Error(
				codes.InvalidArgument,
				"received nil job",
			)
		}

		if pbJob.Id == "" {
			return status.Error(
				codes.InvalidArgument,
				"missing job ID",
			)
		}

		// Convert protobuf into the internal audit.Job.
		job, err := toAuditJob(pbJob)

		if err != nil {
			if sendErr := stream.Send(
				&ingestv1.IngestAck{
					JobId:     pbJob.Id,
					Status:    ingestv1.IngestStatus_INGEST_STATUS_REJECTED,
					Message:   err.Error(),
					Retryable: false,
				},
			); sendErr != nil {
				return sendErr
			}

			continue
		}

		// Enqueue the job.
		if !s.queue.TryEnqueue(job) {
			if err := stream.Send(
				&ingestv1.IngestAck{
					JobId:     pbJob.Id,
					Status:    ingestv1.IngestStatus_INGEST_STATUS_REJECTED,
					Message:   "queue is full or unavailable",
					Retryable: true,
				},
			); err != nil {
				return err
			}

			continue
		}

		// Acknowledge the successful enqueue.
		if err := stream.Send(
			&ingestv1.IngestAck{
				JobId:   pbJob.Id,
				Status:  ingestv1.IngestStatus_INGEST_STATUS_ACCEPTED,
				Message: "job accepted",
			},
		); err != nil {
			return err
		}
	}
}
