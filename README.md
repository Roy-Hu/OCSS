
# OCSS Testbed Setup Instructions

## **Prerequisites**
1. **Set Up the ARP Table:**
   Ensure the ARP table is configured correctly before starting. Run:
   ```bash
   arp -f /home/bold/ethers.arp
   ```

2. **Disable Link Scan on Switch:**
   To maximize throughput while changing the optical circuit:
   - Disable link scanning:
     ```bash
     ./mnt/application/client_drivshell linkscan disable
     ```
   - **Important:** Ensure all ports are active **before disabling link scanning.**
     - For RDC setup, verify active ports using:
       ```bash
       sh ryu_backend/active_testbed_ports.sh
       ```

3. **Run the Ryu Backend:**
   Use the following command to launch the Ryu controller:
   ```bash
   PYTHONPATH=/home/ch155/OCSS ryu-manager switch_controller.py --wsapi-port 8010 --ofp-tcp-listen-port 6633
   ```
   - Wait for the logging message confirming the switch connection.

4. **Run the OCSS System:**
   Once the Ryu controller is running and the switch is connected, start the OCSS system:
   ```bash
   ./bin/ocss --config ./config/ocsscfg_4tor.yaml
   ```

5. **Check the Forwarding Rules:**
   Use the following command to inspect the forwarding table:
   ```bash
   ./mnt/application/client_flowtable_dump 60
   ```

6. **View Logs:**
   OCS logs are maintained by `tl1.py`. Check the logs in the following location:
   ```
   ryu_backend/log
   ```

---

## **Verification and Troubleshooting**

### **a) Verify ARP Table**
Ensure the ARP table is accurate:
```bash
arp -f /home/bold/ethers.arp
```

### **b) Ensure Minimal Network Interfaces**
Check for unnecessary interfaces:
```bash
route -n
```
- Each server should typically have a `10.*` IP address and `192.168.50.*` or `192.168.20.*` addresses.
- If you see unnecessary interfaces like `192.168.10.*`, disable them by running:
  ```bash
  /home/xs6/github/config_netmap.sh
  ```

### **c) Reboot as a Last Resort**
If the above steps don’t resolve the issue, reboot the server:
```bash
reboot
```
- Login details (if prompted):
  - **User:** `root`
  - **Password:** `,l;'

---

This document aims to streamline the setup process and address common issues proactively.
