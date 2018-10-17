package container // import "github.com/docker/docker/daemon/cluster/executor/container"

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/docker/swarmkit/api"
)

func validateMounts(mounts []api.Mount) error {
	for _, mount := range mounts {
		// Target must always be absolute
		// except if target is Windows named pipe
		if !filepath.IsAbs(mount.Target) && mount.Type != api.MountTypeNamedPipe {
			return fmt.Errorf("invalid mount target, must be an absolute path: %s", mount.Target)
		}

		switch mount.Type {
		// The checks on abs paths are required due to the container API confusing
		// volume mounts as bind mounts when the source is absolute (and vice-versa)
		// See #25253
		// TODO: This is probably not necessary once #22373 is merged
		case api.MountTypeBind:
			if !validateMountAbs(mount.Source) {
				return fmt.Errorf("invalid bind mount source, must be an absolute path: %s", mount.Source)
			}
		case api.MountTypeVolume:
			if validateMountAbs(mount.Source) {
				return fmt.Errorf("invalid volume mount source, must not be an absolute path: %s", mount.Source)
			}
		case api.MountTypeTmpfs:
			if mount.Source != "" {
				return errors.New("invalid tmpfs source, source must be empty")
			}
		default:
			return fmt.Errorf("invalid mount type: %s", mount.Type)
		}
	}
	return nil
}

func validateMountAbs(path string) bool {
	// First attempt to validate with filepath
	if !filepath.IsAbs(path) {
		// If the platform is not Windows but the given path contains a non-zero
		// length Windows volumeName, validate it with windowsPathAbs
		if runtime.GOOS != "windows" && len(path[:volumeNameLen(path)]) > 0 {
			if !windowsPathAbs(path) {
				return false
			}
		}
		return false
	}
	return true
}

// windowsPathAbs reports whether the path is absolute for a Windows path.
// it is adoped from the Windows implementation of filepath.IsAbs
func windowsPathAbs(path string) (b bool) {
	l := volumeNameLen(path)
	if l == 0 {
		return false
	}
	path = path[l:]
	if path == "" {
		return false
	}
	return isSlash(path[0])
}

// volumeNameLen returns length of the leading volume name for Windows paths
// it is adopted from the Windows implementation of filepath/path
func volumeNameLen(path string) int {
	if len(path) < 2 {
		return 0
	}
	// with drive letter
	c := path[0]
	if path[1] == ':' && ('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
		return 2
	}
	// is it UNC? https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	if l := len(path); l >= 5 && isSlash(path[0]) && isSlash(path[1]) &&
		!isSlash(path[2]) && path[2] != '.' {
		// first, leading `\\` and next shouldn't be `\`. its server name.
		for n := 3; n < l-1; n++ {
			// second, next '\' shouldn't be repeated.
			if isSlash(path[n]) {
				n++
				// third, following something characters. its share name.
				if !isSlash(path[n]) {
					if path[n] == '.' {
						break
					}
					for ; n < l; n++ {
						if isSlash(path[n]) {
							break
						}
					}
					return n
				}
				break
			}
		}
	}
	return 0
}

func isSlash(c uint8) bool {
	return c == '\\' || c == '/'
}
