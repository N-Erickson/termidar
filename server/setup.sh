#!/bin/bash
set -e

echo "===================================="
echo "Termidar SSH Complete Installation"
echo "===================================="
echo ""
echo "This will install Termidar SSH on port 22"
echo "Regular SSH will be moved to port 2222 for management"
echo ""
echo "Starting installation..."
sleep 3

# Variables
TERMIDAR_DIR="/opt/termidar-ssh"
TERMIDAR_USER="termidar"
GO_VERSION="1.21.5"

# Step 1: Disable SELinux
echo "Step 1: Disabling SELinux..."
sudo setenforce 0
sudo sed -i 's/SELINUX=enforcing/SELINUX=disabled/' /etc/selinux/config

# Step 2: Move SSH to port 2222
echo ""
echo "Step 2: Moving SSH to port 2222..."
sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup
sudo sed -i '/^Port/d' /etc/ssh/sshd_config
echo "Port 2222" | sudo tee -a /etc/ssh/sshd_config
sudo systemctl restart sshd

# Open firewall for port 2222
if systemctl is-active --quiet firewalld; then
    sudo firewall-cmd --permanent --add-port=2222/tcp
    sudo firewall-cmd --reload
fi

echo "SSH moved to port 2222"

# Step 3: Update system and install dependencies
echo ""
echo "Step 3: Installing dependencies..."
sudo dnf update -y
sudo dnf install -y git curl tar

# Step 4: Install Go
echo ""
echo "Step 4: Installing Go..."
if ! command -v go &> /dev/null; then
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) GO_ARCH="amd64" ;;
        aarch64) GO_ARCH="arm64" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    curl -LO "https://dl.google.com/go/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    rm "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
    export PATH=$PATH:/usr/local/go/bin
fi

# Step 5: Create termidar user
echo ""
echo "Step 5: Creating termidar user..."
if ! id -u $TERMIDAR_USER &>/dev/null; then
    sudo useradd -r -s /bin/false -m $TERMIDAR_USER
fi

# Step 6: Set up directories
echo ""
echo "Step 6: Setting up directories..."
sudo mkdir -p $TERMIDAR_DIR
sudo chown $TERMIDAR_USER:$TERMIDAR_USER $TERMIDAR_DIR

# Step 7: Build Termidar
echo ""
echo "Step 7: Building Termidar..."
cd $TERMIDAR_DIR

sudo -u $TERMIDAR_USER bash << 'BUILD'
export PATH=$PATH:/usr/local/go/bin
cd /opt/termidar-ssh

# Clone repository
git clone https://github.com/N-Erickson/termidar.git
cd termidar

# Install dependencies
go get github.com/charmbracelet/ssh
go get github.com/charmbracelet/wish
go get github.com/charmbracelet/wish/activeterm
go get github.com/charmbracelet/wish/bubbletea

# Create SSH server
mkdir -p cmd/ssh
cat > cmd/ssh/main.go << 'EOF'
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/ssh"
    "github.com/charmbracelet/wish"
    "github.com/charmbracelet/wish/activeterm"
    "github.com/charmbracelet/wish/bubbletea"
    "github.com/N-Erickson/termidar/internal/ui"
)

func main() {
    port := os.Getenv("TERMIDAR_PORT")
    if port == "" {
        port = "22"
    }
    
    s, err := wish.NewServer(
        wish.WithAddress("0.0.0.0:"+port),
        wish.WithHostKeyPath("/opt/termidar-ssh/.ssh/id_ed25519"),
        wish.WithPasswordAuth(func(ctx ssh.Context, pass string) bool {
            return true
        }),
        wish.WithMiddleware(
            bubbletea.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
                m := ui.InitialModel()
                return m, []tea.ProgramOption{
                    tea.WithAltScreen(),
                    tea.WithInput(s),
                    tea.WithOutput(s),
                }
            }),
            activeterm.Middleware(),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    done := make(chan os.Signal, 1)
    signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
    
    log.Printf("Termidar SSH server started on port %s", port)
    
    go func() {
        if err = s.ListenAndServe(); err != nil {
            log.Fatal(err)
        }
    }()

    <-done
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    s.Shutdown(ctx)
}
EOF

# Build
go build -buildvcs=false -ldflags="-s -w" -o /opt/termidar-ssh/termidar-ssh ./cmd/ssh
chmod +x /opt/termidar-ssh/termidar-ssh
BUILD

# Step 8: Generate SSH host key
echo ""
echo "Step 8: Generating SSH host key..."
sudo -u $TERMIDAR_USER mkdir -p $TERMIDAR_DIR/.ssh
sudo -u $TERMIDAR_USER ssh-keygen -t ed25519 -f $TERMIDAR_DIR/.ssh/id_ed25519 -N ""

# Step 9: Create systemd service
echo ""
echo "Step 9: Creating systemd service..."
sudo tee /etc/systemd/system/termidar-ssh.service > /dev/null << 'EOF'
[Unit]
Description=Termidar SSH Server
After=network.target

[Service]
Type=simple
User=termidar
Group=termidar
WorkingDirectory=/opt/termidar-ssh
Environment="PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
Environment="TERMIDAR_PORT=22"
ExecStart=/opt/termidar-ssh/termidar-ssh
Restart=always
RestartSec=10

# Capabilities
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

# Step 10: Stop SSH and start Termidar
echo ""
echo "Step 10: Switching services..."
sudo systemctl daemon-reload
sudo systemctl enable termidar-ssh
sudo systemctl stop sshd
sudo systemctl stop sshd.socket
sudo systemctl start termidar-ssh

# Step 11: Verify
sleep 3
if systemctl is-active --quiet termidar-ssh; then
    echo ""
    echo "===================================="
    echo "✅ SUCCESS! Termidar is running!"
    echo "===================================="
    echo ""
    echo "Termidar SSH is now on port 22"
    echo "Users can connect: ssh your-domain.com"
    echo ""
    echo "For server management: ssh -p 2222 opc@your-ip"
    echo ""
else
    echo "❌ Failed to start Termidar"
    sudo systemctl start sshd
    echo "Restored regular SSH on port 22"
fi