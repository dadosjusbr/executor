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

func setupRepo(repoURL string) (repoSetupResult, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return repoSetupResult{}, fmt.Errorf("error parsing repository URL: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	// spaces are super bad for paths in command-line
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("dadosjusbr-executor-%s", path.Base(u.Path)))
	log.Printf("Creating directory:%s\n", tmpDir)

	if err := os.MkdirAll(tmpDir, 0775); err != nil {
		return repoSetupResult{}, fmt.Errorf("error when creating temporary dir: %w", err)
	}
	cid, err := cloneRepository(tmpDir, u.String())
	if err != nil {
		return repoSetupResult{}, fmt.Errorf("error when cloning repo(%s): %w", repoURL, err)
	}
	dir := path.Join(tmpDir)
	log.Printf("Repo cloned successfully! Commit:%s New dir:%s\n", cid, dir)
	return repoSetupResult{dir, cid}, nil
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
