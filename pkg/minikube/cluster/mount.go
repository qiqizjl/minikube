/*
Copyright 2019 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// MountConfig defines the options available to the Mount command
type MountConfig struct {
	// Type is the filesystem type (Typically 9p)
	Type string
	// UID is the User ID which this path will be mounted as
	UID int
	// GID is the Group ID which this path will be mounted as
	GID int
	// Version is the 9P protocol version. Valid options: 9p2000, 9p200.u, 9p2000.L
	Version string
	// MSize is the number of bytes to use for 9p packet payload
	MSize int
	// Port is the port to connect to on the host
	Port int
	// Mode is the file permissions to set the mount to (octals)
	Mode os.FileMode
	// Extra mount options. See https://www.kernel.org/doc/Documentation/filesystems/9p.txt
	Options map[string]string
}

// hostRunner is the subset of host.Host used for mounting
type hostRunner interface {
	RunSSHCommand(cmd string) (string, error)
}

// Mount runs the mount command from the 9p client on the VM to the 9p server on the host
func Mount(h hostRunner, source string, target string, c *MountConfig) error {
	if err := Unmount(h, target); err != nil {
		return errors.Wrap(err, "umount")
	}

	cmd := fmt.Sprintf("sudo mkdir -m %o -p %s && %s", c.Mode, target, mntCmd(source, target, c))
	out, err := h.RunSSHCommand(cmd)
	if err != nil {
		return errors.Wrap(err, out)
	}
	return nil
}

// mntCmd returns a mount command based on a config.
func mntCmd(source string, target string, c *MountConfig) string {
	options := map[string]string{
		"dfltgid": strconv.Itoa(c.GID),
		"dfltuid": strconv.Itoa(c.UID),
	}
	if c.Port != 0 {
		options["port"] = strconv.Itoa(c.Port)
	}
	if c.Version != "" {
		options["version"] = c.Version
	}
	if c.MSize != 0 {
		options["msize"] = strconv.Itoa(c.MSize)
	}

	// Copy in all of the user-supplied keys and values
	for k, v := range c.Options {
		options[k] = v
	}

	// Convert everything into a sorted list for better test results
	opts := []string{}
	for k, v := range options {
		// Mount option with no value, such as "noextend"
		if v == "" {
			opts = append(opts, k)
			continue
		}
		opts = append(opts, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(opts)
	return fmt.Sprintf("sudo mount -t %s -o %s %s %s", c.Type, strings.Join(opts, ","), source, target)
}

// Unmount unmounts a path
func Unmount(h hostRunner, target string) error {
	out, err := h.RunSSHCommand(fmt.Sprintf("findmnt -T %s && sudo umount %s || true", target, target))
	if err != nil {
		return errors.Wrap(err, out)
	}
	return nil
}
