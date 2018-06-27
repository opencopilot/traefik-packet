#!/bin/sh
set -x

#### Install Docker ###
curl -fsSL get.docker.com -o get-docker.sh
sh get-docker.sh

#### Fetch Metadata ###
META_DATA=$(mktemp /tmp/bootstrap_metadata.json.XXX)
curl -sS metadata.packet.net/metadata > $META_DATA

PRIV_IP=$( cat $META_DATA | jq -r '.network.addresses[] | select(.management == true) | select(.public == false) | select(.address_family == 4) | .address')
PACKET_AUTH=$(cat $META_DATA | jq -r .customdata.PACKET_AUTH)
PACKET_PROJ=$(cat $META_DATA | jq -r .customdata.PACKET_PROJ)
BACKEND_TAG=$(cat $META_DATA | jq -r .customdata.BACKEND_TAG)

mkdir /etc/traefik

cat > /etc/traefik/traefik.toml <<EOF
[api]
  entryPoint = "traefik"
  dashboard = true
[rest]
[metrics]
  # To enable Traefik to export internal metrics to Prometheus
  [metrics.prometheus]
EOF

docker run -d -p $PRIV_IP:8080:8080 -p 80:80 -v /etc/traefik/traefik.toml:/etc/traefik/traefik.toml traefik

docker run -d -e PACKET_AUTH=$PACKET_AUTH -e PACKET_PROJ=$PACKET_PROJ -e BACKEND_TAG=$BACKEND_TAG quay.io/patrickdevivo/traefik-packet