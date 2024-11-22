# -*- coding: utf-8 -*-

import logging
import os
from logging.handlers import RotatingFileHandler
from colorama import Fore, Style, init
import time
from datetime import datetime

# Initialize colorama for color support
init(autoreset=True)

# Constants for field names
FIELD_NF = "NF"
FIELD_CATEGORY = "CAT"

# Custom formatter with color support for console and without for file
class CustomFormatter(logging.Formatter):
    LEVEL_COLORS = {
        "CRITICAL": Fore.RED + Style.BRIGHT,  # Maps to Logrus's Panic/Fatal/Error
        "ERROR": Fore.RED,
        "WARNING": Fore.YELLOW,
        "WARN": Fore.YELLOW,  # To handle both 'WARNING' and 'WARN'
        "INFO": Fore.CYAN,
        "DEBUG": Fore.WHITE,
        "TRACE": Fore.WHITE,
    }

    def __init__(self, use_color=True):
        # Initialize the base Formatter
        logging.Formatter.__init__(self)
        self.use_color = use_color

    def get_timezone_offset(self):
        """
        Returns the timezone offset in the format ±HH:MM
        """
        if time.daylight and time.localtime().tm_isdst:
            offset_sec = -time.altzone
        else:
            offset_sec = -time.timezone
        sign = '+' if offset_sec >= 0 else '-'
        offset_sec = abs(offset_sec)
        hours, remainder = divmod(offset_sec, 3600)
        minutes = remainder / 60.0  # Ensure float division
        return "{0}{1:02d}:{2:02d}".format(sign, hours, int(minutes))

    def formatTime(self, record, datefmt=None):
        """
        Formats the timestamp in RFC3339Nano-like format.
        """
        t = datetime.fromtimestamp(record.created)
        s = t.strftime("%Y-%m-%dT%H:%M:%S.%f")  # microseconds
        s += "000"  # simulate nanoseconds
        s += self.get_timezone_offset()
        return s

    def format(self, record):
        """
        Formats the log record with or without color.
        """
        nf = getattr(record, FIELD_NF, "DefaultNF")
        category = getattr(record, FIELD_CATEGORY, "DefaultCAT")
        levelname = record.levelname

        # Apply color if needed
        if self.use_color and levelname in self.LEVEL_COLORS:
            level_color = self.LEVEL_COLORS[levelname]
            colored_fields = "{0}[{1}][{2}][{3}]{4}".format(
                level_color,
                levelname,
                nf,
                category,
                Style.RESET_ALL
            )
        else:
            colored_fields = "[{0}][{1}][{2}]".format(levelname, nf, category)

        # Format timestamp
        timestamp = self.formatTime(record)

        # Message
        message = record.getMessage()

        # Combine all parts
        log_message = "{0} {1} {2}".format(timestamp, colored_fields, message)

        return log_message

# LoggerManager class for centralized logger management
class LoggerManager(object):
    def __init__(self, nf="DefaultNF"):
        self.nf = nf
        self.loggers = {}

    def setup_logger(self, name, category, log_file=None, level=logging.DEBUG):
        """
        Sets up a logger with both file and console handlers.
        """
        logger = logging.getLogger(name)
        logger.setLevel(level)

        # Prevent adding multiple handlers to logger
        if not logger.handlers:
            # Add file handler if log_file is provided
            if log_file:
                file_handler = RotatingFileHandler(
                    log_file, maxBytes=5 * 1024 * 1024, backupCount=5  # 5 MB file size, 5 backups
                )
                file_handler.setLevel(level)
                file_formatter = CustomFormatter(use_color=False)
                file_handler.setFormatter(file_formatter)
                logger.addHandler(file_handler)

            # Add stream handler for console output
            stream_handler = logging.StreamHandler()
            stream_handler.setLevel(level)
            stream_formatter = CustomFormatter(use_color=True)
            stream_handler.setFormatter(stream_formatter)
            logger.addHandler(stream_handler)

        # Add custom fields using LoggerAdapter
        logger_adapter = logging.LoggerAdapter(logger, {FIELD_NF: self.nf, FIELD_CATEGORY: category})
        self.loggers[name] = logger_adapter
        setattr(self, name, logger_adapter)

    def init_loggers(self):
        """
        Initializes all required loggers.
        """
        log_directory = "./log"

        # Check and create directory if it does not exist
        if not os.path.exists(log_directory):
            os.makedirs(log_directory)

        # Define specific loggers
        # TODO: logging levels should be set by config
        self.setup_logger("SwitchLog", "Switch", log_file="{}/Switch.log".format(log_directory), level=logging.INFO)
        self.setup_logger("OcsLog", "OCS", log_file="{}/OCS.log".format(log_directory), level=logging.INFO)
        self.setup_logger("RouterLog", "Router", log_file="{}/Router.log".format(log_directory), level=logging.INFO)
        self.setup_logger("MainLog", "Main", log_file="{}/Main.log".format(log_directory), level=logging.INFO)

# Singleton instance of LoggerManager
logger = LoggerManager(nf="RYU")
logger.init_loggers()
