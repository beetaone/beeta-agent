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
    #    Kill all pids of beeta
    sudo kill -9 $(ps aux | grep beeta | awk '{print $2}') 2>&1
    log_warn Killed all beeta-agent processes
    rm -rf "$BEETA_AGENT_DIR"
    log_warn Removed beeta-agent directory
    rm -rf "$SERVICE_FILE"
    log_warn Removed beeta-agent service file
    rm -rf "$LOG_FILE"
    log_warn Removed installer log file
    rm -rf beeta_Agent.log

    log_warn Cleanup done
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

# execute() {
#     log Executing the beeta-agent binary ...
#     log ""$BEETA_AGENT_DIR"/"$BINARY_NAME" --config "$CONFIG_FILE" 2>&1"
#     if RESULT=$("$BEETA_AGENT_DIR"/"$BINARY_NAME" --config "$CONFIG_FILE" 2>&1); then
#         log beeta-agent binary executed
#     else
#         log_err Error while executing the binary: "$RESULT"
#         cleanup
#         log_err "exiting ..."
#         exit 1
#     fi
# }

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
print_execute_instruction
empty_line
