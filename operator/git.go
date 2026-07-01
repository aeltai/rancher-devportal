package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type gitCreds struct {
	Username string
	Password string
	Token    string
}

func readGitCreds(ctx context.Context, r *reconciler, ns, secretName string) (gitCreds, error) {
	sec, err := r.core.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return gitCreds{}, fmt.Errorf("get secret %s/%s: %w", ns, secretName, err)
	}
	return parseGitSecret(sec), nil
}

func parseGitSecret(sec *corev1.Secret) gitCreds {
	c := gitCreds{}
	if sec == nil || sec.Data == nil {
		return c
	}
	c.Username = string(sec.Data["username"])
	c.Password = string(sec.Data["password"])
	c.Token = string(sec.Data["token"])
	if c.Token == "" {
		c.Token = string(sec.Data["password"])
	}
	if c.Username == "" {
		c.Username = "git"
	}
	return c
}

// pushManifests clones repo, writes files under gitPath, commits and pushes.
func pushManifests(repo, branch, gitPath string, files map[string]string, creds gitCreds) (commit string, err error) {
	if repo == "" {
		return "", fmt.Errorf("git repo URL is empty")
	}
	tmp, err := os.MkdirTemp("", "platform-git-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	authURL, err := embedCreds(repo, creds)
	if err != nil {
		return "", err
	}

	run := func(dir string, args ...string) error {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	if err := run(tmp, "git", "clone", "--depth", "1", "--branch", branch, authURL, "."); err != nil {
		// Try clone without branch (empty repo / branch not exists yet)
		if err2 := run(tmp, "git", "clone", authURL, "."); err2 != nil {
			return "", fmt.Errorf("clone: %w", err)
		}
		_ = run(tmp, "git", "checkout", "-B", branch)
	}

	for relPath, content := range files {
		full := filepath.Join(tmp, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return "", err
		}
	}

	_ = run(tmp, "git", "config", "user.email", "platform-operator@devportal.io")
	_ = run(tmp, "git", "config", "user.name", "Platform Operator")

	if err := run(tmp, "git", "add", gitPath); err != nil {
		return "", err
	}
	msg := fmt.Sprintf("platform: provision %s", filepath.Base(strings.Trim(gitPath, "/")))
	if err := run(tmp, "git", "commit", "-m", msg); err != nil {
		if strings.Contains(err.Error(), "nothing to commit") {
			out, _ := exec.Command("git", "-C", tmp, "rev-parse", "HEAD").CombinedOutput()
			return strings.TrimSpace(string(out)), nil
		}
		return "", err
	}
	if err := run(tmp, "git", "push", "origin", branch); err != nil {
		return "", fmt.Errorf("push: %w", err)
	}
	out, err := exec.Command("git", "-C", tmp, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func embedCreds(repoURL string, creds gitCreds) (string, error) {
	repoURL = strings.TrimSpace(repoURL)
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("only https git repos supported")
	}
	token := creds.Token
	if token == "" {
		return repoURL, nil
	}
	rest := strings.TrimPrefix(repoURL, "https://")
	user := creds.Username
	if user == "" {
		user = "git"
	}
	return fmt.Sprintf("https://%s:%s@%s", user, token, rest), nil
}
