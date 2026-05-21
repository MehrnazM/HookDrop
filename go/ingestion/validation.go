package main

import (
	"io"
	"net/http"

	util "github.com/mehrnazm/hookdrop/go/util"
)

const MaxPayloadSize = 1 * 1024 * 1024 // 1MB

// ValidateRequest checks Content-Length and installs a size cap on the body reader.
func ValidateRequest(w http.ResponseWriter, r *http.Request) *util.HTTPError {
	contentLength := r.ContentLength
	if contentLength > MaxPayloadSize {
		return util.PayloadTooLarge("Content-Length exceeds 1MB")
	}

	// Limit the read to MaxPayloadSize to prevent reading huge bodies.
	// ResponseWriter is required so MaxBytesReader can write a 413 if the limit is hit.
	r.Body = http.MaxBytesReader(w, r.Body, MaxPayloadSize)

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
