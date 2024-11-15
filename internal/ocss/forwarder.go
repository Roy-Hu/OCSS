package ocss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/comp590/ocss/internal/logger"
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

// CreateForwardingTable sends forwarding table entries with IP addresses to the RDC
func (f *Forwarder) CreateForwardingTable(dpid int, in_port []int, out_port []int, src_ips []string, dst_ips []string) error {
	if len(in_port) != len(out_port) || len(in_port) != len(src_ips) || len(in_port) != len(dst_ips) {
		return fmt.Errorf("Lengths of in_port, out_port, and ips must be equal")
	}
	entries := make([]map[string]interface{}, len(in_port))
	for i := range in_port {
		entries[i] = map[string]interface{}{
			"in_port":  in_port[i],
			"out_port": out_port[i],
			"src_ip":   src_ips[i],
			"dst_ip":   dst_ips[i],
		}
	}
	data := map[string]interface{}{
		"entries": entries,
	}
	dpidStr := fmt.Sprintf("%016x", dpid)
	endpoint := fmt.Sprintf("/rdc/createforwardingtable/%s", dpidStr)
	return f.postJSON(endpoint, data)
}

// CreateForwardingTable sends forwarding table entries with IP addresses to the RDC
func (f *Forwarder) UpdateForwardingTable(dpid int, in_port []int, out_port []int, src_ips []string, dst_ips []string) error {
	if len(in_port) != len(out_port) || len(in_port) != len(src_ips) || len(in_port) != len(dst_ips) {
		return fmt.Errorf("Lengths of in_port, out_port, and ips must be equal")
	}
	entries := make([]map[string]interface{}, len(in_port))
	for i := range in_port {
		entries[i] = map[string]interface{}{
			"in_port":  in_port[i],
			"out_port": out_port[i],
			"src_ip":   src_ips[i],
			"dst_ip":   dst_ips[i],
		}
	}
	data := map[string]interface{}{
		"entries": entries,
	}
	dpidStr := fmt.Sprintf("%016x", dpid)
	endpoint := fmt.Sprintf("/rdc/updateforwardingtable/%s", dpidStr)
	return f.putJSON(endpoint, data)
}

// postJSON sends a POST request with JSON data
func (f *Forwarder) postJSON(endpoint string, data interface{}) error {
	return f.sendJSONRequest(http.MethodPost, endpoint, data)
}

// putJSON sends a PUT request with JSON data
func (f *Forwarder) putJSON(endpoint string, data interface{}) error {
	return f.sendJSONRequest(http.MethodPut, endpoint, data)
}

// sendJSONRequest is a generalized helper method to send HTTP requests with JSON data
func (f *Forwarder) sendJSONRequest(method, endpoint string, data interface{}) error {
	// Construct the full URL
	url := f.RDCAddress + endpoint

	// Marshal the data into JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON data: %v", err)
	}

	// Create a new HTTP request with the specified method and JSON data
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.ProcessorLog.Errorf("failed to create %s request: %v", method, err)
		return fmt.Errorf("failed to create %s request: %v", method, err)
	}

	// Set the appropriate headers
	req.Header.Set("Content-Type", "application/json")

	// Initialize the HTTP client (you can customize the client if needed)
	client := &http.Client{}

	// Send the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		logger.ProcessorLog.Errorf("failed to create %s request: %v", method, err)
		return fmt.Errorf("failed to send %s request: %v", method, err)
	}
	defer resp.Body.Close()

	// Check for successful status codes (200 OK or 202 Accepted)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		logger.ProcessorLog.Errorf("failed to create %s request: %v", method, err)
		return fmt.Errorf("HTTP %s request failed with status: %s", method, resp.Status)
	}

	return nil
}
