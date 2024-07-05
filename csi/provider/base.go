package provider

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"k8s.io/mount-utils"
)

var mountUtils = mount.New("")
var globalProviders = make(map[string]Provider)

type MountPoint struct {
	Path     string   `json:"path"`
	Options  []string `json:"options"`
	Readonly bool     `json:"readonly"`
}

type MountBucket struct {
	Name     string            `json:"name"`
	Prefix   string            `json:"prefix"`
	Capacity uint64            `json:"capacity"`
	Secrets  map[string]string `json:"secrets"`
}

type Provider interface {
	SetFlags()
	NodeMountVolume(mountPoint *MountPoint, mountBucket *MountBucket) error
	NodeExpandVolume(bucket *MountBucket) error
	NodeUmountVolume(path string) error
	NodeWaitMountVolume(path string, timeout time.Duration) error
	NodeCheckMountVolume(path string) (bool, error)
	ControllerExpandVolume(mountBucket *MountBucket) error
}

func GetProvider(providerString string) (Provider, error) {
	provider := globalProviders[providerString]
	if provider == nil {
		return nil, fmt.Errorf("provider %v unimplemented", providerString)
	}
	return provider, nil
}

func AllSupports() []*Provider {
	var values []*Provider
	for _, value := range globalProviders {
		values = append(values, &value)
	}
	return values
}

type BaseProvider struct {
	Name        string
	Description string
}

func (provider *BaseProvider) NodeUmountVolume(path string) error {
	if err := mountUtils.Unmount(path); err != nil {
		return err
	}
	// as fuse quits immediately, we will try to wait until the process is done
	process, err := provider.findFuseMountProcess(path)
	if err != nil {
		glog.Errorf("error getting PID of fuse mount: %s", err)
		return nil
	}
	if process == nil {
		glog.Warningf("unable to find PID of fuse mount %s, it must have finished already", path)
		return nil
	}
	glog.Infof("found fuse pid %v of mount %s, checking if it still runs", process.Pid, path)
	return provider.waitForProcess(process, 20)
}

func (provider *BaseProvider) NodeWaitMountVolume(path string, timeout time.Duration) error {
	var elapsed time.Duration
	var interval = 10 * time.Millisecond
	for {
		mounted, err := mountUtils.IsMountPoint(path)
		if err != nil {
			return err
		}
		if mounted {
			return nil
		}
		time.Sleep(interval)
		elapsed = elapsed + interval
		if elapsed >= timeout {
			return errors.New("timeout waiting for mount")
		}
	}
}

func (provider *BaseProvider) NodeCheckMountVolume(path string) (bool, error) {
	notMnt, err := mountUtils.IsLikelyNotMountPoint(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path, 0750); err != nil {
				return false, err
			}
			notMnt = true
		} else {
			return false, err
		}
	}
	return notMnt, nil
}

func (provider *BaseProvider) findFuseMountProcess(path string) (*os.Process, error) {
	dirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	for _, file := range dirs {
		if file.IsDir() {
			if pid, err := strconv.Atoi(file.Name()); err == nil && pid != os.Getpid() {
				cmdLine, err := provider.getCmdLine(pid)
				if err != nil {
					glog.Errorf("unable to get cmdline of PID %v: %s", pid, err)
					continue
				}
				if strings.Contains(cmdLine, path) {
					glog.Infof("found matching pid %v on path %s", pid, path)
					return os.FindProcess(pid)
				}
			}
		}
	}
	return nil, nil
}

func (provider *BaseProvider) waitForProcess(p *os.Process, limit int) error {
	for backoff := 0; backoff < limit; backoff++ {
		cmdLine, err := provider.getCmdLine(p.Pid)
		if err != nil {
			glog.Warningf("error checking cmdline of PID %v, assuming it is dead: %s", p.Pid, err)
			p.Wait()
			return nil
		}
		if cmdLine == "" {
			glog.Warning("fuse process seems dead, returning")
			p.Wait()
			return nil
		}
		if err := p.Signal(syscall.Signal(0)); err != nil {
			glog.Warningf("fuse process does not seem active or we are unprivileged: %s", err)
			p.Wait()
			return nil
		}
		glog.Infof("fuse process with PID %v still active, waiting...", p.Pid)
		time.Sleep(time.Duration(math.Pow(1.5, float64(backoff))*100) * time.Millisecond)
	}
	p.Release()
	return fmt.Errorf("timeout waiting for PID %v to end", p.Pid)
}

func (provider *BaseProvider) getCmdLine(pid int) (string, error) {
	cmdLineFile := fmt.Sprintf("/proc/%v/cmdline", pid)
	cmdLine, err := os.ReadFile(cmdLineFile)
	if err != nil {
		return "", err
	}
	return string(cmdLine), nil
}
