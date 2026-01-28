package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Release represents a GitHub release
type Release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
}

// CheckForUpdates checks GitHub for a newer version of tinyMem
func CheckForUpdates() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/a-marczewski/tinymem/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "tinyMem-Version-Checker")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // No releases found
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if IsNewer(Version, latestVersion) {
		return latestVersion, nil
	}

	return "", nil
}

// IsNewer compares two version strings and returns true if latest is newer than current
func IsNewer(current, latest string) bool {
	if latest == "" {
		return false
	}

	cParts := strings.Split(current, ".")
	lParts := strings.Split(latest, ".")

	for i := 0; i < len(cParts) && i < len(lParts); i++ {
		cVal, _ := strconv.Atoi(cParts[i])
		lVal, _ := strconv.Atoi(lParts[i])

		if lVal > cVal {
			return true
		}
		if lVal < cVal {
			return false
		}
	}

	return len(lParts) > len(cParts)
}
