// get-hardware-details is a tool that can be used to convert raw Ironic introspection data into the HardwareDetails
// type used by Metal3.
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/nodes"
	"k8s.io/klog/v2"

	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/clients"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/hardwaredetails"
)

type options struct {
	Endpoint   string
	AuthConfig clients.AuthConfig
	NodeID     string
}

func main() {
	opts := getOptions()
	ironicTrustedCAFile := os.Getenv("IRONIC_CACERT_FILE")
	ironicInsecureStr := os.Getenv("IRONIC_INSECURE")
	ironicInsecure := false
	if strings.ToLower(ironicInsecureStr) == "true" {
		ironicInsecure = true
	}

	tlsConf := clients.TLSConfig{
		TrustedCAFile:      ironicTrustedCAFile,
		InsecureSkipVerify: ironicInsecure,
	}

	endpoint := opts.Endpoint
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		fmt.Printf("invalid ironic endpoint: %s", err)
		os.Exit(1)
	}

	if parsedEndpoint.Port() == "5050" {
		parsedEndpoint.Host = strings.Replace(parsedEndpoint.Host, ":5050", ":6385", 1)
		endpoint = parsedEndpoint.String()
	}

	ironic, err := clients.IronicClient(endpoint, opts.AuthConfig, tlsConf)
	if err != nil {
		fmt.Printf("could not get ironic client: %s", err)
		os.Exit(1)
	}

	introData := nodes.GetInventory(ironic, opts.NodeID)
	data, err := introData.Extract()
	if err != nil {
		fmt.Printf("could not get inspection data: %s", err)
		os.Exit(1)
	}

	json, err := json.MarshalIndent(hardwaredetails.GetHardwareDetails(data, klog.NewKlogr()), "", "\t")
	if err != nil {
		fmt.Printf("could not convert inspection data: %s", err)
		os.Exit(1)
	}

	fmt.Println(string(json))
}

func getOptions() (o options) {
	if len(os.Args) != 3 {
		fmt.Println("Usage: get-hardware-details <ironic URI> <node UUID>")
		os.Exit(1)
	}

	var err error
	o.Endpoint, o.AuthConfig, err = clients.ConfigFromEndpointURL(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	o.NodeID = os.Args[2]
	return
}
