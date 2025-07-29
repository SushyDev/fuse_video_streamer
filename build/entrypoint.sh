#!/bin/sh
set -e

# Use PUID/PGID environment variables, with a default of 1000
# This allows the container to run as the host user.
PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Create a group and user
if ! getent group appgroup > /dev/null; then
  addgroup -g "$PGID" appgroup
fi

if ! getent passwd appuser > /dev/null; then
  adduser -D -H -u "$PUID" -G appgroup appuser
fi

# Set ownership of the app's internal working directories at runtime
# This ensures the app can write to its logs, config, etc.
# Note: We do NOT chown the volume mounts themselves (like /mnt/fvs)
# The permissions for those are managed on the host.
chown appuser:appgroup /app

# Drop privileges and execute the main application
echo "Starting application with UID: $PUID, GID: $PGID"
exec su-exec appuser /app/main
