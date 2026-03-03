#!/bin/sh

# Start dbus (required for libvirt)
mkdir -p /var/run/dbus
dbus-daemon --system

# Start libvirtd in the background
echo "Starting libvirtd..."
/usr/sbin/libvirtd -d

# Wait for the socket to be ready
MAX_RETRIES=10
COUNT=0
while [ ! -e /var/run/libvirt/libvirt-sock ] && [ $COUNT -lt $MAX_RETRIES ]; do
    echo "Waiting for libvirt socket..."
    sleep 1
    COUNT=$((COUNT + 1))
done

if [ ! -e /var/run/libvirt/libvirt-sock ]; then
    echo "Error: libvirt socket not found after $MAX_RETRIES seconds"
    exit 1
fi

echo "Libvirt is ready. Starting API..."

# Start the API
exec ./api
