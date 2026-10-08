package dashboard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type GithubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type UpdateStatus struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	DownloadURL    string `json:"download_url,omitempty"`
	AssetName      string `json:"asset_name,omitempty"`
	Error          string `json:"error,omitempty"`
}

const (
	checksumExtension = ".sha256"
)

var releaseHTTPClient = &http.Client{Timeout: 30 * time.Second}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update to the latest version",
	Run:   runUpdate,
}

func init() {
	RootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	status, err := checkForUpdate()
	if err != nil {
		fatal(fmt.Sprintf("Error checking for updates: %v", err), 1)
	}

	if !status.HasUpdate {
		fmt.Printf("You are already on the latest version (%s)\n", status.CurrentVersion)
		return
	}

	fmt.Printf("Updating from version %s to %s...\n", status.CurrentVersion, status.LatestVersion)

	if err := performUpdate(status); err != nil {
		fatal(fmt.Sprintf("Error during update: %v", err), 1)
	}

	fmt.Printf("Update complete! You are now on version %s\n", status.LatestVersion)
}

// checkForUpdate checks if a new version is available
func checkForUpdate() (*UpdateStatus, error) {
	currentVersion := Version
	status := &UpdateStatus{
		CurrentVersion: currentVersion,
	}

	// Get latest release info
	release, err := getLatestRelease()
	if err != nil {
		status.Error = err.Error()
		return status, err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	status.LatestVersion = latestVersion
	status.HasUpdate = isNewer(latestVersion, currentVersion)

	if status.HasUpdate {
		// Find the appropriate asset for current platform
		expectedName := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
		for _, asset := range release.Assets {
			if asset.Name == expectedName+".tar.gz" {
				status.DownloadURL = asset.BrowserDownloadURL
				status.AssetName = asset.Name
				break
			}
		}

		if status.DownloadURL == "" {
			err := fmt.Errorf("no release found for %s/%s", runtime.GOOS, runtime.GOARCH)
			status.Error = err.Error()
			return status, err
		}
	}

	return status, nil
}

// performUpdate downloads and installs the update
func performUpdate(status *UpdateStatus) error {
	if os.Getenv("CRONTAB_DASHBOARD_CONTAINER") == "1" {
		return fmt.Errorf("update this container by pulling a new image and recreating it")
	}
	if !status.HasUpdate {
		return fmt.Errorf("no update available")
	}
	release, err := getLatestRelease()
	if err != nil {
		return err
	}
	checksum, err := downloadChecksum(release, status.AssetName+checksumExtension)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	directory := filepath.Dir(executable)
	stage, err := os.MkdirTemp(directory, ".dashboard-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	archive := filepath.Join(stage, "bundle.tar.gz")
	if err := downloadAndVerifyFile(status.DownloadURL, archive, strings.TrimSpace(string(checksum))); err != nil {
		return err
	}
	if err := extractBundle(archive, stage); err != nil {
		return err
	}
	return installBundle(directory, stage)
}

func getLatestRelease() (*GithubRelease, error) {
	resp, err := releaseHTTPClient.Get("https://api.github.com/repos/cronitorio/crontab-dashboard/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func downloadChecksum(release *GithubRelease, checksumFile string) ([]byte, error) {
	for _, asset := range release.Assets {
		if asset.Name == checksumFile {
			resp, err := releaseHTTPClient.Get(asset.BrowserDownloadURL)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			return io.ReadAll(resp.Body)
		}
	}
	return nil, fmt.Errorf("checksum file not found for release")
}

func downloadAndVerifyFile(url, dest, expectedChecksum string) error {
	// Download to memory first to verify before writing to disk
	resp, err := releaseHTTPClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Read the entire response body into memory and calculate checksum
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Verify checksum before proceeding
	hasher := sha256.New()
	hasher.Write(body)
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum verification failed (expected: %s, got: %s)", expectedChecksum, actualChecksum)
	}

	return os.WriteFile(dest, body, 0600)
}

func isNewer(latest, current string) bool {
	lp, cp := strings.Split(strings.TrimPrefix(latest, "v"), "."), strings.Split(strings.TrimPrefix(current, "v"), ".")
	for i := 0; i < 3; i++ {
		var l, c int
		if i < len(lp) {
			l, _ = strconv.Atoi(lp[i])
		}
		if i < len(cp) {
			c, _ = strconv.Atoi(cp[i])
		}
		if l != c {
			return l > c
		}
	}
	return false
}
