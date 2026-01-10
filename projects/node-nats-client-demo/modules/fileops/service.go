package fileops

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-monolith/mono"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
	"github.com/google/uuid"
)

// handleFileSave handles the file.save RequestReplyService.
func (m *Module) handleFileSave(ctx context.Context, req FileSaveRequest, _ *mono.Msg) (FileSaveResponse, error) {
	// Validate request
	if req.Filename == "" {
		return FileSaveResponse{Error: "filename is required"}, nil
	}
	if req.Content == nil {
		return FileSaveResponse{Error: "content is required"}, nil
	}

	// Generate unique file ID
	fileID := uuid.New().String()

	// Convert content to JSON bytes
	contentBytes, err := json.Marshal(req.Content)
	if err != nil {
		return FileSaveResponse{Error: fmt.Sprintf("failed to serialize content: %v", err)}, nil
	}

	// Create storage key: {uuid}/{filename}.json
	storageKey := fmt.Sprintf("%s/%s", fileID, req.Filename)

	// Save to bucket with metadata
	objInfo, err := m.bucket.Put(ctx, storageKey, contentBytes,
		fsjetstream.WithHeaders(map[string]string{
			"Content-Type": "application/json",
			"File-ID":      fileID,
			"Saved-At":     time.Now().Format(time.RFC3339),
		}),
	)
	if err != nil {
		return FileSaveResponse{Error: fmt.Sprintf("failed to save file: %v", err)}, nil
	}

	// Build success response
	return FileSaveResponse{
		FileID:   fileID,
		Filename: req.Filename,
		Size:     int64(objInfo.Size),
		Digest:   objInfo.Digest,
		SavedAt:  objInfo.ModTime,
	}, nil
}

// handleFileArchive handles the file.archive QueueGroupService.
func (m *Module) handleFileArchive(ctx context.Context, msg *mono.Msg) error {
	var req FileArchiveRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		m.logger.Error("Failed to unmarshal archive request", "error", err)
		return nil // Fire-and-forget: don't propagate error
	}

	// Validate file ID
	if req.FileID == "" {
		m.logger.Warn("Archive request missing file_id")
		return nil
	}

	// Find the file by ID prefix
	files, err := m.bucket.List(fsjetstream.WithPrefix(req.FileID + "/"))
	if err != nil {
		m.logger.Error("Failed to list files for archive", "file_id", req.FileID, "error", err)
		return nil
	}

	if len(files) == 0 {
		m.logger.Warn("File not found for archive", "file_id", req.FileID)
		return nil
	}

	// Get the first file (should be the JSON file)
	jsonFile := files[0]

	// Skip if it's already a ZIP file
	if strings.HasSuffix(jsonFile.Name, ".zip") {
		m.logger.Info("File is already archived", "file_id", req.FileID, "name", jsonFile.Name)
		return nil
	}

	// Get the file data
	fileData, err := m.bucket.Get(jsonFile.Name)
	if err != nil {
		m.logger.Error("Failed to retrieve file for archive", "file_id", req.FileID, "error", err)
		return nil
	}

	// Create ZIP archive in memory
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	// Extract original filename from storage key (after fileID/)
	originalFilename := jsonFile.Name
	if len(req.FileID)+1 < len(jsonFile.Name) {
		originalFilename = jsonFile.Name[len(req.FileID)+1:]
	}

	// Add file to ZIP
	writer, err := zipWriter.Create(originalFilename)
	if err != nil {
		m.logger.Error("Failed to create ZIP entry", "error", err)
		return nil
	}

	if _, err := writer.Write(fileData); err != nil {
		m.logger.Error("Failed to write file to ZIP", "error", err)
		return nil
	}

	if err := zipWriter.Close(); err != nil {
		m.logger.Error("Failed to close ZIP writer", "error", err)
		return nil
	}

	// Construct ZIP filename (replace .json with .zip)
	zipFilename := originalFilename
	if strings.HasSuffix(zipFilename, ".json") {
		zipFilename = strings.TrimSuffix(zipFilename, ".json") + ".zip"
	} else {
		zipFilename = zipFilename + ".zip"
	}

	// Save ZIP file to bucket
	zipKey := fmt.Sprintf("%s/%s", req.FileID, zipFilename)
	_, err = m.bucket.Put(ctx, zipKey, zipBuf.Bytes(),
		fsjetstream.WithHeaders(map[string]string{
			"Content-Type":     "application/zip",
			"Original-File-ID": req.FileID,
			"Archived-At":      time.Now().Format(time.RFC3339),
		}),
	)
	if err != nil {
		m.logger.Error("Failed to save ZIP file", "error", err)
		return nil
	}

	// Delete original JSON file
	if err := m.bucket.Delete(jsonFile.Name); err != nil {
		m.logger.Error("Failed to delete original file after archive", "file", jsonFile.Name, "error", err)
		return nil
	}

	m.logger.Info("File archived successfully",
		"file_id", req.FileID,
		"original", originalFilename,
		"archive", zipFilename)

	return nil
}

