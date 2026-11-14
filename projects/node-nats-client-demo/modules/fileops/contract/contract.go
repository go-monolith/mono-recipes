// Package contract is the published port of the fileops module: the DTOs of
// its services and the typed contracts they are registered, called and sent
// through. It is the only package of the fileops module that other code may
// import.
//
// It imports only the standard library and mono/pkg/helper. The subjects
// services.fileops.save and services.fileops.archive are what the Node.js
// clients in client/ and demo.js use, so module and service names must not
// change.
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// FileSaveRequest is a request to save a JSON document to the bucket.
type FileSaveRequest struct {
	Filename string         `json:"filename"`
	Content  map[string]any `json:"content"`
}

// FileSaveResponse is the result of a save. A failed save is reported in
// Error, not as a service error.
type FileSaveResponse struct {
	FileID   string    `json:"file_id"`
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	Digest   string    `json:"digest"`
	SavedAt  time.Time `json:"saved_at"`
	Error    string    `json:"error,omitempty"`
}

// FileArchiveRequest asks to archive a saved file as a ZIP.
type FileArchiveRequest struct {
	FileID string `json:"file_id"`
}

// Save is the request-reply service on services.fileops.save.
var Save = helper.RequestReplyService[FileSaveRequest, FileSaveResponse]("fileops", "save")

// Archive is the fire-and-forget queue group service on
// services.fileops.archive.
var Archive = helper.QueueGroupService[FileArchiveRequest]("fileops", "archive")
