#!/bin/bash
apt-get update && apt-get upgrade
apk add --update ca-certificates
mkdir -p /nats/bin
wget -O - 'https://binaries.nats.dev/nats-io/nats-server/v2@v2.10.4' | PREFIX=/nats/bin/ sh
chmod a+x /nats/bin/nats-server
ln -ns /nats/bin/nats-server /bin/nats-server
ln -ns /nats/bin/nats-server /nats-server
ln -ns /nats/bin/nats-server /usr/local/bin/nats-server
./bin/nats-server
