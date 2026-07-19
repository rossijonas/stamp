package cli

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func downloadFile(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	return data, nil
}

func verifyChecksum(data []byte, expectedHex string) error {
	h := sha256.Sum256(data)
	got := fmt.Sprintf("%x", h)
	if got != expectedHex {
		return fmt.Errorf("checksum mismatch: got %s, expected %s", got, expectedHex)
	}
	return nil
}

func checksumFor(assetName string, body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == assetName {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", assetName)
}

func extractBinary(data []byte, dest string) error {
	gzr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to open gzip: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		cleanName := filepath.Base(filepath.Clean(header.Name))
		if cleanName != "stamp" {
			continue
		}

		//nolint:gosec // dest is a temp file created by os.CreateTemp in the binary's directory
		out, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("failed to create temp binary: %w", err)
		}

		//nolint:gosec // G110: trusted source — downloaded from official GitHub release and SHA-256 verified
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return fmt.Errorf("failed to extract binary: %w", err)
		}

		if err := out.Close(); err != nil {
			return fmt.Errorf("failed to close temp binary: %w", err)
		}

		return nil
	}

	return fmt.Errorf("binary not found in archive")
}
