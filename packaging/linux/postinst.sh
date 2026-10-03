#!/bin/bash
# Post-install script for Vajra Debian package
set -e

echo "Installing Vajra..."

# Create system user/group
if ! id "vajra" &>/dev/null; then
    useradd --system --create-home --shell /bin/bash vajra 2>/dev/null || true
fi

# Create directories
mkdir -p /opt/vajra/data /opt/vajra/logs /opt/vajra/config
chown -R vajra:vajra /opt/vajra

# Install systemd service
cp /opt/vajra/vajra.service /etc/systemd/system/vajra.service
systemctl daemon-reload
systemctl enable vajra 2>/dev/null || true

echo "Vajra installed! Run 'systemctl start vajra' to start the service."
