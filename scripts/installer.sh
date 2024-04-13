#!/bin/sh

print_logo() {
    echo "  _               _                               "
    echo " | |             | |                              "
    echo " | |__   ___  ___| |_ __ _        ___  _ __   ___ "
    echo " | '_ \ / _ \/ _ \ __/ _\` |      / _ \| '_ \ / _ \\"
    echo " | |_) |  __/  __/ || (_| |  _  | (_) | | | |  __/"
    echo " |_.__/ \___|\___|\__\__,_| (_)  \___/|_| |_|\___|"
    echo "                                                  "
    echo "                                                  "

    echo "\033[0;34m" "beeta one Agent Installer" "\033[0m"
    echo "                                                  "
}

print_execute_instruction() {
    echo "To execute the beeta-agent binary, run the following command:"
    echo ""$BEETA_AGENT_DIR"/"$BINARY_NAME" --config "$CONFIG_FILE" 2>&1 &"
}

create_service_file() {
    log Copying the binary to /opt/beeta-agent
    sudo cp "$BEETA_AGENT_DIR"/"$BINARY_NAME" "$BEETA_AGENT_SERVICE_DIR"

    log Copy the config file to /opt/beeta-agent
    sudo cp "$CONFIG_FILE" "$BEETA_AGENT_SERVICE_DIR"

    log Creating the service file ...
    sudo tee "$SERVICE_FILE" >/dev/null <<EOF
[Unit]
Description=beeta-agent
After=network.target

[Service]
Type=simple
RestartSec=60s
Restart=always
WorkingDirectory=$BEETA_AGENT_SERVICE_DIR
ExecStart=$BEETA_AGENT_SERVICE_DIR/$BINARY_NAME --config $BEETA_AGENT_SERVICE_DIR/$(basename $CONFIG_FILE) 2>&1

[Install]
WantedBy=multi-user.target
EOF

    log Enabling the service ...
    sudo systemctl enable beeta-agent

    log Reloading the systemd manager configuration ...
    sudo systemctl daemon-reload

    log Starting the service ...
    sudo systemctl start beeta-agent

    log beeta-agent service started
}

LOG_FILE=installer.log
BEETA_AGENT_DIR="$PWD/beeta-agent"
SERVICE_FILE=/lib/systemd/system/beeta-agent.service
BEETA_AGENT_SERVICE_DIR=/opt/beeta-agent
BASE_DOWNLOAD_URL=https://file-service.theone.beeta.one/agent
CONFIG_FILE="$1"

log() {
    # logger
    echo "\033[0;34m" '[' "$(date +"%Y-%m-%d %T")" ']:' INFO "$@" "\033[0m" | sudo tee -a "$LOG_FILE"
}

log_err() {
    # logger in red color
    echo "\033[0;31m" '[' "$(date +"%Y-%m-%d %T")" ']:' ERROR "$@" "\033[0m" | sudo tee -a "$LOG_FILE"
}

log_warn() {
    # logger in yellow color
    echo "\033[0;33m" '[' "$(date +"%Y-%m-%d %T")" ']:' WARN "$@" "\033[0m" | sudo tee -a "$LOG_FILE"
}

empty_line() {
    echo ""
}

get_os_arch_info() {
    # log Detecting the OS of the machine ...
    OS=$(uname -s)
    log Detected OS: "$OS"

    log Detecting the architecture of the machine ...
    ARCH=$(uname -m)
    log Architecture: "$ARCH"

    case "$ARCH" in
    "x86_64")
        BINARY_ARCH="amd64"
        ;;
    "arm" | "armv7l")
        BINARY_ARCH="arm"
        ;;
    "arm64" | "aarch64" | "aarch64_be" | "armv8b" | "armv8l")
        BINARY_ARCH="arm64"
        ;;
    *)
        log_err Unsupported architecture: "$ARCH"
        log_err "exiting ..."
        exit 1
        ;;
    esac

    case "$OS" in
    "Linux")
        BINARY_OS="linux"
        ;;
    "Darwin")
        BINARY_OS="macos"
        ;;
    *)
        log_err Unsupported OS: "$OS"
        log_err "exiting ..."
        exit 1
        ;;
    esac

    BINARY_NAME="beeta-agent-$BINARY_OS-$BINARY_ARCH"
    log Binary name: "$BINARY_NAME"
}

