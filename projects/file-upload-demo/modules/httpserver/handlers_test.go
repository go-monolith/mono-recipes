package httpserver

import (
	"testing"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename    string
		expected    string
		description string
	}{
		// Text files
		{"document.txt", "text/plain", "plain text file"},
		{"page.html", "text/html", "HTML file"},
		{"page.htm", "text/html", "HTM file"},
		{"styles.css", "text/css", "CSS file"},

		// JavaScript
		{"script.js", "application/javascript", "JavaScript file"},

		// Data formats
		{"data.json", "application/json", "JSON file"},
		{"config.xml", "application/xml", "XML file"},

		// Documents
		{"report.pdf", "application/pdf", "PDF file"},
		{"document.doc", "application/msword", "Word doc"},
		{"document.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "Word docx"},
		{"spreadsheet.xls", "application/vnd.ms-excel", "Excel xls"},
		{"spreadsheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "Excel xlsx"},

		// Archives
		{"archive.zip", "application/zip", "ZIP file"},
		{"archive.tar", "application/x-tar", "TAR file"},
		{"archive.gz", "application/gzip", "GZIP file"},
		{"archive.gzip", "application/gzip", "GZIP file (alternate extension)"},

		// Images
		{"photo.jpg", "image/jpeg", "JPEG image"},
		{"photo.jpeg", "image/jpeg", "JPEG image (alternate extension)"},
		{"image.png", "image/png", "PNG image"},
		{"animation.gif", "image/gif", "GIF image"},
		{"icon.svg", "image/svg+xml", "SVG image"},
		{"photo.webp", "image/webp", "WebP image"},

		// Audio
		{"song.mp3", "audio/mpeg", "MP3 audio"},
		{"sound.wav", "audio/wav", "WAV audio"},

		// Video
		{"video.mp4", "video/mp4", "MP4 video"},
		{"video.webm", "video/webm", "WebM video"},

		// Unknown/fallback
		{"file.unknown", "application/octet-stream", "unknown extension"},
		{"noextension", "application/octet-stream", "no extension"},
		{"file.xyz123", "application/octet-stream", "random extension"},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			result := detectContentType(tc.filename)
			if result != tc.expected {
				t.Errorf("detectContentType(%q) = %q, expected %q",
					tc.filename, result, tc.expected)
			}
		})
	}
}

func TestDetectContentTypeCaseInsensitive(t *testing.T) {
	// Test that extension detection is case-insensitive
	tests := []struct {
		filename string
		expected string
	}{
		{"FILE.TXT", "text/plain"},
		{"Image.PNG", "image/png"},
		{"Document.PDF", "application/pdf"},
		{"Archive.ZIP", "application/zip"},
		{"video.MP4", "video/mp4"},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			result := detectContentType(tc.filename)
			if result != tc.expected {
				t.Errorf("detectContentType(%q) = %q, expected %q",
					tc.filename, result, tc.expected)
			}
		})
	}
}

func TestDetectContentTypeEdgeCases(t *testing.T) {
	tests := []struct {
		filename    string
		expected    string
		description string
	}{
		{"", "application/octet-stream", "empty filename"},
		{".", "application/octet-stream", "just a dot"},
		{".txt", "text/plain", "hidden file with extension"},
		{"file.name.txt", "text/plain", "multiple dots in filename"},
		{"file.tar.gz", "application/gzip", "compound extension"},
		{"a.b.c.d.pdf", "application/pdf", "many dots"},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			result := detectContentType(tc.filename)
			if result != tc.expected {
				t.Errorf("detectContentType(%q) = %q, expected %q",
					tc.filename, result, tc.expected)
			}
		})
	}
}
