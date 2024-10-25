
def PrintConnections(LOG, in_ports, out_ports, direction = '|'):
    """
    Logs the connections between input and output ports in a formatted manner.

    Args:
        in_ports (list): List of input port numbers.
        out_ports (list): List of output port numbers.
    """
    # Validate that both lists have the same length
    if len(in_ports) != len(out_ports):
        LOG.error("Mismatch in the number of input and output ports.")
        LOG.error("Total In Ports: %d, Total Out Ports: %d", len(in_ports), len(out_ports))
        return  # Exit the method to prevent incorrect logging

    # Convert port numbers to strings
    in_labels = [str(port) for port in in_ports]
    out_labels = [str(port) for port in out_ports]

    # Define spacing between ports
    spacing = 5  # Number of spaces between ports

    # Initialize variables to build the connection diagram
    line1 = ''
    connector_positions = []
    current_pos = 0

    # Build the first line with input ports and calculate connector positions
    for port in in_labels:
        line1 += port
        port_length = len(port)
        center_pos = current_pos + port_length // 2
        if port_length % 2 == 1:
            center_pos += 1
        connector_positions.append(center_pos)
        current_pos += port_length + spacing
        line1 += ' ' * spacing
    line1 = line1.rstrip()

    # Create a mutable list for line2 to place connectors
    line2_list = list(' ' * len(line1))
    for pos in connector_positions:
        if pos < len(line2_list):
            line2_list[pos] = direction
    line2 = ''.join(line2_list)

    # Build the third line with output ports
    line3 = ''
    current_pos = 0
    for port in out_labels:
        line3 += port
        port_length = len(port)
        current_pos += port_length + spacing
        line3 += ' ' * spacing
    line3 = line3.rstrip()

    # If there's only one port, ensure the vertical line is placed between input and output
    if len(in_ports) == 1:
        center_pos = len(in_labels[0]) // 2
        line2 = ' ' * center_pos + direction + ' ' * (len(line1) - center_pos - 1)
        
    # Log the connection diagram
    LOG.info(line1)
    LOG.info(line2)
    LOG.info(line3)