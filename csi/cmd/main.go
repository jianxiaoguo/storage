package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/drycc/storage/csi/driver"
	"github.com/drycc/storage/csi/provider"
	"github.com/golang/glog"
)

var version = "v1.2.0"

var usageTemplate = `
Start a non blocking grpc server for csi.

Usage: {{.Name}} <provider> [options]

Arguments:
  <provider>
    container storage interface for provider, current support:
      {{- range $provider := .Providers}}
      {{$provider.Name}}: {{$provider.Description}}
      {{- end}}

Use '{{.Name}} <provider> --help' to learn more.
`

func getProvider() *provider.Provider {
	content := struct {
		Name      string
		Providers []*provider.Provider
	}{
		Name:      os.Args[0],
		Providers: provider.AllSupports(),
	}
	var text bytes.Buffer
	t := template.Must(template.New("csi").Parse(usageTemplate))
	if err := t.Execute(&text, content); err != nil {
		log.Fatal(err)
	}

	if len(os.Args) == 1 {
		fmt.Println(text.String())
		os.Exit(0)
	}
	backend := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...)
	provider, err := provider.GetProvider(backend)
	if err != nil {
		fmt.Println(text.String())
		os.Exit(0)
	}
	return &provider
}

func main() {
	provider := *getProvider()
	nodeID := flag.String("node-id", "", "node id")
	endpoint := flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint to accept gRPC calls")
	components := flag.String("components", "controller,node", "components to run, by default both controller and node")
	driverName := flag.String("driver-name", "storage.drycc.cc", "name is the prefix of all objects in data storage")
	healthPort := flag.Int("health-port", 9808, "bbolt db path for savepoint")
	savepointDB := flag.String("savepoint-db", "/csi/savepoint.db", "bbolt db path for savepoint")
	provider.SetFlags()
	flag.Parse()
	if *nodeID == "" {
		log.Fatal("node-id is required")
	}
	driverInfo := &driver.DriverInfo{
		NodeID:      *nodeID,
		DriverName:  *driverName,
		Endpoint:    *endpoint,
		Components:  *components,
		Version:     version,
		SavepointDB: *savepointDB,
		HealthPort:  *healthPort,
	}
	driver, err := driver.New(driverInfo, provider)
	if err != nil {
		log.Fatal(err)
	}
	if strings.Contains(*components, "node") {
		if err := driver.Restore(); err != nil {
			glog.Errorf("node restore error: %v", err)
		}
	}
	driver.Serve()
	os.Exit(0)
}
