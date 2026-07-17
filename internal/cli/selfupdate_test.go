package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tarGzWithBinary(t *testing.T, binaryName string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	hdr := &tar.Header{
		Name: binaryName,
		Size: int64(len(content)),
		Mode: 0755,
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func checksumsFile(tarballName string, tarballData []byte) []byte {
	h := sha256.Sum256(tarballData)
	return []byte(fmt.Sprintf("%x  %s\n", h, tarballName))
}

func setupSelfUpdateServer(t *testing.T, tagName, tarballName string, tarballData []byte) (*httptest.Server, string) {
	t.Helper()
	checksumName := "checksums.txt"
	checksumData := checksumsFile(tarballName, tarballData)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			base := fmt.Sprintf("http://%s", r.Host)
			rel := release{
				TagName: tagName,
				Assets: []asset{
					{Name: tarballName, BrowserDownloadURL: base + "/downloads/" + tarballName},
					{Name: checksumName, BrowserDownloadURL: base + "/downloads/" + checksumName},
				},
			}
			data, _ := json.Marshal(rel)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		case strings.HasSuffix(r.URL.Path, checksumName):
			_, _ = w.Write(checksumData)
		case strings.HasSuffix(r.URL.Path, tarballName):
			_, _ = w.Write(tarballData)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Store the previous globals and set test values
	oldAPI := githubAPI
	githubAPI = ts.URL + "/repos/rossijonas/stamp/releases/latest"

	t.Cleanup(func() {
		ts.Close()
		githubAPI = oldAPI
	})

	return ts, ts.URL
}

func assetName(version string) string {
	return releaseAssetName(version, runtime.GOOS, runtime.GOARCH)
}

func TestSelfUpdate_CheckNewVersion(t *testing.T) {
	tarballData := tarGzWithBinary(t, "stamp", []byte("new-binary-content"))
	tarballName := assetName("v2.0.0")

	setupSelfUpdateServer(t, "v2.0.0", tarballName, tarballData)

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update", "--check"})
	err := root.Execute()
	require.Error(t, err)
	output := buf.String()
	assert.Contains(t, output, "Self-Update")
	assert.Contains(t, output, "v2.0.0")
	assert.Contains(t, output, "A new version is available")
}

func TestSelfUpdate_CheckUpToDate(t *testing.T) {
	tarballData := tarGzWithBinary(t, "stamp", []byte("new-binary-content"))
	tarballName := assetName("v1.0.0")

	setupSelfUpdateServer(t, "v1.0.0", tarballName, tarballData)

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update", "--check"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Already up to date")
}

func TestSelfUpdate_AlreadyUpToDate(t *testing.T) {
	tarballData := tarGzWithBinary(t, "stamp", []byte("new-binary-content"))
	tarballName := assetName("v1.0.0")

	setupSelfUpdateServer(t, "v1.0.0", tarballName, tarballData)

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Already up to date")
}

func TestSelfUpdate_FullUpdate(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho stamp v2.0.0")
	tarballData := tarGzWithBinary(t, "stamp", binaryContent)
	tarballName := assetName("v2.0.0")

	setupSelfUpdateServer(t, "v2.0.0", tarballName, tarballData)

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	oldExec := osExecutable
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // temp file in test directory
	require.NoError(t, os.WriteFile(exePath, []byte("old-binary"), 0755))
	osExecutable = func() (string, error) { return exePath, nil }
	defer func() { osExecutable = oldExec }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Updated to v2.0.0")

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(exePath)
	require.NoError(t, err)
	assert.Equal(t, string(binaryContent), string(data))
}

func TestSelfUpdate_SelfUpgradeAlias(t *testing.T) {
	tarballData := tarGzWithBinary(t, "stamp", []byte("new-binary"))
	tarballName := assetName("v2.0.0")

	setupSelfUpdateServer(t, "v2.0.0", tarballName, tarballData)

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	oldExec := osExecutable
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // temp file in test directory
	require.NoError(t, os.WriteFile(exePath, []byte("old"), 0755))
	osExecutable = func() (string, error) { return exePath, nil }
	defer func() { osExecutable = oldExec }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-upgrade"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Updated to v2.0.0")
}

func TestSelfUpdate_NoReleaseFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	oldAPI := githubAPI
	githubAPI = ts.URL + "/notfound"
	defer func() { githubAPI = oldAPI }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch release")
}

