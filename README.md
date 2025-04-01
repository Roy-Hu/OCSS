
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
7. **Terminate:**
   It is recommended to terminate the OCSS system first, as it will send delete forwarding table commands to Ryu to remove all rules created by the program.

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
---

## **Main code**
The repository is structured into three main components: State, System, and Ryu.

 - Ryu: Located in the ryu_backend, this component contains the code responsible for managing forwarding rules and handling ocs connection changes.

 - System: This serves as the bridge between Ryu and State. It determines the forwarding rules that need to be added or deleted based on the actions defined in the State and sends the necessary commands to Ryu to apply these changes to the switches. Additionally, it includes a traffic monitor that periodically pulls traffic data from each ToR (Top-of-Rack) switch, providing information for the State to use.

## **Important files**

### internal/context
This folder defines the data structures used in the system and state. It also contains the shared data structure `OCSSContext`, which facilitates communication between the state and the system.

### internal/ocss
- **`process.go`**  
  Represents the system, responsible for creating, updating, and deleting forwarding rules based on configurations and state actions.

- **`forwarder.go`**  
  Implements the REST API to facilitate communication between the system and Ryu.

### ryu_backend
Contains the library for OpenFlow and OCS, used to directly configure switches. It includes:

- **router.py**  
  Handles REST API requests from the system.

- **switch_controller.py**  
  Manages the setup of forwarding rules and OCS connections.

#### How It Works

 ### **System (processor.go)**

The main function of the system is to set up the forwarding rules for the packet switch. It provides two functions:

1. **`CreateForwardingTables`**  
   Creates the forwarding table as provided by the config **before** the state starts.

2. **`UpdateForwardingTables`**  
   Updates the forwarding table based on any changes to the OCS connections.

---

#### Forwarding Table Setup

The forwarding table is set up in three steps:

1. **Server -> Packet Switch -> Edge OCS In-Port**  
   - Performed by **`CreateForwardingTables`** at system initialization.  
   - Establishes a **logical** connection between the server and the Edge OCS.  
     - Physically, the server is connected to a packet switch, which then connects to the Edge OCS.  
     - To simplify this for the user, only the logical server-OCS port mapping is needed.  
   - Creates a forwarding rule in the packet switch to pass all server traffic to the configured Edge OCS in port.

2. **Edge OCS Out Port -> ToR In-Port**  
   - Configures which server is connected to a ToR in-port.  
   - By Step 1, the OCS in-port is logically connected to a server. Given the Edge OCS configuration (from either initial config or user action), the Edge OCS out port and the ToR in-port now know which server they connect to.  
   - Uses `setOcsNxtSwitch(self *ocss_context.OCSSContext, ocs *ocss_context.OCS, ocs_in_port int)` to establish the server ↔ Edge OCS out port ↔ ToR in-port link.  
   - **Note**: This step is needed in both **`CreateForwardingTables`** and **`UpdateForwardingTables`**. (If the Edge OCS connection does not change, it could be skipped theoretically. However, skipping it causes inconsistencies in the connected server info for the ToR port, so it is required.)

3. **ToR-A Out Port <-> OCS Core In-Port <-> OCS Core Out Port <-> ToR-B Out Port**  
   - Sets up the forwarding rules in the ToR.  
   - Implemented in `updateCoreOCSAndTor`.  
   - Iterates over the in/out port pairs in the core OCS and creates forwarding rules for:  
     - **ToR A -> OCS -> ToR B**:  
       1. Find ToR A out port and create a unicast forwarding rule for **all** ToR A in-ports to reach this out port.  
       2. Find the ToR B out port connected to the same OCS out port, then create forwarding rules for that port to all ToR B in-ports.  
     - **Reverse Direction** (ToR B -> OCS -> ToR A):  
       1. Repeat the above steps, but swap ToR A and ToR B.

After the fowarding rule is setup by the system and the OCS connection is setup by the config or action in state, we can call function like `SetOCS`, `CreateForwardingTable` in fowarder.go to let ryu to set up the rules for us.

### **Ryu (ryu_backend)**
`ryu_backend` contains all the parts that directly interact with the switches.

#### **ocs**
A library for interacting with the optical circuit switch.

#### **ofdpa**
A library for interacting with switches via OpenFlow.

#### **router.py** and **switch_controller.py**
- **router.py** acts as the interface (using REST APIs) between the system (`forwarder.go`) and the Ryu backend, calling the corresponding Ryu functions in `switch_controller.py`.
- The major functions include:

  - **`OCS_create_initial_connections(self, ocs_in_port, ocs_out_port)`**:
    Creates OCS connections.

  - **`set_acl_unicast_vlan_inPort_srcIp_dstIp(self, dp, torid, vlan, inPort, srcIp, dstIp, outputPort, cmd, priority=3)`**:
    Manages forwarding rules with commands like `CREATE`, `UPDATE`, or `DELETE`.

  - **`flow_stats_reply_handler(self, ev)`**:
    Update the traffic matrix while `request_flow_stats` is called



