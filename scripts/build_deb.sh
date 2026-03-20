#!/usr/bin/env bash
# Script: scripts/build_deb.sh
# Builds a self-contained .deb package for Noxfort Monitor
# Usage: bash scripts/build_deb.sh

set -e

PACKAGE="noxfort-monitor"
VERSION="2.0.0"
ARCH="amd64"
INSTALL_DIR="/opt/noxfort-monitor"
DEB_DIR="/tmp/${PACKAGE}_${VERSION}_${ARCH}"
SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> [1/5] Cleaning previous build..."
rm -rf "$DEB_DIR"
mkdir -p "${DEB_DIR}/DEBIAN"
mkdir -p "${DEB_DIR}${INSTALL_DIR}/web/templates"
mkdir -p "${DEB_DIR}${INSTALL_DIR}/web/static"
mkdir -p "${DEB_DIR}/usr/share/applications"
mkdir -p "${DEB_DIR}/usr/share/pixmaps"
mkdir -p "${DEB_DIR}/etc/xdg/autostart"

echo "==> [2/5] Compiling binary..."
cd "$SCRIPT_DIR"
/usr/lib/go-1.22/bin/go build -ldflags="-s -w" -o "${DEB_DIR}${INSTALL_DIR}/noxfort-monitor" ./cmd/server/main.go
echo "    Binary size: $(du -sh ${DEB_DIR}${INSTALL_DIR}/noxfort-monitor | cut -f1)"

echo "==> [3/5] Copying assets..."
cp -r web/templates/. "${DEB_DIR}${INSTALL_DIR}/web/templates/"
cp -r web/static/.   "${DEB_DIR}${INSTALL_DIR}/web/static/"
cp web/static/img/logo.png "${DEB_DIR}/usr/share/pixmaps/noxfort-monitor.png"

echo "==> [4/5] Creating package metadata..."

# DEBIAN/control
cat > "${DEB_DIR}/DEBIAN/control" << EOF
Package: noxfort-monitor
Version: ${VERSION}
Section: net
Priority: optional
Architecture: amd64
Depends: mosquitto, libayatana-appindicator3-1 | libappindicator3-1
Maintainer: Gabriel Moraes <gabriel@noxfort.com>
Homepage: https://github.com/noxfort/monitor
Description: Noxfort Monitor - Industrial Orchestration System
 Noxfort Monitor is an open-source industrial telemetry,
 observability, and incident response orchestration system.
 Monitors IoT/industrial devices via MQTT and sends alerts
 via email and Telegram. Includes a real-time web dashboard.
EOF

# DEBIAN/postinst — runs after installation
cat > "${DEB_DIR}/DEBIAN/postinst" << 'POSTINST'
#!/bin/bash
set -e
# Create launcher symlink
ln -sf /opt/noxfort-monitor/noxfort-monitor /usr/local/bin/noxfort-monitor 2>/dev/null || true
chmod +x /opt/noxfort-monitor/noxfort-monitor

# Enable mosquitto autostart
systemctl enable mosquitto 2>/dev/null || true
systemctl start  mosquitto 2>/dev/null || true

echo ""
echo "✅ Noxfort Monitor instalado com sucesso!"
echo "   Execute: noxfort-monitor"
echo "   Ou procure 'Noxfort Monitor' no menu de aplicativos."
echo ""
POSTINST
chmod 755 "${DEB_DIR}/DEBIAN/postinst"

# DEBIAN/prerm — runs before removal
cat > "${DEB_DIR}/DEBIAN/prerm" << 'PRERM'
#!/bin/bash
set -e
pkill -f noxfort-monitor 2>/dev/null || true
rm -f /usr/local/bin/noxfort-monitor
PRERM
chmod 755 "${DEB_DIR}/DEBIAN/prerm"

# .desktop file — application menu entry
cat > "${DEB_DIR}/usr/share/applications/noxfort-monitor.desktop" << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Noxfort Monitor
GenericName=Industrial Monitoring System
Comment=Noxfort Monitor™ — Industrial Telemetry, Observability & Incident Response
Exec=/opt/noxfort-monitor/noxfort-monitor
Icon=noxfort-monitor
Terminal=false
StartupNotify=true
Categories=Network;Monitor;System;
Keywords=monitor;iot;mqtt;telemetry;industrial;alert;
EOF

# Autostart entry — so it starts with the system session
cp "${DEB_DIR}/usr/share/applications/noxfort-monitor.desktop" \
   "${DEB_DIR}/etc/xdg/autostart/noxfort-monitor.desktop"

echo "==> [5/5] Building .deb package..."
cd /tmp
dpkg-deb --build --root-owner-group "${PACKAGE}_${VERSION}_${ARCH}"

DEB_FILE="${SCRIPT_DIR}/${PACKAGE}_${VERSION}_${ARCH}.deb"
mv "${DEB_DIR}.deb" "$DEB_FILE"

echo ""
echo "============================================================"
echo " ✅ Package built: ${DEB_FILE}"
echo " Size: $(du -sh "$DEB_FILE" | cut -f1)"
echo ""
echo " Install with:  sudo dpkg -i ${PACKAGE}_${VERSION}_${ARCH}.deb"
echo "                sudo apt-get install -f   (fix deps if needed)"
echo "============================================================"
