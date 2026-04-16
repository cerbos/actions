// Copyright 2026 Zenauth Ltd.

package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/multierr"
	"golang.org/x/sync/semaphore"

	"github.com/cerbos/actions/hack/go/pkg/digest"
	"github.com/cerbos/actions/hack/go/pkg/log"
	"github.com/cerbos/actions/hack/go/pkg/semver"
)

const (
	maxConcurrency = 8
	minReleaseAge  = 5 * 24 * time.Hour
)

type Client struct {
	github    *github.Client
	semaphore *semaphore.Weighted
}

func NewClient(ctx context.Context) *Client {
	client := github.NewClient(nil)

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = client.WithAuthToken(token)
	} else {
		log.Warn(ctx, "GITHUB_TOKEN not set; stricter rate limits will apply")
	}

	return &Client{
		github:    client,
		semaphore: semaphore.NewWeighted(maxConcurrency),
	}
}

type Repository struct {
	Owner string
	Name  string
}

func (r Repository) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("owner", r.Owner),
		slog.String("name", r.Name),
	)
}

func (r Repository) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

type Release struct {
	Created time.Time
	Assets  map[string]*Asset
	Repo    Repository
	Tag     string
	Version semver.Version
}

func (r Release) Asset(name string) (*Asset, error) {
	if asset, ok := r.Assets[name]; ok {
		return asset, nil
	}
	return nil, fmt.Errorf("missing asset %s", name)
}

func (r Release) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("repo", r.Repo),
		slog.Any("version", r.Version),
	)
}

func (r Release) String() string {
	return fmt.Sprintf("%s@%s", r.Repo, r.Version)
}

type Asset struct {
	Name     string
	URL      string
	Contents []byte
	ID       int64
	Digest   digest.SHA256
}

type FindNewerReleaseOption func(*findNewerReleaseOptions)

type findNewerReleaseOptions struct {
	versionConstraint func(semver.Version) bool
	versionFromTag    func(string) semver.Version
}

func VersionConstraint(versionConstraint func(semver.Version) bool) FindNewerReleaseOption {
	return func(options *findNewerReleaseOptions) {
		options.versionConstraint = versionConstraint
	}
}

func VersionFromTag(versionFromTag func(string) semver.Version) FindNewerReleaseOption {
	return func(options *findNewerReleaseOptions) {
		options.versionFromTag = versionFromTag
	}
}

func (c *Client) FindNewerRelease(ctx context.Context, repo Repository, oldVersion semver.Version, options ...FindNewerReleaseOption) (*Release, error) {
	ctx = log.With(ctx, "repo", repo)

	opts := findNewerReleaseOptions{
		versionConstraint: func(semver.Version) bool {
			return true
		},
		versionFromTag: func(tag string) semver.Version {
			return semver.Version(tag)
		},
	}

	for _, option := range options {
		option(&opts)
	}

	if err := c.acquire(ctx); err != nil {
		return nil, err
	}
	defer c.release()

	var newer *github.RepositoryRelease
	const maxPerPage = 100
	for release, err := range c.github.Repositories.ListReleasesIter(ctx, repo.Owner, repo.Name, &github.ListOptions{PerPage: maxPerPage}) {
		if err != nil {
			return nil, fmt.Errorf("failed to list releases in %s: %w", repo, err)
		}

		createdAt := release.GetCreatedAt().Time

		tag := release.GetTagName()
		version := opts.versionFromTag(tag)
		if !version.IsValid() {
			log.Debug(ctx, "Skipped", "reason", "invalid tag", "tag", tag)
			continue
		}

		ctx := log.With(ctx, "oldVersion", oldVersion, "newVersion", version)

		if release.GetPrerelease() {
			log.Debug(ctx, "Skipped", "reason", "prerelease")
			continue
		}

		if createdAt.IsZero() {
			log.Debug(ctx, "Skipped", "reason", "missing timestamp")
			continue
		}

		if repo.Owner != "cerbos" {
			if age := time.Since(createdAt); age < minReleaseAge {
				log.Debug(ctx, "Skipped", "reason", "too recent", "age", age)
				continue
			}
		}

		if semver.Compare(version, oldVersion) <= 0 {
			log.Debug(ctx, "Skipped", "reason", "not newer")
			continue
		}

		if !opts.versionConstraint(version) {
			log.Debug(ctx, "Skipped", "reason", "constraint not satisfied")
			continue
		}

		log.Debug(ctx, "Found newer release")
		oldVersion = version
		newer = release
	}

	if newer == nil {
		log.Debug(ctx, "No newer release found")
		return nil, nil
	}

	release := &Release{
		Repo:    repo,
		Tag:     newer.GetTagName(),
		Version: oldVersion,
		Created: newer.GetCreatedAt().Time,
		Assets:  make(map[string]*Asset, len(newer.Assets)),
	}

	for _, asset := range newer.Assets {
		name := asset.GetName()

		digest, err := digest.Parse(asset.GetDigest())
		if err != nil {
			return nil, fmt.Errorf("failed to parse digest for asset %s: %w", name, err)
		}

		release.Assets[name] = &Asset{
			ID:     asset.GetID(),
			Name:   name,
			URL:    asset.GetBrowserDownloadURL(),
			Digest: digest,
		}
	}

	return release, nil
}

