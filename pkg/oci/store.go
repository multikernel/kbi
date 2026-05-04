package oci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/validate"
)

// Store manages KBI images stored locally in OCI image layout format.
type Store struct {
	root string
}

// NewStore creates a Store rooted at the given directory.
func NewStore(root string) *Store {
	return &Store{root: root}
}

// DefaultStore returns a Store rooted at ~/.kbi.
func DefaultStore() *Store {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return NewStore(filepath.Join(home, ".kbi"))
}

// Save writes an OCI image to local storage under the given reference.
// Re-saving the same ref replaces any prior content so the OCI index never
// accumulates stale manifests across rebuilds or pulls.
func (s *Store) Save(ref string, img v1.Image) error {
	imgPath := s.refToPath(ref)

	lock, err := s.lock()
	if err != nil {
		return err
	}
	defer lock.Close()

	// Clear any prior image at this path. layout.Write + AppendImage append to
	// existing index.json, so without this a second Save would leave the first
	// manifest in place and Load would return the stale one.
	if err := os.RemoveAll(imgPath); err != nil {
		return fmt.Errorf("clearing image directory: %w", err)
	}
	if err := os.MkdirAll(imgPath, 0700); err != nil {
		return fmt.Errorf("creating image directory: %w", err)
	}

	lp, err := layout.Write(imgPath, empty.Index)
	if err != nil {
		return fmt.Errorf("initializing OCI layout: %w", err)
	}

	if err := lp.AppendImage(img); err != nil {
		return fmt.Errorf("appending image: %w", err)
	}

	if err := s.updateIndexLocked(ref, imgPath); err != nil {
		return fmt.Errorf("updating index: %w", err)
	}

	return nil
}

// Load reads an OCI image from local storage for the given reference.
func (s *Store) Load(ref string) (v1.Image, error) {
	imgPath := s.refToPath(ref)

	lp, err := layout.FromPath(imgPath)
	if err != nil {
		return nil, fmt.Errorf("image %q not found: %w", ref, err)
	}

	idx, err := lp.ImageIndex()
	if err != nil {
		return nil, fmt.Errorf("reading image index: %w", err)
	}

	manifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("reading index manifest: %w", err)
	}

	if len(manifest.Manifests) == 0 {
		return nil, fmt.Errorf("image %q has no manifests", ref)
	}

	digest := manifest.Manifests[0].Digest
	img, err := idx.Image(digest)
	if err != nil {
		return nil, fmt.Errorf("loading image: %w", err)
	}

	// Verify layer digests match the manifest to detect tampering.
	if err := validate.Image(img, validate.Fast); err != nil {
		return nil, fmt.Errorf("integrity check failed for %q: %w", ref, err)
	}

	return img, nil
}

// Exists reports whether the image for the given reference exists locally.
func (s *Store) Exists(ref string) bool {
	imgPath := s.refToPath(ref)
	_, err := os.Stat(filepath.Join(imgPath, "index.json"))
	return err == nil
}

// refToPath converts a reference like "registry.io/org/kernel:tag" to
// a local path like "<root>/images/registry.io/org/kernel/tag".
// Uses go-containerregistry's name.ParseReference for correct parsing
// of references with ports (e.g. localhost:5000/foo:bar) and digests.
func (s *Store) refToPath(ref string) string {
	parsed, err := name.ParseReference(ref)
	if err != nil {
		// Reject unparseable references to prevent path traversal
		// via crafted refs containing "../".
		return filepath.Join(s.root, "images", "_invalid")
	}
	// Use registry/repository/identifier as path
	registry := parsed.Context().RegistryStr()
	repo := parsed.Context().RepositoryStr()
	identifier := parsed.Identifier()
	return filepath.Join(s.root, "images", registry, repo, identifier)
}

// Remove deletes an image from local storage and removes it from the index.
func (s *Store) Remove(ref string) error {
	imgPath := s.refToPath(ref)

	lock, err := s.lock()
	if err != nil {
		return err
	}
	defer lock.Close()

	if _, err := os.Stat(imgPath); err != nil {
		return fmt.Errorf("image %q not found", ref)
	}

	if err := os.RemoveAll(imgPath); err != nil {
		return fmt.Errorf("removing image directory: %w", err)
	}

	indexPath := filepath.Join(s.root, "kbi.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil // index missing is not an error
	}

	var idx kbiIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil
	}

	filtered := idx.Images[:0]
	for _, entry := range idx.Images {
		if entry.Ref != ref {
			filtered = append(filtered, entry)
		}
	}
	idx.Images = filtered

	out, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	return os.WriteFile(indexPath, out, 0600)
}

// List returns all image references stored locally.
func (s *Store) List() ([]string, error) {
	indexPath := filepath.Join(s.root, "kbi.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading index: %w", err)
	}

	var idx kbiIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}

	refs := make([]string, len(idx.Images))
	for i, entry := range idx.Images {
		refs[i] = entry.Ref
	}
	return refs, nil
}

// lock acquires an exclusive flock on the store's kbi.lock file. Held across
// the full Save/Remove operation so layout writes and the kbi.json index stay
// consistent under concurrent invocations.
func (s *Store) lock() (*os.File, error) {
	lockPath := filepath.Join(s.root, "kbi.lock")
	if err := os.MkdirAll(s.root, 0700); err != nil {
		return nil, fmt.Errorf("creating store root: %w", err)
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}
	return f, nil
}

// kbiIndex is the structure for the kbi.json index file.
type kbiIndex struct {
	Images []kbiIndexEntry `json:"images"`
}

type kbiIndexEntry struct {
	Ref  string `json:"ref"`
	Path string `json:"path"`
}

// updateIndexLocked maintains kbi.json. The caller must hold the store lock.
func (s *Store) updateIndexLocked(ref, path string) error {
	indexPath := filepath.Join(s.root, "kbi.json")

	var idx kbiIndex
	data, err := os.ReadFile(indexPath)
	if err == nil {
		_ = json.Unmarshal(data, &idx)
	}

	found := false
	for i, entry := range idx.Images {
		if entry.Ref == ref {
			idx.Images[i].Path = path
			found = true
			break
		}
	}
	if !found {
		idx.Images = append(idx.Images, kbiIndexEntry{Ref: ref, Path: path})
	}

	out, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}

	return os.WriteFile(indexPath, out, 0600)
}
