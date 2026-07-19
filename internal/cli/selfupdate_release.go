package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var githubAPI = "https://api.github.com/repos/rossijonas/stamp/releases/latest"

func releaseAssetName(version, goos, arch string) string {
	return fmt.Sprintf("stamp_%s_%s_%s.tar.gz", version, goos, arch)
}

func fetchLatestRelease() (*release, error) {
	req, err := http.NewRequest(http.MethodGet, githubAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch release: HTTP %d", resp.StatusCode)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release has no tag_name")
	}
	return &rel, nil
}

func findAsset(assets []asset, name string) *asset {
	for _, a := range assets {
		if a.Name == name {
			return &a
		}
	}
	return nil
}
