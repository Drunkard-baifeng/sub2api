//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release        *GitHubRelease
	latestErr      error
	latestRepo     string
	recentReleases []*GitHubRelease
	recentErr      error
	recentRepo     string
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(_ context.Context, repo string) (*GitHubRelease, error) {
	s.latestRepo = repo
	return s.release, s.latestErr
}

func (s *updateServiceGitHubClientStub) FetchRecentReleases(_ context.Context, repo string, _ int) ([]*GitHubRelease, error) {
	s.recentRepo = repo
	return s.recentReleases, s.recentErr
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.132",
				Name:    "v0.1.132",
			},
		},
		"0.1.132",
		"release",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
}

func newRollbackTestService(current string, releases []*GitHubRelease) *UpdateService {
	return NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentReleases: releases},
		current,
		"release",
	)
}

func TestUpdateServiceListRollbackVersionsFiltersAndCaps(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.148", PublishedAt: "2026-07-09T00:00:00Z"},                       // newer than current: excluded
		{TagName: "v0.1.147", PublishedAt: "2026-07-08T00:00:00Z"},                       // current: excluded
		{TagName: "v0.1.146-rc1", PublishedAt: "2026-07-07T12:00:00Z", Prerelease: true}, // prerelease: excluded
		{TagName: "v0.1.146", PublishedAt: "2026-07-07T00:00:00Z"},
		{TagName: "v0.1.145", PublishedAt: "2026-07-06T00:00:00Z", Draft: true}, // draft: excluded
		{TagName: "v0.1.144", PublishedAt: "2026-07-05T00:00:00Z"},
		{TagName: "v0.1.144", PublishedAt: "2026-07-05T00:00:00Z"}, // duplicate: excluded
		{TagName: "v0.1.143", PublishedAt: "2026-07-04T00:00:00Z"},
		{TagName: "v0.1.142", PublishedAt: "2026-07-03T00:00:00Z"}, // beyond cap of 3: excluded
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.1.146", versions[0].Version)
	require.Equal(t, "0.1.144", versions[1].Version)
	require.Equal(t, "0.1.143", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsSortsUnorderedInput(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.144"},
		{TagName: "v0.1.146"},
		{TagName: "v0.1.145"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.1.146", versions[0].Version)
	require.Equal(t, "0.1.145", versions[1].Version)
	require.Equal(t, "0.1.144", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsEmptyWhenNoneOlder(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.147"},
		{TagName: "v0.1.148"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestUpdateServiceListRollbackVersionsPropagatesFetchError(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentErr: errors.New("github unavailable")},
		"0.1.147",
		"release",
	)

	_, err := svc.ListRollbackVersions(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "github unavailable")
}

func TestUpdateServiceRollbackToVersionRejectsDisallowedTargets(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.148"},
		{TagName: "v0.1.147"},
		{TagName: "v0.1.146"},
		{TagName: "v0.1.145"},
		{TagName: "v0.1.144"},
		{TagName: "v0.1.143"},
		{TagName: "v0.1.142"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	for _, target := range []string{
		"",         // empty
		"0.1.147",  // current version
		"v0.1.147", // current version with prefix
		"0.1.148",  // newer than current
		"0.1.142",  // older than the 3 most recent
		"9.9.9",    // nonexistent
	} {
		err := svc.RollbackToVersion(context.Background(), target)
		require.ErrorIs(t, err, ErrRollbackVersionNotAllowed, "target %q should be rejected", target)
	}
}

func TestUpdateServiceRollbackToVersionAcceptsVPrefix(t *testing.T) {
	// No platform asset in the release: the target passes the allowlist check
	// and fails later at asset lookup, proving the version itself was accepted.
	releases := []*GitHubRelease{
		{TagName: "v0.1.147"},
		{TagName: "v0.1.146"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	err := svc.RollbackToVersion(context.Background(), "v0.1.146")

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrRollbackVersionNotAllowed)
	require.Contains(t, err.Error(), "no compatible release found")
}

func TestUpdateServiceCompareCustomVersions(t *testing.T) {
	for _, tc := range []struct {
		current string
		latest  string
		want    int
	}{
		{"0.2.13", "0.2.13-custom.1", -1},
		{"v0.2.13-custom.1", "0.2.13-custom.2", -1},
		{"0.2.13-custom.9", "v0.2.13-custom.10", -1},
		{"0.2.13-custom.10", "0.2.13-custom.2", 1},
		{"0.2.13-custom.1", "v0.2.13-custom.1", 0},
		{"0.2.13-custom.1", "0.2.13", 1},
		{"0.2.13-custom.10", "0.2.14-custom.1", -1},
		{"0.2.14", "0.2.13-custom.10", 1},
		{"0.2.13-custom.10", "0.2.14", -1},
		{"0.2.13", "0.2.13", 0},
		// Existing non-custom suffix handling stays unchanged.
		{"0.2.13-rc1", "0.2.13", 0},
		{"0.2.13-custom.2-rc.1", "0.2.13-custom.1", -1},
	} {
		t.Run(tc.current+"_to_"+tc.latest, func(t *testing.T) {
			require.Equal(t, tc.want, compareVersions(tc.current, tc.latest))
		})
	}
}

func TestUpdateServiceUsesForkAndCachesCustomUpdate(t *testing.T) {
	ctx := context.Background()
	cache := &updateServiceCacheStub{}
	client := &updateServiceGitHubClientStub{
		release: &GitHubRelease{TagName: "v0.2.13-custom.2"},
	}
	svc := NewUpdateService(cache, client, "0.2.13-custom.1", "release")
	info, err := svc.CheckUpdate(ctx, false)
	require.NoError(t, err)
	require.True(t, info.HasUpdate)
	require.False(t, info.Cached)
	require.Equal(t, "Drunkard-baifeng/sub2api", client.latestRepo)
	var cached map[string]any
	require.NoError(t, json.Unmarshal([]byte(cache.data), &cached))
	require.Equal(t, "Drunkard-baifeng/sub2api", cached["repository"])

	// Reuse a valid fork cache even if GitHub is unavailable.
	client.latestErr = errors.New("github unavailable")
	info, err = svc.CheckUpdate(ctx, true)
	require.NoError(t, err)
	require.True(t, info.Cached)
	require.True(t, info.HasUpdate)
	require.Contains(t, info.Warning, "github unavailable")

	// The running version is re-evaluated after an upgrade.
	updated := NewUpdateService(cache, client, "0.2.13-custom.2", "release")
	info, err = updated.CheckUpdate(ctx, false)
	require.NoError(t, err)
	require.True(t, info.Cached)
	require.False(t, info.HasUpdate)
}

func TestUpdateServiceRejectsUpstreamAndLegacyCache(t *testing.T) {
	for _, repository := range []string{"", "Wei-Shaw/sub2api"} {
		for _, unavailable := range []bool{false, true} {
			t.Run(repository+map[bool]string{true: "/offline", false: "/online"}[unavailable], func(t *testing.T) {
				data := map[string]any{
					"latest": "99.0.0", "timestamp": time.Now().Unix(),
					"release_info": &ReleaseInfo{HTMLURL: "https://github.com/Wei-Shaw/sub2api/releases/tag/v99.0.0"},
				}
				if repository != "" {
					data["repository"] = repository
				}
				encoded, err := json.Marshal(data)
				require.NoError(t, err)
				client := &updateServiceGitHubClientStub{release: &GitHubRelease{TagName: "v0.2.13-custom.2"}}
				if unavailable {
					client.latestErr = errors.New("github unavailable")
				}
				svc := NewUpdateService(&updateServiceCacheStub{data: string(encoded)}, client, "0.2.13-custom.1", "release")
				info, err := svc.CheckUpdate(context.Background(), false)
				require.NoError(t, err)
				require.False(t, info.Cached)
				require.Equal(t, "Drunkard-baifeng/sub2api", client.latestRepo)
				if unavailable {
					require.False(t, info.HasUpdate)
					require.Nil(t, info.ReleaseInfo)
					require.Contains(t, info.Warning, "github unavailable")
				} else {
					require.True(t, info.HasUpdate)
					require.Equal(t, "0.2.13-custom.2", info.LatestVersion)
				}
			})
		}
	}
}

func TestUpdateServiceCustomRollbackOrderAndRepository(t *testing.T) {
	client := &updateServiceGitHubClientStub{recentReleases: []*GitHubRelease{
		{TagName: "v0.2.14-custom.1"},
		{TagName: "v0.2.13-custom.11"},
		{TagName: "v0.2.13-custom.2"},
		{TagName: "v0.2.13-custom.10"},
		{TagName: "v0.2.13-custom.9"},
		{TagName: "v0.2.13"},
		{TagName: "v0.2.13-custom.8", Prerelease: true},
	}}
	svc := NewUpdateService(&updateServiceCacheStub{}, client, "0.2.13-custom.11", "release")
	versions, err := svc.ListRollbackVersions(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Drunkard-baifeng/sub2api", client.recentRepo)
	require.Len(t, versions, 3)
	require.Equal(t, "0.2.13-custom.10", versions[0].Version)
	require.Equal(t, "0.2.13-custom.9", versions[1].Version)
	require.Equal(t, "0.2.13-custom.2", versions[2].Version)
}
