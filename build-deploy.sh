#!/bin/bash

APP="Reverse Proxy"
OUTPUT_DIR="/opt/mrps"
MAIN_PACKAGE="./cmd/server/main.go"
BINARY_NAME="mrps"
YAML_FILE="./mrps.yaml"
YAML_PATH=$OUTPUT_DIR/mrps.yaml
SERVICE_FILE="mrps.service"
SERVICE_PATH="/etc/systemd/system/$SERVICE_FILE"

copy_file() {
    local source_file=$1
    local dest_file=$2
    echo "$APP: Copying $source_file to $dest_file..."
    sudo cp "$source_file" "$dest_file"
    if [ $? -eq 0 ]; then
        echo "$APP: $source_file successfully copied to $dest_file"
    else
        echo "$APP: Failed to move $source_file. Check permissions or path."
        exit 1
    fi
}

copy_file ./$SERVICE_FILE $SERVICE_PATH
copy_file $YAML_FILE $YAML_PATH

sudo mkdir -p $OUTPUT_DIR

echo "$APP: Building the binary..."
sudo go build -ldflags="-s -w" -o $OUTPUT_DIR/$BINARY_NAME $MAIN_PACKAGE
if [ $? -eq 0 ]; then
    echo "$APP: Build successful. Binary located at $OUTPUT_DIR/$BINARY_NAME"
else
    echo "$APP: Build failed. Check errors above."
    exit 1
fi

echo "$APP: Reloading systemd daemon..."
sudo systemctl daemon-reload
echo "$APP: Daemon reloaded"

if systemctl is-active --quiet $SERVICE_FILE; then
	echo "$APP: Restarting the service..."
    sudo systemctl restart $SERVICE_FILE
	echo "$APP: Service restarted"
else
	echo "$APP: Starting the service..."
    sudo systemctl start $SERVICE_FILE
	echo "$APP: Service started"
fi

sudo systemctl status $SERVICE_FILE
