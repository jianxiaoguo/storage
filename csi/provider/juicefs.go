package provider

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/golang/glog"
)

var (
	juicefsOpts juicefsOptions
)

func init() {
	name := "juicefs"
	globalProviders[name] = &juicefsProvider{
		BaseProvider: BaseProvider{
			Name:        name,
			Description: "JuiceFS is a distributed POSIX file system built on top of S3 API.",
		},
	}
}

type juicefsOptions struct {
	Name      *string
	MetaURL   *string
	BlockSize *uint64
	TrashDays *uint64
}

type juicefsProvider struct {
	BaseProvider
}

func (provider *juicefsProvider) SetFlags() {
	juicefsOpts.Name = flag.String("juicefs-name", "drycc", "name is the prefix of all objects in data storage")
	juicefsOpts.MetaURL = flag.String("juicefs-meta-url", "", "meta-url is used to set up the metadata engine")
	juicefsOpts.BlockSize = flag.Uint64("juicefs-block-size", 4096, "size of block in KiB")
	juicefsOpts.TrashDays = flag.Uint64("juicefs-trash-days", 0, "number of days after which removed files will be permanently deleted")
}

func (provider *juicefsProvider) NodeMountVolume(mountPoint *MountPoint, mountBucket *MountBucket) error {
	metaURL, err := provider.formatJuicefs(mountBucket.Name, mountBucket.Prefix, mountBucket.Capacity, mountBucket.Secrets)
	if err != nil {
		return err
	}
	args := []string{
		"mount",
		metaURL,
		mountPoint.Path,
		"--background",
	}
	if mountPoint.Readonly {
		args = append(args, "--read-only")
	}
	args = append(args, mountPoint.Options...)
	cmd := exec.Command("juicefs", args...)
	cmd.Stderr = os.Stderr
	glog.V(3).Infof("juicefs format with command: %s and args: %s", "juicefs", args)

	if out, err := cmd.Output(); err != nil {
		return fmt.Errorf("error exec command: %s\nargs: %s\noutput: %s", "juicefs", args, out)
	}

	return provider.NodeWaitMountVolume(mountPoint.Path, 10*time.Second)
}

func (provider *juicefsProvider) NodeExpandVolume(mountBucket *MountBucket) error {
	glog.V(3).Infof("node expand volume mount bucket %+v", mountBucket)
	return nil
}

func (provider *juicefsProvider) ControllerExpandVolume(mountBucket *MountBucket) error {
	inode := mountBucket.Capacity / *juicefsOpts.BlockSize
	metaURL := fmt.Sprintf("%s/%s/%s", *juicefsOpts.MetaURL, mountBucket.Name, mountBucket.Prefix)
	args := []string{
		"config",
		metaURL,
		"--inodes", strconv.FormatUint(inode, 10),
		"--capacity", strconv.FormatUint(provider.formatCapacity(mountBucket.Capacity), 10),
	}
	cmd := exec.Command("juicefs", args...)

	cmd.Stderr = os.Stderr
	glog.V(3).Infof("juicefs config with command: %s and args: %s and secrets: %s", "juicefs", args, mountBucket.Secrets)

	input, e := cmd.StdinPipe()
	defer func() {
		if err := input.Close(); err != nil {
			glog.V(3).Infof("juicefs close cmd error: %v", err)
		}
	}()
	if e != nil {
		return e
	}
	input.Write([]byte("n"))

	if out, err := cmd.Output(); err != nil {
		return fmt.Errorf("error exec command: %s\nargs: %s\noutput: %s", "juicefs", args, out)
	}
	return nil
}

func (provider *juicefsProvider) formatCapacity(capacity uint64) uint64 {
	capacity = capacity / (1024 * 1024 * 1024)
	if capacity < 1 {
		return 1
	}
	return capacity
}

func (provider *juicefsProvider) formatJuicefs(bucket, prefix string, capacity uint64, secrets map[string]string) (string, error) {
	endpoint := secrets["endpoint"]
	accessKey := secrets["accesskey"]
	secretKey := secrets["secretkey"]
	inode := capacity / *juicefsOpts.BlockSize
	metaURL := fmt.Sprintf("%s/%s/%s", *juicefsOpts.MetaURL, bucket, prefix)

	if out, err := exec.Command("juicefs", []string{"status", metaURL}...).Output(); err == nil {
		glog.V(3).Infof("%s has been formatted: %s", "juicefs", out)
		return metaURL, nil
	}

	args := []string{
		"format",
		"--inodes", strconv.FormatUint(inode, 10),
		"--block-size", strconv.FormatUint(*juicefsOpts.BlockSize, 10),
		"--capacity", strconv.FormatUint(provider.formatCapacity(capacity), 10),
		"--trash-days", strconv.FormatUint(*juicefsOpts.TrashDays, 10),
		"--storage", "s3",
		"--bucket", fmt.Sprintf("%s/%s/%s", endpoint, bucket, prefix),
		"--access-key", accessKey,
		"--secret-key", secretKey,
		metaURL,
		*juicefsOpts.Name,
	}
	cmd := exec.Command("juicefs", args...)
	cmd.Stderr = os.Stderr
	glog.V(3).Infof("juicefs format with command: %s and args: %s", "juicefs", args)
	if out, err := cmd.Output(); err != nil {
		return metaURL, fmt.Errorf("error exec command: %s\nargs: %s\noutput: %s", "juicefs", args, out)
	}
	return metaURL, nil
}
