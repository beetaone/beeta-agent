#!/bin/bash

create_debs() {
    for ARCH in amd64 armhf arm64; do
        mkdir -p deb-${ARCH}
        cp -r DEBIAN deb-${ARCH}

        # populate control file
        echo "Package: beeta-agent
Version: ${VERSION}
License: GPL-3.0
Architecture: ${ARCH}
Maintainer: ${FULL_NAME} <${EMAIL}>
Section: misc
Priority: optional
Homepage: https://beeta.one
Description: A client to manage docker containers on IoT devices." >deb-${ARCH}/DEBIAN/control

        mkdir -p deb-${ARCH}/var/lib/beeta-agent
        cp ca.crt deb-${ARCH}/var/lib/beeta-agent

        mkdir -p deb-${ARCH}/lib/systemd/system/
        cp beeta-agent.service deb-${ARCH}/lib/systemd/system

        mkdir -p deb-${ARCH}/usr/bin/
        if [ "$ARCH" = "armhf" ]; then
            GOARCH="arm"
        else
            GOARCH=$ARCH
        fi
        cp bin/beeta-agent-linux-${GOARCH} deb-${ARCH}/usr/bin/beeta-agent

        # TODO: generate the changelog

        dpkg-deb -Zxz --build deb-${ARCH} "beeta-agent_${VERSION}_${ARCH}.deb"
    done
}

create_sign_release() {
    cp beeta.gpg apt-repo
    cd apt-repo
    mkdir -p pool/main/

    for ARCH in amd64 armhf arm64; do
        # copy the deb
        cp "../beeta-agent_${VERSION}_${ARCH}.deb" pool/main/

        # create Packages* files
        mkdir -p dists/stable/main/binary-${ARCH}
        dpkg-scanpackages --arch ${ARCH} --multiversion pool/ >dists/stable/main/binary-${ARCH}/Packages
        cat dists/stable/main/binary-${ARCH}/Packages | gzip -9 >dists/stable/main/binary-${ARCH}/Packages.gz
    done

    # create *Release* files
    cd dists/stable
    apt-ftparchive release . >Release
    gpg -abs -o - Release >Release.gpg
    gpg --clearsign -o - Release >InRelease
    cd ../../..
}

configure_gpg() {
    echo -n "${GPG_SIGNING_KEY}" | base64 --decode | gpg --import
}

FULL_NAME="Sanyam Arya"
EMAIL=sanyam.arya@beeta.one

VERSION=$(git tag | sort -V | tail -n 1 | cut -c2-)

"$@"
