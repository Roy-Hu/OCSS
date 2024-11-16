# Makefile

# =============================
# Variables
# =============================

# Binary and directories
BINARY_NAME=ocss
SOURCE_DIR=cmd
BIN_DIR=bin

# tmux session name
SESSION_NAME=ocss_ryu

# Commands to run in tmux panes
OCSS_CMD=./$(BIN_DIR)/$(BINARY_NAME) --config ./config/ocsscfg_2tor.yaml
RYU_CMD=cd ryu_backend && PYTHONPATH=/home/ch155/OCSS ryu-manager switch_controller.py --wsapi-port 8010 --ofp-tcp-listen-port 6633

# =============================
# Default Target
# =============================

.PHONY: all
all: build

# =============================
# Build Target
# =============================

.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@cd $(SOURCE_DIR) && go build -o ../$(BIN_DIR)/$(BINARY_NAME) main.go
	@echo "Build completed. Binary located at $(BIN_DIR)/$(BINARY_NAME)"

# =============================
# Clean Target
# =============================

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BIN_DIR)/$(BINARY_NAME)
	@echo "Clean completed."

# =============================
# tmux Management Targets
# =============================

# Target to start tmux session with two panes
.PHONY: run_tmux
run_tmux: build
	echo "Creating new tmux session '$(SESSION_NAME)'..."; \
	# Start new tmux session detached with the first pane running OCSS_CMD
	tmux new-session -d -s $(SESSION_NAME) -n main '$(RYU_CMD)'; \
	sleep 1; \
	# Split window horizontally and run RYU_CMD in the new pane
	tmux split-window -h -t $(SESSION_NAME):main 'bash -c "$(OCSS_CMD)"'; \
	# Select tiled layout for better visibility
	tmux select-layout -t $(SESSION_NAME) tiled; \
	# Attach to the newly created session
	tmux attach -t $(SESSION_NAME); \
