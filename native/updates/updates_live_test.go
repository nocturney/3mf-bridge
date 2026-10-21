package updates

import (
	"context"
	"os"
	"testing"
	"time"
)

// This network test is deliberately opt-in: normal CI/unit tests are offline.
func TestPublicGitHubReleaseCheck(t *testing.T) {
	if os.Getenv("BRIDGE_TEST_GITHUB") != "1" {
		t.Skip("Set BRIDGE_TEST_GITHUB=1 to check the real GitHub endpoint")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := CheckInstalled(ctx, "0.4.0-beta.1")
	if err != nil {
		t.Fatalf("GitHub Releases request failed: %v", err)
	}
	switch result.Status {
	case "no_releases", "up_to_date", "update_available", "platform_unavailable":
		t.Logf("Live GitHub check: %+v", result)
	default:
		t.Fatalf("Unexpected update-check status: %+v", result)
	}
}
