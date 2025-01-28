#!/bin/sh
set -e

USER_ID=${USER_ID:-'0000000'}
LICENSE_KEY=${LICENSE_KEY:-'0000000'}
PRODUCT_IDS=${PRODUCT_IDS:-'GeoLite2-City GeoLite2-Country GeoLite2-ASN'}

mkdir -p /usr/local/etc/
cat > /usr/local/etc/GeoIP.conf <<EOL
AccountID $USER_ID
LicenseKey $LICENSE_KEY
EditionIDs $PRODUCT_IDS
EOL

echo "Downloading GeoIP databases..."
/usr/bin/geoipupdate -v

exec "$@"