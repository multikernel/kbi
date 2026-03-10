package oci

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/authn"
)

func Push(ref string, img v1.Image) error {
	r, err := name.ParseReference(ref)
	if err != nil {
		return fmt.Errorf("parsing reference %s: %w", ref, err)
	}
	if err := remote.Write(r, img, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
		return fmt.Errorf("pushing %s: %w", ref, err)
	}
	return nil
}
