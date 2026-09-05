#!/usr/bin/env bash
# ==============================================================================
# Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
# Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program. If not, see <https://www.gnu.org/licenses/>.
#
# File: build_installer.sh
# Author: Gabriel Moraes
# Date: 2026-09-04
# Packages Noxfort Monitor Server into a standalone Debian package (.deb)
# ==============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_NAME="noxfort-monitor"
VERSION="2.0.1"
ARCH="amd64"
DEB_FILENAME="${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
BUILD_DIR="${SCRIPT_DIR}/build_deb/${PACKAGE_NAME}_${VERSION}_${ARCH}"

echo -e "${CYAN}"
echo "╔══════════════════════════════════════════════════════╗"
echo "║       Noxfort Monitor™ .deb Installer Builder        ║"
echo "╚══════════════════════════════════════════════════════╝"
echo -e "${NC}"

# 1. Compile binary
echo -e "${YELLOW}[1/4] Sincronizando ícones e compilando binário Desktop (Wails v2)...${NC}"
if [ -f "${SCRIPT_DIR}/web/static/img/logo.png" ]; then
    python3 -c '
from PIL import Image
src = Image.open("web/static/img/logo.png")
resized = src.resize((512, 512), Image.Resampling.LANCZOS)
resized.save("internal/tray/icon.png", format="PNG", optimize=True)
' 2>/dev/null || cp "${SCRIPT_DIR}/web/static/img/logo.png" "${SCRIPT_DIR}/internal/tray/icon.png"
fi

mkdir -p "${SCRIPT_DIR}/bin"
GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -tags "production,webkit2_41" -ldflags="-s -w" -o "${SCRIPT_DIR}/bin/noxfort-monitor" "${SCRIPT_DIR}/cmd/server/main.go"
echo -e "${GREEN}✓ Binário compilado com sucesso: bin/noxfort-monitor${NC}"

# 2. Prepare Debian package directory structure
echo -e "${YELLOW}[2/4] Preparando estrutura de arquivos do pacote...${NC}"
rm -rf "${SCRIPT_DIR}/build_deb"
mkdir -p "${BUILD_DIR}/DEBIAN"
mkdir -p "${BUILD_DIR}/opt/noxfort-monitor"
mkdir -p "${BUILD_DIR}/usr/share/applications"
mkdir -p "${BUILD_DIR}/usr/share/pixmaps"
mkdir -p "${BUILD_DIR}/etc/xdg/autostart"

# Copy binary and assets to /opt/noxfort-monitor
cp "${SCRIPT_DIR}/bin/noxfort-monitor" "${BUILD_DIR}/opt/noxfort-monitor/"
cp -r "${SCRIPT_DIR}/web" "${BUILD_DIR}/opt/noxfort-monitor/"

# Icons & Desktop integration
cp "${SCRIPT_DIR}/web/static/img/logo.png" "${BUILD_DIR}/usr/share/pixmaps/noxfort-monitor.png"

# Generate standard hicolor icons for desktop environments (GNOME, KDE, etc.)
python3 -c '
from PIL import Image
import os
src = Image.open("web/static/img/logo.png")
build_dir = "'"${BUILD_DIR}"'"
for s in [16, 24, 32, 48, 64, 128, 256, 512]:
    p = os.path.join(build_dir, f"usr/share/icons/hicolor/{s}x{s}/apps")
    os.makedirs(p, exist_ok=True)
    src.resize((s, s), Image.Resampling.LANCZOS).save(os.path.join(p, "noxfort-monitor.png"), format="PNG", optimize=True)
' 2>/dev/null || true

cat << 'EOF' > "${BUILD_DIR}/usr/share/applications/noxfort-monitor.desktop"
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
StartupWMClass=noxfort-monitor
EOF
chmod 644 "${BUILD_DIR}/usr/share/applications/noxfort-monitor.desktop"

# Autostart desktop file
cp "${BUILD_DIR}/usr/share/applications/noxfort-monitor.desktop" "${BUILD_DIR}/etc/xdg/autostart/noxfort-monitor.desktop"

# 3. Create DEBIAN control & maintainer scripts
echo -e "${YELLOW}[3/4] Criando arquivos de controle e scripts de instalação...${NC}"

cat << EOF > "${BUILD_DIR}/DEBIAN/control"
Package: noxfort-monitor
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${ARCH}
Depends: mosquitto, libayatana-appindicator3-1 | libappindicator3-1, libwebkit2gtk-4.1-0 | libwebkit2gtk-4.0-37, libgtk-3-0
Maintainer: Gabriel Moraes <gabriel@noxfort.com>
Homepage: https://github.com/noxfort/monitor
Description: Noxfort Monitor - Industrial Orchestration Desktop Application
 Noxfort Monitor is an open-source industrial telemetry,
 observability, and incident response orchestration system.
 Monitors IoT/industrial devices via MQTT and sends alerts
 via email and Telegram. Includes a native desktop interface.
EOF

cat << 'EOF' > "${BUILD_DIR}/DEBIAN/postinst"
#!/bin/bash
set -e
# Create launcher symlink
ln -sf /opt/noxfort-monitor/noxfort-monitor /usr/local/bin/noxfort-monitor
chmod +x /opt/noxfort-monitor/noxfort-monitor

# Enable mosquitto autostart
systemctl enable mosquitto 2>/dev/null || true
systemctl start mosquitto 2>/dev/null || true

# Update desktop and icon caches
gtk-update-icon-cache -q /usr/share/icons/hicolor 2>/dev/null || true
update-desktop-database -q /usr/share/applications 2>/dev/null || true

echo ""
echo "✅ Noxfort Monitor instalado com sucesso!"
echo "   Execute: noxfort-monitor"
echo "   Ou procure 'Noxfort Monitor' no menu de aplicativos."
echo ""
exit 0
EOF
chmod 755 "${BUILD_DIR}/DEBIAN/postinst"

cat << 'EOF' > "${BUILD_DIR}/DEBIAN/prerm"
#!/bin/bash
set -e
rm -f /usr/local/bin/noxfort-monitor
exit 0
EOF
chmod 755 "${BUILD_DIR}/DEBIAN/prerm"

cat << 'EOF' > "${BUILD_DIR}/DEBIAN/postrm"
#!/bin/bash
set -e
gtk-update-icon-cache -q /usr/share/icons/hicolor 2>/dev/null || true
update-desktop-database -q /usr/share/applications 2>/dev/null || true
exit 0
EOF
chmod 755 "${BUILD_DIR}/DEBIAN/postrm"

# 4. Build .deb package
echo -e "${YELLOW}[4/4] Gerando pacote .deb...${NC}"
dpkg-deb --build --root-owner-group "${BUILD_DIR}" "${SCRIPT_DIR}/${DEB_FILENAME}"
rm -rf "${SCRIPT_DIR}/build_deb"

DEB_SIZE=$(du -sh "${SCRIPT_DIR}/${DEB_FILENAME}" | cut -f1)

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  INSTALADOR GERADO COM SUCESSO!                      ║${NC}"
echo -e "${GREEN}║                                                      ║${NC}"
echo -e "${GREEN}║  Pacote: ${DEB_FILENAME} (${DEB_SIZE})               ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
