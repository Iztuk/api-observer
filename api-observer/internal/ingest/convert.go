package ingest

import (
	"fmt"
	"net/http"
	"net/url"

	"api-observer/internal/audit"
	ingestv1 "api-observer/proto/ingest/v1"
)

func toAuditJob(pb *ingestv1.Job) (audit.Job, error) {
	if pb == nil || pb.Request == nil {
		return audit.Job{}, fmt.Errorf("missing request")
	}

	req := pb.Request

	if req.Metadata == nil {
		return audit.Job{}, fmt.Errorf("missing request metadata")
	}

	if req.Metadata.RequestId == "" {
		return audit.Job{}, fmt.Errorf("missing request ID")
	}

	u, err := url.Parse(req.Url)
	if err != nil {
		return audit.Job{}, fmt.Errorf("invalid request URL: %w", err)
	}

	if req.Url == "" {
		return audit.Job{}, fmt.Errorf("missing request URL")
	}

	job := audit.Job{
		Request: &audit.RequestJob{
			Method:        req.Method,
			URL:           u,
			Header:        toHTTPHeaders(req.Headers),
			Body:          req.Body,
			ContentLength: req.ContentLength,

			Metadata: audit.Metadata{
				RequestID: req.Metadata.RequestId,
				Source:    req.Metadata.Source,
			},
		},
	}

	switch pb.Type {
	case ingestv1.JobType_JOB_TYPE_REQUEST:
		if pb.Response != nil {
			return audit.Job{}, fmt.Errorf(
				"request job must not contain a response",
			)
		}

		job.Type = audit.JobTypeRequest

	case ingestv1.JobType_JOB_TYPE_RESPONSE:
		if pb.Response == nil {
			return audit.Job{}, fmt.Errorf("missing response")
		}

		resp := pb.Response

		if resp.Metadata == nil {
			return audit.Job{}, fmt.Errorf("missing response metadata")
		}

		if resp.Metadata.RequestId != req.Metadata.RequestId {
			return audit.Job{}, fmt.Errorf(
				"request and response IDs do not match",
			)
		}

		job.Type = audit.JobTypeResponse

		job.Response = &audit.ResponseJob{
			StatusCode:    int(resp.StatusCode),
			Header:        toHTTPHeaders(resp.Headers),
			Body:          resp.Body,
			ContentLength: resp.ContentLength,

			Metadata: audit.Metadata{
				RequestID: resp.Metadata.RequestId,
				Source:    resp.Metadata.Source,
			},
		}

	default:
		return audit.Job{}, fmt.Errorf(
			"unsupported job type %v",
			pb.Type,
		)
	}

	return job, nil
}

func toHTTPHeaders(
	headers map[string]*ingestv1.HeaderValues,
) http.Header {
	result := make(http.Header)

	for name, values := range headers {
		if values == nil {
			continue
		}

		result[http.CanonicalHeaderKey(name)] =
			append([]string(nil), values.Values...)
	}

	return result
}
