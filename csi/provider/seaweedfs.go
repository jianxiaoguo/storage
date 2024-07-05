package provider

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/golang/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/mount_pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	seaweedfsOpts seaweedfsOptions
)

func init() {
	name := "seaweedfs"
	globalProviders[name] = &seaweedfsProvider{
		BaseProvider: BaseProvider{
			Name:        name,
			Description: "SeaweedFS is a simple and highly scalable distributed file system.",
		},
	}
}

type seaweedfsOptions struct {
	Filer *string
}

type seaweedfsProvider struct {
	BaseProvider
}

func (provider *seaweedfsProvider) SetFlags() {
	seaweedfsOpts.Filer = flag.String("seaweedfs-filer", "localhost:8888", "comma-separated weed filer location")
}

func (provider *seaweedfsProvider) NodeMountVolume(mountPoint *MountPoint, mountBucket *MountBucket) error {
	command := "weed"
	sock := provider.getLocalSocket(mountBucket.Name, mountBucket.Prefix)
	kwargs := map[string]string{
		"dirAutoCreate":   "true",
		"umask":           "000",
		"filer":           *seaweedfsOpts.Filer,
		"filer.path":      fmt.Sprintf("/buckets/%s/%s", mountBucket.Name, mountBucket.Prefix),
		"collection":      mountBucket.Name,
		"dir":             mountPoint.Path,
		"localSocket":     sock,
		"cacheDir":        os.TempDir(),
		"cacheCapacityMB": "100",
	}
	args := []string{"-logtostderr=true", "mount"}
	for k, v := range kwargs {
		if v != "" {
			args = append(args, fmt.Sprintf("-%s=%s", k, v))
		}
	}
	if mountPoint.Readonly {
		args = append(args, "-readOnly")
	}
	args = append(args, mountPoint.Options...)
	cmd := exec.Command(command, args...)
	glog.V(0).Infof("Mounting fuse with command: %s and args: %s", command, args)

	// log fuse process messages - we need an easy way to investigate crashes in case it happens
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Start()
	if err != nil {
		glog.Errorf("running weed mount: %v", err)
		return fmt.Errorf("error fuseMount command: %s\nargs: %s\nerror: %v", command, args, err)
	}

	// avoid zombie processes
	go func() {
		if err := cmd.Wait(); err != nil {
			glog.Errorf("weed mount exit, pid: %d, path: %v, error: %v", cmd.Process.Pid, mountPoint.Path, err)
		} else {
			glog.Infof("weed mount exit, pid: %d, path: %v", cmd.Process.Pid, mountPoint.Path)
		}
		// make sure we'll have no stale mounts
		time.Sleep(time.Millisecond * 100)
		mountUtils.Unmount(mountPoint.Path)
	}()
	err = provider.NodeWaitMountVolume(mountPoint.Path, 10*time.Second)
	if err == nil && mountBucket.Capacity > 0 {
		return provider.quota(sock, int64(mountBucket.Capacity))
	}
	return err
}

func (provider *seaweedfsProvider) NodeExpandVolume(mountBucket *MountBucket) error {
	glog.V(3).Infof("node expand volume %+v", mountBucket)
	sock := provider.getLocalSocket(mountBucket.Name, mountBucket.Prefix)
	return provider.quota(sock, int64(mountBucket.Capacity))
}

func (provider *seaweedfsProvider) ControllerExpandVolume(mountBucket *MountBucket) error {
	glog.V(3).Infof("controller expand volume %+v", mountBucket)
	return nil
}

func (vol *seaweedfsProvider) quota(localSocket string, sizeByte int64) error {
	target := fmt.Sprintf("passthrough:///unix://%s", localSocket)
	dialOption := grpc.WithTransportCredentials(insecure.NewCredentials())

	clientConn, err := grpc.NewClient(target, dialOption)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	// We can't create PV of zero size, so we're using quota of 1 byte to define no quota.
	if sizeByte == 1 {
		sizeByte = 0
	}

	client := mount_pb.NewSeaweedMountClient(clientConn)
	_, err = client.Configure(context.Background(), &mount_pb.ConfigureRequest{
		CollectionCapacity: sizeByte,
	})
	return err
}

func (provider *seaweedfsProvider) getLocalSocket(bucket, prefix string) string {
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%s-%s", bucket, prefix)))
	b := h.Sum(nil)
	socket := fmt.Sprintf("/tmp/seaweedfs-mount-%d.sock", binary.BigEndian.Uint64(b))
	return socket
}
