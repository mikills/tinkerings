package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// RepoSpec represents something like:
// - "user/repo"
// - "github:user/repo"
// - "github:user/repo/subdir"
type RepoSpec struct {
	Host   string // e.g. "github.com"
	Owner  string
	Repo   string
	Subdir string // optional subdirectory inside the repo
	Ref    string // branch/tag/commit, default "HEAD"
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <repo> <target-dir>\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), `
				Examples:
				  %[1]s user/repo my-app
				  %[1]s github:user/repo my-app
				  %[1]s github:user/repo/subdir my-app

				This will download the repository contents (like degit) without git history.
				`,
			os.Args[0])
	}
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	repoStr := flag.Arg(0)
	targetDir := flag.Arg(1)

	if err := run(repoStr, targetDir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(repoStr, targetDir string) error {
	spec, err := parseRepoSpec(repoStr)
	if err != nil {
		return fmt.Errorf("parse repo: %w", err)
	}

	if spec.Host != "github.com" {
		return fmt.Errorf("unsupported host %q (only github.com supported for now)", spec.Host)
	}

	archiveURL := buildGitHubArchiveURL(spec)
	fmt.Printf("Fetching from %s\n", archiveURL)

	resp, err := http.Get(archiveURL)
	if err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status downloading archive: %s", resp.Status)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	if err := extractTarGz(resp.Body, targetDir, spec.Subdir); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	fmt.Printf("✔ Done. Files written to %s\n", targetDir)
	return nil
}

// parseRepoSpec parses strings like:
// - "user/repo"
// - "github:user/repo"
// - "github:user/repo/subdir"
func parseRepoSpec(input string) (*RepoSpec, error) {
	spec := &RepoSpec{
		Host: "github.com",
		Ref:  "HEAD",
	}

	// Optional prefix like "github:"
	if strings.Contains(input, ":") {
		parts := strings.SplitN(input, ":", 2)
		prefix := parts[0]
		rest := parts[1]

		switch prefix {
		case "github":
			spec.Host = "github.com"
		default:
			return nil, fmt.Errorf("unsupported prefix %q", prefix)
		}
		input = rest
	}

	// Now we expect "owner/repo" or "owner/repo/subdir"
	parts := strings.Split(input, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid repo spec %q, expected owner/repo", input)
	}

	spec.Owner = parts[0]
	spec.Repo = parts[1]

	if len(parts) > 2 {
		spec.Subdir = path.Join(parts[2:]...)
	}

	return spec, nil
}

// buildGitHubArchiveURL builds a URL like:
// https://codeload.github.com/<owner>/<repo>/tar.gz/<ref>
func buildGitHubArchiveURL(spec *RepoSpec) string {
	ref := spec.Ref
	if ref == "" {
		ref = "HEAD"
	}
	return fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s",
		spec.Owner, spec.Repo, ref)
}

// extractTarGz extracts a .tar.gz from r into targetDir.
// It assumes the archive has a leading top-level directory (GitHub does: repo-<hash>/).
// If subdir is non-empty, only that subtree is extracted into targetDir.
func extractTarGz(r io.Reader, targetDir, subdir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var rootDir string
	if subdir != "" {
		// normalise: remove any leading slashes
		subdir = strings.TrimLeft(subdir, "/")
	}

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir, tar.TypeReg, tar.TypeSymlink:
			// Continue
		default:
			// Skip unusual entries
			continue
		}

		name := header.Name

		// GitHub archives always have "repo-ref/" prefix.
		// Capture it on the first entry.
		if rootDir == "" {
			// rootDir is the first path segment, e.g. "repo-abc123"
			rootDir = strings.SplitN(name, "/", 2)[0]
		}

		// Strip that rootDir prefix
		rel := strings.TrimPrefix(name, rootDir)
		rel = strings.TrimPrefix(rel, "/") // remove a possible leading slash

		if rel == "" {
			// the root directory entry itself
			continue
		}

		// If subdir is specified, filter everything outside it
		if subdir != "" {
			if !strings.HasPrefix(rel, subdir+"/") && rel != subdir {
				// not in the desired subtree
				continue
			}
			// Strip subdir so that files are written as if subdir is the root
			rel = strings.TrimPrefix(rel, subdir)
			rel = strings.TrimPrefix(rel, "/")
			if rel == "" {
				// picking the directory itself; nothing to write yet
				continue
			}
		}

		targetPath := filepath.Join(targetDir, filepath.FromSlash(rel))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("mkdir %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", filepath.Dir(targetPath), err)
			}
			if err := writeFileFromTar(tr, targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Optional: handle symlinks. GitHub does include them in archives.
			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", filepath.Dir(targetPath), err)
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("create symlink %s -> %s: %w", targetPath, header.Linkname, err)
			}
		}
	}

	return nil
}

// writeFileFromTar copies the current file entry from tar.Reader to disk.
func writeFileFromTar(tr *tar.Reader, targetPath string, mode os.FileMode) error {
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create file %s: %w", targetPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, tr); err != nil {
		return fmt.Errorf("write file %s: %w", targetPath, err)
	}
	return nil
}
