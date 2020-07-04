package gito

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type G struct {
	config *Config
}

func New(config *Config) *G {
	return &G{config: config}
}

func (g *G) Get(repo string) error {
	// where repo will live in the PATH
	fullPath := filepath.Join(g.config.active.path[0], repo)

	err := os.MkdirAll(fullPath, 0755)
	if err != nil {
		// TODO
		return err
	}

	if exists, err := doGitClone(repo, fullPath); exists {
		return fmt.Errorf("something already exists at %q", fullPath)
	} else if err != nil {
		return err
	}

	return nil
}

func doGitClone(repo, fullPath string) (bool, error) {
	_, err := os.Stat(fullPath)
	if !os.IsNotExist(err) {
		return true, nil
	}

	gitRepo := fmt.Sprintf("https://%s.git", repo) // simpler than ssh
	cmd := exec.Command("git", "clone", "--", gitRepo, fullPath)
	buf := &bytes.Buffer{}
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		// TODO
		return false, fmt.Errorf("error cloning repo: %v, stderr: %q", err, buf.String())
	}

	cmd = exec.Command("git", "submodule", "update", "--init", "--recursive")
	buf.Reset()
	cmd.Stderr = buf
	cmd.Dir = fullPath
	if err := cmd.Run(); err != nil {
		// TODO
		return false, fmt.Errorf("error updating submodules: %v, stderr: %s", err, buf.String())
	}

	return false, nil
}

func (g *G) Where(repo string) (string, error) {
	for _, dir := range g.config.active.path {
		fullPath, ok := in(repo, filepath.Join(dir), "", 0)
		if ok {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("%q not found", repo)
}

// in is a recursive function that checks for repo inside of dir.
func in(repo, dir, soFar string, depth int) (string, bool) {
	// don't check git directories
	if dir == ".git" {
		return "", false
	}

	// found it
	if repo == dir {
		return filepath.Join(soFar, repo), true
	}

	// in case repo is a partial name (ie r-medina/gito)
	fullPath := filepath.Join(soFar, dir, repo)
	f, err := os.Stat(fullPath)
	if err == nil {
		return fullPath, f.IsDir() // make sure we're not getting a file
	}

	// don't want to go past repositories
	if depth == 3 {
		return "", false
	}

	files, err := ioutil.ReadDir(filepath.Join(soFar, dir))
	if err != nil {
		return "", false
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		fullPath, ok := in(repo, file.Name(), filepath.Join(soFar, dir), depth+1)
		if ok {
			return fullPath, ok
		}
	}

	return "", false
}

func (g *G) URL(repo string) (string, error) {
	fullPath, err := g.Where(repo)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = fullPath
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error getting git remote for %q", repo)
	}

	url := buf.String()
	url = strings.TrimSpace(url)

	if strings.HasPrefix(url, "git@") {
		// TODO is this  the best implementation?

		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
		buf.Reset()
		buf.WriteString("https://")
		buf.WriteString(url)
		url = buf.String()
	}

	// when origin is specified with http, all we need to do is trim suffix

	url = strings.TrimSuffix(url, ".git")

	return url, nil
}
