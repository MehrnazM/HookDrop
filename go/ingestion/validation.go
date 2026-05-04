package main

import (
	"io"
	"net/http"

	util "github.com/mehrnazm/webhookx/go/util"
)

const MaxPayloadSize = 1 * 1024 * 1024 // 1MB

// ValidateRequest checks Content-Length and content-type
func ValidateRequest(r *http.Request) *util.HTTPError {
	contentLength := r.ContentLength
	if contentLength > MaxPayloadSize {
		return util.PayloadTooLarge("Content-Length exceeds 1MB")
	}

	// Limit the read to MaxPayloadSize to prevent reading huge bodies
	r.Body = http.MaxBytesReader(nil, r.Body, MaxPayloadSize)

	return nil
}

// ReadRequestBody safely reads the request body with size limit
func ReadRequestBody(r *http.Request) ([]byte, *util.HTTPError) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			return nil, util.PayloadTooLarge("Request body exceeds 1MB")
		}
		return nil, util.ValidationError("Failed to read request body")
	}
	return body, nil
}
