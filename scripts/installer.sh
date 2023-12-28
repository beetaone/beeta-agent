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

LOG_FILE=installer.log
BEETA_AGENT_DIR="$PWD/beeta-agent"
SERVICE_FILE=/lib/systemd/system/beeta-agent.service
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
empty_line() {
    echo ""
}

cleanup() {
    log INFO Cleaning up ...
    rm -rf "$BEETA_AGENT_DIR"
    log INFO Removed beeta-agent directory
    rm -rf "$SERVICE_FILE"
    log INFO Removed beeta-agent service file
    log INFO Cleanup done
}

validate_config_file_is_present() {
    log INFO Validating if config file is present ...
    if [ -f "$CONFIG_FILE" ]; then
        log INFO Config file is present at "$CONFIG_FILE" ✅
    else
        log_err Config file is not present
        log_err "exiting ..."
        exit 1
    fi
}

validate_docker_installation() {
    log INFO Validating if docker is installed and running ...
    if RESULT=$(docker ps 2>&1); then
        log INFO Docker is installed and running ✅
    else
        log_err Docker is not installed or not running: "$RESULT"
        log_err exiting ...
        exit 1
    fi
}

download_binary() {
    log INFO Detecting the architecture of the machine ...
    ARCH=$(uname -m)
    log INFO Architecture: "$ARCH"

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

    # downloading the respective beeta-agent binary
    BINARY_NAME="beeta-agent-$BINARY_OS-$BINARY_ARCH"
    if RESULT=$(mkdir -p "$BEETA_AGENT_DIR" &&
        cd "$BEETA_AGENT_DIR" &&
        wget $BASE_DOWNLOAD_URL/"$BINARY_NAME" 2>&1); then
        log INFO beeta-agent binary downloaded
        chmod u+x "$BEETA_AGENT_DIR"/"$BINARY_NAME"
        log INFO Changed file permission
    else
        log_err Error while downloading the executable: "$RESULT"
        cleanup
        log_err "exiting ..."
        exit 1
    fi
}

execute() {
    log INFO Executing the beeta-agent binary ...
    if RESULT=$("$BEETA_AGENT_DIR"/"$BINARY_NAME" --config "$CONFIG_FILE" 2>&1); then
        log INFO beeta-agent binary executed
    else
        log_err Error while executing the binary: "$RESULT"
        cleanup
        log_err "exiting ..."
        exit 1
    fi
}

print_logo
cleanup
empty_line
log INFO Detecting the OS of the machine ...
OS=$(uname -s)
log INFO Detected OS: "$OS"
empty_line
validate_config_file_is_present
empty_line
validate_docker_installation
empty_line
download_binary
empty_line
execute
empty_line
