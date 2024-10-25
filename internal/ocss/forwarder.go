package ocss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/comp590/ocss/pkg/app"
)

type FowarderOCSS interface {
	app.App
}

type Forwarder struct {
	RDCAddress string // The base URL of the RDC REST API, e.g., "http://localhost:8080"
	FowarderOCSS
}

func NewForwarder(ocss FowarderOCSS) (*Forwarder, error) {
	f := &Forwarder{
		FowarderOCSS: ocss,
		RDCAddress:   "http://127.0.0.1:8010",
	}

	return f, nil
}

// SetSwitchInfo sends switch IDs, host ports, and switch ports to the RDC
func (f *Forwarder) SetSwitchInfo(dpid int, hostPorts []int, switchPorts []int) error {
	data := map[string]interface{}{
		"hostPorts":   hostPorts,
		"switchPorts": switchPorts,
	}
	// Correctly convert dpid to 16-digit hexadecimal string
	dpidStr := fmt.Sprintf("%016x", dpid)
	endpoint := fmt.Sprintf("/rdc/set_switch_info/%s", dpidStr)
	return f.postJSON(endpoint, data)
}

// SetOCS sends OCS connection data to the RDC
func (f *Forwarder) SetOCS(ip string, ocs_in_port []int, ocs_out_port []int) error {
	data := map[string]interface{}{
		"ocs_in_port":  ocs_in_port,
		"ocs_out_port": ocs_out_port,
	}

	endpoint := fmt.Sprintf("/rdc/set_ocs/%s", ip)
	return f.postJSON(endpoint, data)
}

// SetForwardingTable sends forwarding table entries to the RDC
func (f *Forwarder) SetForwardingTable(dpid int, in_port []int, out_port []int) error {
	entries := make([][]int, len(in_port))
	for i := range in_port {
		entries[i] = []int{in_port[i], out_port[i]}
	}
	dpidStr := fmt.Sprintf("%016x", dpid)
	endpoint := fmt.Sprintf("/rdc/forwardingtable/%s", dpidStr)

	data := map[string]interface{}{
		"entries": entries, // Each entry is [in_port, out_port]
	}
	return f.postJSON(endpoint, data)
}

// SetForwardingTableWithIp sends forwarding table entries with IP addresses to the RDC
func (f *Forwarder) SetForwardingTableWithIp(dpid int, in_port []int, out_port []int, ips []string) error {
	if len(in_port) != len(out_port) || len(in_port) != len(ips) {
		return fmt.Errorf("Lengths of in_port, out_port, and ips must be equal")
	}
	entries := make([]map[string]interface{}, len(in_port))
	for i := range in_port {
		entries[i] = map[string]interface{}{
			"in_port":  in_port[i],
			"out_port": out_port[i],
			"ip":       ips[i],
		}
	}
	data := map[string]interface{}{
		"entries": entries,
	}
	dpidStr := fmt.Sprintf("%016x", dpid)
	endpoint := fmt.Sprintf("/rdc/forwardingtablewithip/%s", dpidStr)
	return f.postJSON(endpoint, data)
}

// Helper method to send POST requests with JSON data
func (f *Forwarder) postJSON(endpoint string, data interface{}) error {
	url := f.RDCAddress + endpoint
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("HTTP request failed with status %s", resp.Status)
	}

	return nil
}
