package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

type cacheData struct {
	LatestVersion string    `json:"latest_version"`
	LastChecked   time.Time `json:"last_checked"`
}

const cacheFileName = "update_cache.json"

// CheckAsync evaluates whether an update check is due. If it is, it spawns
// a detached process to fetch the latest release from GitHub so that the user's
// CLI experience is not blocked.
func CheckAsync(dataDir string) {
	cachePath := filepath.Join(dataDir, cacheFileName)

	// Check if cache exists and is fresh
	b, err := os.ReadFile(cachePath)
	if err == nil {
		var c cacheData
		if err := json.Unmarshal(b, &c); err == nil {
			if time.Since(c.LastChecked) < 24*time.Hour {
				return // Cache is fresh, do nothing
			}
		}
	}

	// Cache is missing or stale -> spawn detached worker
	execPath, err := os.Executable()
	if err != nil {
		execPath = os.Args[0]
	}

	cmd := exec.Command(execPath, "hidden-update-check")
	_ = cmd.Start() // start in background, ignore errors
}

// FetchAndUpdateCache does the actual heavy lifting: fetching the GitHub API
// and updating the local JSON cache. It is meant to be called by the detached worker.
func FetchAndUpdateCache(dataDir string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/KunalSin9h/yaad/releases/latest", nil)
	if err != nil {
		return err
	}
	// Use minimal headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var res struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	tag := res.TagName
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	// Save to cache
	c := cacheData{
		LatestVersion: tag,
		LastChecked:   time.Now(),
	}
	cachePath := filepath.Join(dataDir, cacheFileName)
	cb, _ := json.Marshal(c)
	return os.WriteFile(cachePath, cb, 0644)
}

// PrintWarning checks the local cache and prints a warning if a new version is available.
func PrintWarning(dataDir, currentVersion string) {
	if currentVersion == "dev" {
		return // Skip update warnings for local dev builds
	}
	
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}

	cachePath := filepath.Join(dataDir, cacheFileName)
	b, err := os.ReadFile(cachePath)
	if err != nil {
		return
	}

	var c cacheData
	if err := json.Unmarshal(b, &c); err != nil {
		return
	}

	// Check if cached latest version is strictly greater than current version
	if semver.IsValid(c.LatestVersion) && semver.IsValid(currentVersion) {
		if semver.Compare(c.LatestVersion, currentVersion) > 0 {
			fmt.Printf("\n💡 A new version of yaad (%s) is available!\n", c.LatestVersion)
			fmt.Printf("   Run: curl -fsSL https://yaad.knl.co.in/install.sh | bash\n\n")
		}
	}
}
