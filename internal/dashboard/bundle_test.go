package dashboard

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBundleRejectsMissingDuplicateAndLinkedExecutables(t *testing.T) {
	for _, kind := range []string{"missing", "duplicate", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, "bundle.tar.gz")
			file, _ := os.Create(archive)
			gz := gzip.NewWriter(file)
			writer := tar.NewWriter(gz)
			entry := func(name string, typ byte) {
				data := []byte("binary")
				header := &tar.Header{Name: name, Typeflag: typ, Size: int64(len(data))}
				if typ == tar.TypeSymlink {
					header.Linkname = "/bin/sh"
					header.Size = 0
				}
				if err := writer.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
				if typ == tar.TypeReg {
					writer.Write(data)
				}
			}
			entry("crontab-dashboard", tar.TypeReg)
			if kind == "duplicate" {
				entry("crontab-dashboard", tar.TypeReg)
			}
			if kind == "symlink" {
				entry("cronitor", tar.TypeSymlink)
			}
			writer.Close()
			gz.Close()
			file.Close()
			out := filepath.Join(dir, "out")
			os.Mkdir(out, 0755)
			if err := extractBundle(archive, out); err == nil {
				t.Fatalf("accepted %s bundle", kind)
			}
		})
	}
}

func TestBundleInstallRollsBackBothBinariesOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures")
	}
	dir := t.TempDir()
	stage := filepath.Join(dir, "stage")
	os.Mkdir(stage, 0755)
	for _, name := range []string{"cronitor", "crontab-dashboard"} {
		os.WriteFile(filepath.Join(dir, name), []byte("old-"+name), 0755)
		body := "#!/bin/sh\nexit 0\n"
		if name == "crontab-dashboard" {
			body = "#!/bin/sh\nrm -- \"$0\"\nexit 0\n"
		}
		os.WriteFile(filepath.Join(stage, name), []byte(body), 0755)
	}
	if err := installBundle(dir, stage); err == nil {
		t.Fatal("install unexpectedly succeeded")
	}
	for _, name := range []string{"cronitor", "crontab-dashboard"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(data) != "old-"+name {
			t.Fatalf("%s was not restored: %s %v", name, data, err)
		}
	}
}
