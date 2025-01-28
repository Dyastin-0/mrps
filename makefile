APP := Reverse Proxy
OUTPUT_DIR := /opt/mrps
MAIN_PACKAGE := ./cmd/server/main.go
BINARY_NAME := mrps
YAML_FILE := ./mrps.yaml
YAML_PATH := $(OUTPUT_DIR)/mrps.yaml
SERVICE_FILE := mrps.service
SERVICE_PATH := /etc/systemd/system/$(SERVICE_FILE)
ENV_FILE := ./.env
ENV_PATH := $(OUTPUT_DIR)/.env

.PHONY: all build install  copy_config  reload restart start status

install:  copy_config build reload restart status

build:
	@echo "$(APP): Building the binary..."
	@sudo mkdir -p $(OUTPUT_DIR)
	@sudo go build -ldflags="-s -w" -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@if [ $$? -eq 0 ]; then \
		echo "$(APP): Build successful. Binary located at $(OUTPUT_DIR)/$(BINARY_NAME)"; \
	else \
		echo "$(APP): Build failed. Check errors above."; \
		exit 1; \
	fi

 copy_config:
	@echo "$(APP): Copying files..."
	@sudo cp $(YAML_FILE) $(YAML_PATH)
	@if [ $$? -eq 0 ]; then \
		echo "$(APP): $(YAML_FILE) successfully copied to $(YAML_PATH)"; \
	else \
		echo "$(APP): Failed to copy $(YAML_FILE). Check permissions or path."; \
		exit 1; \
	fi
	@sudo cp $(SERVICE_FILE) $(SERVICE_PATH)
	@if [ $$? -eq 0 ]; then \
		echo "$(APP): $(SERVICE_FILE) successfully copied to $(SERVICE_PATH)"; \
	else \
		echo "$(APP): Failed to copy $(SERVICE_FILE). Check permissions or path."; \
		exit 1; \
	fi
	@sudo cp $(ENV_FILE) $(ENV_PATH)
	@if [ $$? -eq 0 ]; then \
		echo "$(APP): $(ENV_FILE) successfully copied to $(ENV_PATH)"; \
	else \
		echo "$(APP): Failed to copy $(ENV_FILE). Check permissions or path."; \
		exit 1; \
	fi

 reload:
	@echo "$(APP): Reloading systemd daemon..."
	@sudo systemctl daemon-reload
	@echo "$(APP): Daemon reloaded"

restart:
	@if systemctl is-active --quiet $(SERVICE_FILE); then \
		echo "$(APP): Restarting the service..."; \
		sudo systemctl restart $(SERVICE_FILE); \
		echo "$(APP): Service restarted"; \
	else \
		$(MAKE) start; \
	fi

start:
	@echo "$(APP): Starting the service..."
	@sudo systemctl start $(SERVICE_FILE)
	@echo "$(APP): Service started"

status:
	@sudo systemctl status $(SERVICE_FILE)
