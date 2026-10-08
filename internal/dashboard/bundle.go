package dashboard

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Only named regular binaries are extracted, so an archive cannot write paths
// outside staging or silently substitute a symlink for an executable.
func extractBundle(archive, directory string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	found := map[string]bool{}
	for {
		entry, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if entry.Name != "crontab-dashboard" && entry.Name != "cronitor" {
			continue
		}
		if entry.Typeflag != tar.TypeReg || entry.Size <= 0 || entry.Size > 150*1024*1024 || found[entry.Name] {
			return fmt.Errorf("invalid bundle entry %q", entry.Name)
		}
		out, err := os.OpenFile(filepath.Join(directory, entry.Name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, reader)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		found[entry.Name] = true
	}
	if !found["crontab-dashboard"] || !found["cronitor"] {
		return fmt.Errorf("release bundle must contain both dashboard and CLI")
	}
	return nil
}

func installBundle(directory, stage string) error {
	names := []string{"cronitor", "crontab-dashboard"}
	// Validate both before replacing either executable.
	for _, name := range names {
		if err := exec.Command(filepath.Join(stage, name), "--help").Run(); err != nil {
			return fmt.Errorf("invalid %s executable: %w", name, err)
		}
	}
	installed := []string{}
	backedUp := []string{}
	rollback := func() {
		for _, name := range installed {
			os.Remove(filepath.Join(directory, name))
		}
		for _, name := range backedUp {
			os.Rename(filepath.Join(stage, name+".old"), filepath.Join(directory, name))
		}
	}
	for _, name := range names {
		dest := filepath.Join(directory, name)
		if _, err := os.Lstat(dest); err == nil {
			if err := os.Rename(dest, filepath.Join(stage, name+".old")); err != nil {
				rollback()
				return err
			}
			backedUp = append(backedUp, name)
		} else if !os.IsNotExist(err) {
			rollback()
			return err
		}
		if err := os.Rename(filepath.Join(stage, name), dest); err != nil {
			rollback()
			return err
		}
		installed = append(installed, name)
	}
	return nil
}
