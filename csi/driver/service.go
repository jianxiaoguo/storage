package driver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/drycc/storage/csi/provider"
	"github.com/golang/glog"
)

type DriverInfo struct {
	NodeID      string
	DriverName  string
	Endpoint    string
	Components  string
	Version     string
	SavepointDB string
	HealthPort  int
}

type DriveService struct {
	driver     *CSIDriver
	provider   provider.Provider
	driverInfo *DriverInfo

	ids *IdentityServer
	ns  *NodeServer
	cs  *ControllerServer
}

// New initializes the driver
func New(driverInfo *DriverInfo, provider provider.Provider) (*DriveService, error) {
	d := NewCSIDriver(driverInfo.DriverName, driverInfo.Version, driverInfo.NodeID)
	if d == nil {
		glog.Fatalln("failed to initialize CSI Driver.")
	}
	service := &DriveService{driver: d, provider: provider, driverInfo: driverInfo}
	if err := service.initComponents(d); err != nil {
		return nil, err
	}
	return service, nil
}

func (service *DriveService) startHealthz(port int) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("ok\r\n"))
		}))
		server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
		server.ListenAndServe()
	}()
}

func (service *DriveService) initComponents(driver *CSIDriver) error {
	service.ids = &IdentityServer{driver: driver}
	for _, component := range strings.Split(service.driverInfo.Components, ",") {
		switch component {
		case "node":
			savepoint, err := NewSavepoint(service.driverInfo.SavepointDB)
			if err != nil {
				return err
			}
			service.ns = &NodeServer{provider: service.provider, driver: driver, savepoint: savepoint}
		case "controller":
			service.cs = &ControllerServer{provider: service.provider, driver: driver}
		default:
			return fmt.Errorf("unknown component: %s", component)
		}
	}
	return nil
}

func (service *DriveService) Serve() {
	glog.Infof("driver: %v ", service.driver.name)
	glog.Infof("version: %v ", service.driver.version)
	glog.Infof("components: %v ", service.driverInfo.Components)

	// Initialize default library driver
	service.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME})
	service.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER})

	// Create GRPC servers
	s := NewNonBlockingGRPCServer()
	s.Start(service.driverInfo.Endpoint, service.ids, service.cs, service.ns)
	service.startHealthz(service.driverInfo.HealthPort)
	s.Wait()
}

func (service *DriveService) Restore() error {
	return service.ns.savepoint.View(func(saveEntry *SaveEntry) {
		if err := service.provider.NodeMountVolume(&saveEntry.Point, &saveEntry.Bucket); err != nil {
			glog.Errorf("restore node mount error: %+v, %+v", saveEntry, err)
		}
	})
}
