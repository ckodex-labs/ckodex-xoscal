package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"dagger/xoscal/internal/dagger"
)

var validAptPackageName = regexp.MustCompile(`^[a-z0-9][a-z0-9.+-]*$`)

// withAptPackages installs packages inside c using locked APT caches keyed by
// the SHA-256 digest of imageRef so heterogeneous base images never share lists.
func withAptPackages(c *dagger.Container, imageRef string, pkgs ...string) *dagger.Container {
	if len(pkgs) == 0 {
		panic("withAptPackages requires at least one package name")
	}
	for _, pkg := range pkgs {
		if !validAptPackageName.MatchString(pkg) {
			panic(fmt.Sprintf("invalid apt package name: %q", pkg))
		}
	}
	sum := sha256.Sum256([]byte(imageRef))
	key := hex.EncodeToString(sum[:8])
	aptCache := dag.CacheVolume("apt-cache-" + key)
	aptLists := dag.CacheVolume("apt-lists-" + key)

	return c.
		WithEnvVariable("DEBIAN_FRONTEND", "noninteractive").
		WithMountedCache("/var/cache/apt", aptCache, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModeLocked}).
		WithMountedCache("/var/lib/apt/lists", aptLists, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModeLocked}).
		WithExec([]string{"sh", "-ec", "rm -f /etc/apt/apt.conf.d/docker-clean && apt-get update && apt-get install -y --no-install-recommends " + strings.Join(pkgs, " ")}).
		WithoutMount("/var/cache/apt").
		WithoutMount("/var/lib/apt/lists")
}
