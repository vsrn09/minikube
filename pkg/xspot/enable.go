/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package xspot

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/pkg/errors"
	"k8s.io/minikube/pkg/minikube/assets"
	"k8s.io/minikube/pkg/minikube/constants"
	"k8s.io/minikube/pkg/minikube/vmpath"
)

const (
	nodeDir                        = "/node"
	containerdConfigTomlPath       = "/etc/containerd/config.toml"
	storedContainerdConfigTomlPath = "/tmp/config.toml"
	xspotContainerdShimURL        = "https://github.com/google/gvisor-containerd-shim/releases/download/v0.0.3/containerd-shim-runsc-v1.linux-amd64"
	xspotURL                      = "https://venkat-xspot-bucket.s3.us-east-2.amazonaws.com/xspot/kata-runtime"
)

// Enable follows these steps for enabling xspot in minikube:
//   1. creates necessary directories for storing binaries and runxc logs
//   2. downloads runxc and xspot-containerd-shim
//   3. copies necessary containerd config files
//   4. restarts containerd
func Enable() error {
	if err := makeXSpotDirs(); err != nil {
		return errors.Wrap(err, "creating directories on node")
	}
	if err := downloadBinaries(); err != nil {
		return errors.Wrap(err, "downloading binaries")
	}
	if err := copyConfigFiles(); err != nil {
		return errors.Wrap(err, "copying config files")
	}
	if err := restartContainerd(); err != nil {
		return errors.Wrap(err, "restarting containerd")
	}
	// When pod is terminated, disable xspot and exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if err := Disable(); err != nil {
			log.Printf("Error disabling xspot: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	log.Print("xspot successfully enabled in cluster")
	// sleep for one year so the pod continuously runs
	select {}
}

// makeXSpotDirs creates necessary directories on the node
func makeXSpotDirs() error {
	// Make /run/containerd/runxc to hold logs
	fp := filepath.Join(nodeDir, "run/containerd/runxc")
	if err := os.MkdirAll(fp, 0755); err != nil {
		return errors.Wrap(err, "creating runxc dir")
	}

	// Make /tmp/runxc to also hold logs
	fp = filepath.Join(nodeDir, "tmp/runxc")
	if err := os.MkdirAll(fp, 0755); err != nil {
		return errors.Wrap(err, "creating runxc logs dir")
	}

	return nil
}

func downloadBinaries() error {
	if err := runxc(); err != nil {
		return errors.Wrap(err, "downloading runxc")
	}
	if err := xspotContainerdShim(); err != nil {
		return errors.Wrap(err, "downloading xspot-containerd-shim")
	}
	return nil
}

// downloads the xspot-containerd-shim
func xspotContainerdShim() error {
	dest := filepath.Join(nodeDir, "usr/bin/containerd-shim-runsc-v1")
	return downloadFileToDest(xspotContainerdShimURL, dest)
}

// downloads the runxc binary and returns a path to the binary
func runxc() error {
	dest := filepath.Join(nodeDir, "usr/bin/runxc")
	return downloadFileToDest(xspotURL, dest)
}

// downloadFileToDest downloads the given file to the dest
// if something already exists at dest, first remove it
func downloadFileToDest(url, dest string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errors.Wrapf(err, "creating request for %s", url)
	}
	req.Header.Set("User-Agent", "minikube")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if _, err := os.Stat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			return errors.Wrapf(err, "removing %s for overwrite", dest)
		}
	}
	fi, err := os.Create(dest)
	if err != nil {
		return errors.Wrapf(err, "creating %s", dest)
	}
	defer fi.Close()
	if _, err := io.Copy(fi, resp.Body); err != nil {
		return errors.Wrap(err, "copying binary")
	}
	if err := fi.Chmod(0777); err != nil {
		return errors.Wrap(err, "fixing perms")
	}
	return nil
}

// Must write the following files:
//    1. xspot-containerd-shim.toml
//    2. xspot containerd config.toml
// and save the default version of config.toml
func copyConfigFiles() error {
	log.Printf("Storing default config.toml at %s", storedContainerdConfigTomlPath)
	if err := mcnutils.CopyFile(filepath.Join(nodeDir, containerdConfigTomlPath), filepath.Join(nodeDir, storedContainerdConfigTomlPath)); err != nil {
		return errors.Wrap(err, "copying default config.toml")
	}
	log.Printf("Copying %s asset to %s", constants.XSpotConfigTomlTargetName, filepath.Join(nodeDir, containerdConfigTomlPath))
	if err := copyAssetToDest(constants.XSpotConfigTomlTargetName, filepath.Join(nodeDir, containerdConfigTomlPath)); err != nil {
		return errors.Wrap(err, "copying xspot version of config.toml")
	}
	return nil
}

func copyAssetToDest(targetName, dest string) error {
	var asset *assets.BinAsset
	for _, a := range assets.Addons["xspot"].Assets {
		if a.GetTargetName() == targetName {
			asset = a
		}
	}
	if asset == nil {
		return fmt.Errorf("no asset matching target %s among %+v", targetName, assets.Addons["xspot"])
	}

	// Now, copy the data from this asset to dest
	src := filepath.Join(vmpath.GuestXSpotDir, asset.GetTargetName())
	log.Printf("%s asset path: %s", targetName, src)
	contents, err := os.ReadFile(src)
	if err != nil {
		return errors.Wrapf(err, "getting contents of %s", asset.GetSourcePath())
	}
	if _, err := os.Stat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			return errors.Wrapf(err, "removing %s", dest)
		}
	}

	log.Printf("creating %s", dest)
	f, err := os.Create(dest)
	if err != nil {
		return errors.Wrapf(err, "creating %s", dest)
	}
	if _, err := f.Write(contents); err != nil {
		return errors.Wrapf(err, "writing contents to %s", f.Name())
	}
	return nil
}

func restartContainerd() error {
	log.Print("restartContainerd black magic happening")

	log.Print("Stopping rpc-statd.service...")
	cmd := exec.Command("/usr/sbin/chroot", "/node", "sudo", "systemctl", "stop", "rpc-statd.service")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(string(out))
		return errors.Wrap(err, "stopping rpc-statd.service")
	}

	log.Print("Restarting containerd...")
	cmd = exec.Command("/usr/sbin/chroot", "/node", "sudo", "systemctl", "restart", "containerd")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Print(string(out))
		return errors.Wrap(err, "restarting containerd")
	}

	log.Print("Starting rpc-statd...")
	cmd = exec.Command("/usr/sbin/chroot", "/node", "sudo", "systemctl", "start", "rpc-statd.service")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Print(string(out))
		return errors.Wrap(err, "restarting rpc-statd.service")
	}
	log.Print("containerd restart complete")
	return nil
}