cleanup() {
    log_warn Cleaning up ...
    sudo kill -9 $(ps aux | grep beeta-agent | awk '{print $2}') 2>&1
    log_warn Killed all beeta-agent processes

    rm -rf "$BEETA_AGENT_DIR"
    log_warn Removed beeta-agent service file

    rm -rf "$LOG_FILE"
    log_warn Removed installer log file
    rm -rf beeta_Agent.log

    if [ -f "$SERVICE_FILE" ]; then

        log_warn Stopping the service ...
        sudo systemctl stop beeta-agent
        log_warn Stopped the service

        log_warn Disabling the service ...
        sudo systemctl disable beeta-agent
        log_warn Disabled the service

        log_warn Removing the service file ...
        sudo rm -rf "$SERVICE_FILE"
        log_warn Removed the service file

        log_warn Reloading the systemd manager configuration ...
        sudo systemctl daemon-reload
        log_warn Reloaded the systemd manager configuration

        log Backup known manifest and log files
        sudo cp $BEETA_AGENT_SERVICE_DIR/known_manifests.jsonl /tmp/known_manifests.jsonl.bak
        sudo cp $BEETA_AGENT_SERVICE_DIR/beeta_Agent.log /tmp/beeta_Agent.log.bak

        log_warn Removing the service directory ...
        sudo rm -rf "$BEETA_AGENT_SERVICE_DIR"
        log_warn Removed the service directory

        log Restoring known manifest and log files
        sudo mkdir -p $BEETA_AGENT_SERVICE_DIR
        sudo cp /tmp/known_manifests.jsonl.bak $BEETA_AGENT_SERVICE_DIR/known_manifests.jsonl
        sudo cp /tmp/beeta_Agent.log.bak $BEETA_AGENT_SERVICE_DIR/beeta_Agent.log

        # log Cleaning temporary files
        # rm -rf /tmp/known_manifests.jsonl.bak
        # rm -rf /tmp/beeta_Agent.log.bak

    fi

    log_warn Cleanup done
}

post_installation() {
    echo Installation completed successfully
    echo "Please check the instalation logs at $LOG_FILE"
    empty_line
    echo "To see the beeta-agent logs, run the following command:"
    echo "journalctl -u beeta-agent"
    empty_line
    echo "Cleaning installation files ..."
    rm -rf "$BEETA_AGENT_DIR"
    empty_line
    echo "Service status:"
    sudo systemctl status beeta-agent

}

validate_config_file_is_present() {
    log Validating if config file is present ...
    if [ -f "$CONFIG_FILE" ]; then
        log Config file is present at "$CONFIG_FILE" ✅
    else
        log_err Config file is not present
        log_err "exiting ..."
        exit 1
    fi
}

validate_docker_installation() {
    log Validating if docker is installed and running ...
    if RESULT=$(docker ps 2>&1); then
        log Docker is installed and running ✅
    else
        log_err Docker is not installed or not running: "$RESULT"
        log_err exiting ...
        exit 1
    fi
}

download_binary() {

    if RESULT=$(mkdir -p "$BEETA_AGENT_DIR" &&
        cd "$BEETA_AGENT_DIR" &&
        wget $BASE_DOWNLOAD_URL/"$BINARY_NAME" 2>&1); then
        log beeta-agent binary downloaded
        chmod u+x "$BEETA_AGENT_DIR"/"$BINARY_NAME"
        log Changed file permission
    else
        log_err Error while downloading the executable:
        # if not found, then print "the binary is not found for the current architecture and OS" else print the error
        if echo "$RESULT" | grep -q "404 Not Found"; then
            log_err "the binary is not found for the current architecture and OS - $BINARY_NAME"
        else
            log_err "$RESULT"
        fi
        cleanup
        log_err "exiting ..."
        exit 1
    fi
}

print_logo
get_os_arch_info
empty_line
cleanup
empty_line

empty_line
validate_config_file_is_present
empty_line
validate_docker_installation
empty_line
download_binary
empty_line
if [ "$OS" = "Linux" ]; then
    create_service_file
    empty_line
    post_installation
else
    print_execute_instruction
fi