func TestSelfUpdate_IntegrityCheckFails(t *testing.T) {
	tarballData := tarGzWithBinary(t, "stamp", []byte("content"))
	tarballName := assetName("v2.0.0")
	wrongChecksum := []byte("0000000000000000000000000000000000000000000000000000000000000000  " + tarballName + "\n")
	checksumName := "checksums.txt"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			base := fmt.Sprintf("http://%s", r.Host)
			rel := release{
				TagName: "v2.0.0",
				Assets: []asset{
					{Name: tarballName, BrowserDownloadURL: base + "/downloads/" + tarballName},
					{Name: checksumName, BrowserDownloadURL: base + "/downloads/" + checksumName},
				},
			}
			data, _ := json.Marshal(rel)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		case strings.HasSuffix(r.URL.Path, checksumName):
			_, _ = w.Write(wrongChecksum)
		case strings.HasSuffix(r.URL.Path, tarballName):
			_, _ = w.Write(tarballData)
		}
	}))
	defer ts.Close()

	oldAPI := githubAPI
	githubAPI = ts.URL + "/repos/rossijonas/stamp/releases/latest"
	defer func() { githubAPI = oldAPI }()

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	oldExec := osExecutable
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // temp file in test directory
	require.NoError(t, os.WriteFile(exePath, []byte("old"), 0755))
	osExecutable = func() (string, error) { return exePath, nil }
	defer func() { osExecutable = oldExec }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "integrity check failed")
}

func TestSelfUpdate_MissingChecksum(t *testing.T) {
	tarballData := tarGzWithBinary(t, "stamp", []byte("content"))
	tarballName := assetName("v2.0.0")
	wrongChecksum := []byte("# empty checksums file\n")
	checksumName := "checksums.txt"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			base := fmt.Sprintf("http://%s", r.Host)
			rel := release{
				TagName: "v2.0.0",
				Assets: []asset{
					{Name: tarballName, BrowserDownloadURL: base + "/downloads/" + tarballName},
					{Name: checksumName, BrowserDownloadURL: base + "/downloads/" + checksumName},
				},
			}
			data, _ := json.Marshal(rel)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		case strings.HasSuffix(r.URL.Path, checksumName):
			_, _ = w.Write(wrongChecksum)
		case strings.HasSuffix(r.URL.Path, tarballName):
			_, _ = w.Write(tarballData)
		}
	}))
	defer ts.Close()

	oldAPI := githubAPI
	githubAPI = ts.URL + "/repos/rossijonas/stamp/releases/latest"
	defer func() { githubAPI = oldAPI }()

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	oldExec := osExecutable
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // temp file in test directory
	require.NoError(t, os.WriteFile(exePath, []byte("old"), 0755))
	osExecutable = func() (string, error) { return exePath, nil }
	defer func() { osExecutable = oldExec }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse checksums")
}

func TestSelfUpdate_NoBinaryInArchive(t *testing.T) {
	// Create a valid tar.gz but without a "stamp" entry
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "README.md", Size: 4, Mode: 0644}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte("data"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	tarballData := buf.Bytes()
	tarballName := assetName("v2.0.0")

	setupSelfUpdateServer(t, "v2.0.0", tarballName, tarballData)

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	oldExec := osExecutable
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // temp file in test directory
	require.NoError(t, os.WriteFile(exePath, []byte("old"), 0755))
	osExecutable = func() (string, error) { return exePath, nil }
	defer func() { osExecutable = oldExec }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	output := new(bytes.Buffer)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"self-update"})
	err = root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binary not found in archive")
}

func TestSelfUpdate_ExecutableError(t *testing.T) {
	oldExec := osExecutable
	osExecutable = func() (string, error) { return "", assert.AnError }
	defer func() { osExecutable = oldExec }()

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get executable path")
}

func TestSelfUpdate_AssetNotFound(t *testing.T) {
	tarballData := tarGzWithBinary(t, "stamp", []byte("content"))
	tarballName := "wrong-name.tar.gz"

	setupSelfUpdateServer(t, "v2.0.0", tarballName, tarballData)

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	oldExec := osExecutable
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // temp file in test directory
	require.NoError(t, os.WriteFile(exePath, []byte("old"), 0755))
	osExecutable = func() (string, error) { return exePath, nil }
	defer func() { osExecutable = oldExec }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSelfUpdate_PermissionDenied(t *testing.T) {
	tarballData := tarGzWithBinary(t, "stamp", []byte("content"))
	tarballName := assetName("v2.0.0")

	setupSelfUpdateServer(t, "v2.0.0", tarballName, tarballData)

	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	// Write binary first, then make dir read-only
	oldExec := osExecutable
	readonlyDir := t.TempDir()
	exePath := filepath.Join(readonlyDir, "stamp")
	//nolint:gosec // temp file in test directory
	require.NoError(t, os.WriteFile(exePath, []byte("old"), 0755))
	//nolint:gosec // directory permissions for testing read-only dir scenario
	require.NoError(t, os.Chmod(readonlyDir, 0500))
	//nolint:gosec // restore permissions for cleanup
	t.Cleanup(func() { _ = os.Chmod(readonlyDir, 0700) })
	osExecutable = func() (string, error) { return exePath, nil }
	defer func() { osExecutable = oldExec }()

	root := NewRootCmd(WithAdapters(nil), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"self-update"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}
