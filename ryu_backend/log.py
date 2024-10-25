# log.py

import logging
import sys

# Create a named logger
LOG = logging.getLogger("RDC_2Servers")
LOG.setLevel(logging.INFO)  # Set the minimum log level for the logger

# Prevent the logger from propagating messages to the root logger
LOG.propagate = False

# Check if handlers are already added to avoid duplicate logs
if not LOG.handlers:
    # Create a console handler
    ch = logging.StreamHandler(sys.stdout)
    ch.setLevel(logging.INFO)  # Set the minimum log level for the handler

    # Create a formatter and set it for the handler
    formatter = logging.Formatter('%(asctime)s - %(message)s')
    ch.setFormatter(formatter)

    # Add the handler to the logger
    LOG.addHandler(ch)
