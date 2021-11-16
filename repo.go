package executor

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/go-git/go-git/v5"
)

type repoSetupResult struct {
	dir      string
	commitID string
}

func setupRepo(repoURL, baseDir, dir string) (repoSetupResult, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return repoSetupResult{}, fmt.Errorf("error parsing repository URL: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	// Considering baseDir and dir.
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	if dir == "" {
		// spaces are super bad for paths in command-line
		dir = path.Base(u.Path)
	}

	repoPath := filepath.Join(baseDir, dir)
	log.Printf("Creating directory:%s\n", repoPath)

	if err := os.MkdirAll(repoPath, 0775); err != nil {
		return repoSetupResult{}, fmt.Errorf("error when creating temporary dir: %w", err)
	}
	cid, err := cloneRepository(repoPath, u.String())
	if err != nil {
		return repoSetupResult{}, fmt.Errorf("error when cloning repo(%s): %w", repoURL, err)
	}
	log.Printf("Repo cloned successfully! Commit:%s New dir:%s\n", cid, repoPath)
	return repoSetupResult{repoPath, cid}, nil
}

// cloneRepository is responsible for get the latest code version of pipeline repository.
// Creates and returns the DefaultBaseDir for the pipeline and the latest commit in the repository.
func cloneRepository(dir, repoURL string) (string, error) {
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("error cloning the repository. error removing previous directory: %q", err)
	}

	log.Printf("Cloning the repository [%s] into [%s]\n", repoURL, dir)
	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return "", fmt.Errorf("error cloning the repository: %q", err)
	}

	ref, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("error cloning the repository. error getting the HEAD reference of the repository: %q", err)
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("error cloning the repository. error getting the lattest commit of the repository: %q", err)
	}
	return commit.Hash.String(), nil
}
