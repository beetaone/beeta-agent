#!/bin/sh

log() {
    echo -e "\033[34m$1\033[0m"
}

SERVER_HOST=the-one-server
SERVER_PATH=/home/ubuntu/workspace/uploads/agent

runOnServer() {
    serverHost=$1
    command=$2
    # if missing server host
    if [ -z "$serverHost" ]; then
        log "Missing server host"
        return
    fi
    # if missing command
    if [ -z "$command" ]; then
        log "Missing command"
        return
    fi
    log "Running command on server"
    ssh -t $serverHost "$command"
}

# Sends data to a server by compressing and transferring it in parts
# Arguments:
#   $1: Source path - the path of the data to be sent
#   $2: Server host - the host address of the server
#   $3: Destination path - the path on the server where the data will be stored
sendDataToServer() {
    sourcePath=$1

    serverHost=$2
    # if missing server host
    if [ -z "$serverHost" ]; then
        log "Missing server host"
        return
    fi

    destinationPath=$3
    # if missing destination path
    if [ -z "$destinationPath" ]; then
        log "Missing destination path"
        return
    fi

    log "Creating temporary directory locally"
    mkdir -p temp

    log "Compressing data"
    zip -r temp/data.zip $sourcePath

    log "Splitting data"
    split -b 4M temp/data.zip temp/data.zip.part

    log "Creating temporary directory on server"
    runOnServer $serverHost "mkdir -p $destinationPath/temp"

    log "Copying data to server"
    scp temp/data.zip.part* $serverHost:$destinationPath/temp

    log "Merging data on server"
    runOnServer $serverHost "cat $destinationPath/temp/data.zip.part* > $destinationPath/temp/data.zip"

    log "Unzipping data on server"
    runOnServer $serverHost "unzip $destinationPath/temp/data.zip -d $destinationPath"

    log "Data sent to server"

    log "Removing temporary directory on server"
    runOnServer $serverHost "rm -rf $destinationPath/temp"

    log "Removing temporary directory locally"
    rm -rf temp

}

log "Build for all platforms"
rm -rf bin
make cross
log "Build for all platforms done"

log "Empty server directory"
runOnServer $SERVER_HOST "rm -rf $SERVER_PATH/*"

log "Copy binaries to server"
sendDataToServer bin $SERVER_HOST $SERVER_PATH

log "Copy scripts to server"
scp -r scripts/installer.sh $SERVER_HOST:$SERVER_PATH

log "Move binaries to server path"
runOnServer $SERVER_HOST "mv $SERVER_PATH/bin/* $SERVER_PATH"

log "Remove temporary binaries directory"
runOnServer $SERVER_HOST "rm -rf $SERVER_PATH/bin"
