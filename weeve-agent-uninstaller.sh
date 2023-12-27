#!/bin/sh

# logger
log() {
  echo '[' "$(date +"%Y-%m-%d %T")" ']:' INFO "$@"
}

log Detecting the OS of the machine ...
OS=$(uname -s)
log Detected OS: "$OS"

SERVICE_FILE=/lib/systemd/system/beeta-agent.service

# Exctrating the command to run beeta-agent
LINE=$(grep "ExecStart" "$SERVICE_FILE")
COMMAND="sudo ${LINE#ExecStart=} --delete"

BEETA_AGENT_DIR="$PWD/beeta-agent"
if [ ! -d "$BEETA_AGENT_DIR" ]; then
  log beeta-agent directory does not exists in the current path
  log please run the script in the path where beeta-agent directory exists
  exit 1
fi

if [ "$OS" = "Linux" ]; then
  # if in case the user have deleted the beeta-agent.service and did not reload the systemd daemon
  sudo systemctl daemon-reload
fi

if [ "$OS" = "Linux" ]; then
  if RESULT=$(systemctl is-active beeta-agent 2>&1); then
    sudo systemctl stop beeta-agent
  else
    log beeta-agent service not running
  fi

  if [ -f "$SERVICE_FILE" ]; then
    sudo rm "$SERVICE_FILE"
    log "$SERVICE_FILE" removed
  else
    log "$SERVICE_FILE" doesnt exists
  fi
fi

log beeta-agent disconnecting ...
if RESULT=$(cd "$BEETA_AGENT_DIR" && eval "$COMMAND" 2>&1); then
  log beeta-agent disconnected
else
  log Error while restarting beeta-agent for disconnection: "$RESULT"
fi

if [ -d "$BEETA_AGENT_DIR" ]; then
  sudo rm -r "$BEETA_AGENT_DIR"
  log "$BEETA_AGENT_DIR" removed
else
  log "$BEETA_AGENT_DIR" doesnt exists
fi

log Removed beeta-agent contents if any.