func (c *Client) DownloadAssets(ctx context.Context, release *Release, names ...string) error {
	if len(names) == 1 {
		return c.downloadAssetContents(ctx, release, names[0])
	}

	downloads := pool.New().WithContext(ctx).WithFailFast()

	for _, name := range names {
		downloads.Go(func(ctx context.Context) error {
			return c.downloadAssetContents(ctx, release, name)
		})
	}

	return downloads.Wait()
}

func (c *Client) downloadAssetContents(ctx context.Context, release *Release, name string) (err error) {
	asset, err := release.Asset(name)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to download asset %s: %w", name, err)
		}
	}()

	reader, err := c.DownloadAsset(ctx, release, asset)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(reader))

	asset.Contents, err = io.ReadAll(reader)
	return err
}

type downloader struct {
	source io.ReadCloser
	eof    func(int)
	close  func() error
	size   int
}

func (d *downloader) Read(buffer []byte) (int, error) {
	n, err := d.source.Read(buffer)
	d.size += n
	if err == io.EOF && d.eof != nil {
		d.eof(d.size)
	}
	return n, err
}

func (d *downloader) Close() error {
	return multierr.Append(d.source.Close(), d.close())
}

func (c *Client) DownloadAsset(ctx context.Context, release *Release, asset *Asset) (io.ReadCloser, error) {
	if err := c.acquire(ctx); err != nil {
		return nil, err
	}

	start := time.Now()

	contents, _, err := c.github.Repositories.DownloadReleaseAsset(ctx, release.Repo.Owner, release.Repo.Name, asset.ID, http.DefaultClient)
	if err != nil {
		return nil, err
	}

	return &downloader{
		source: digest.NewReader(contents, asset.Digest),
		eof: func(size int) {
			log.Debug(ctx, "Downloaded asset", "release", release, "asset", asset.Name, "size", size, "duration", time.Since(start))
		},
		close: func() error {
			c.release()
			return contents.Close()
		},
	}, nil
}

func (c *Client) DownloadFile(ctx context.Context, repo Repository, ref, path string) (io.ReadCloser, error) {
	file, _, err := c.github.Repositories.DownloadContents(ctx, repo.Owner, repo.Name, path, &github.RepositoryContentGetOptions{Ref: ref})
	return file, err
}

type Commit struct {
	SHA     string
	Message string
}

func (c Commit) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("sha", c.SHA),
		slog.Any("message", c.Message),
	)
}

func (c *Client) FindLatestCommitForPath(ctx context.Context, repo Repository, path string) (*Commit, error) {
	commits, _, err := c.github.Repositories.ListCommits(ctx, repo.Owner, repo.Name, &github.CommitsListOptions{
		Path:        path,
		ListOptions: github.ListOptions{PerPage: 1},
	})
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("%s not found in %s", path, repo)
	}

	message, _, _ := strings.Cut(commits[0].GetCommit().GetMessage(), "\n")

	return &Commit{
		SHA:     commits[0].GetSHA(),
		Message: message,
	}, nil
}

func (c *Client) acquire(ctx context.Context) error {
	return c.semaphore.Acquire(ctx, 1)
}

func (c *Client) release() {
	c.semaphore.Release(1)
}
