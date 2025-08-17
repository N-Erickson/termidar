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

# Step 1: Configure firewall FIRST (before changing SSH)
echo "Step 1: Configuring firewall..."
if systemctl is-active --quiet firewalld; then
    echo "Opening port 2222 for management SSH..."
    sudo firewall-cmd --permanent --add-port=2222/tcp
    sudo firewall-cmd --permanent --add-port=22/tcp
    sudo firewall-cmd --reload
    echo "Firewall configured"
else
    echo "Firewalld not active, checking iptables..."
    if command -v iptables &> /dev/null; then
        sudo iptables -A INPUT -p tcp --dport 2222 -j ACCEPT
        sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
        # Try to save iptables rules
        if command -v iptables-save &> /dev/null; then
            sudo iptables-save > /etc/sysconfig/iptables 2>/dev/null || true
        fi
    fi
fi

# Open OCI security list reminder
echo ""
echo "⚠️  IMPORTANT: Make sure your OCI Security List allows:"
echo "   - Port 22 (for Termidar)"
echo "   - Port 2222 (for SSH management)"
echo ""
sleep 5

# Step 2: Disable SELinux
echo "Step 2: Configuring SELinux..."
if [ -f /etc/selinux/config ]; then
    sudo setenforce 0 || true
    sudo sed -i 's/SELINUX=enforcing/SELINUX=disabled/' /etc/selinux/config
    echo "SELinux disabled"
fi

# Step 3: Configure SSH for port 2222 (more carefully)
echo ""
echo "Step 3: Configuring SSH for port 2222..."

# Backup original SSH config
sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup.$(date +%Y%m%d)

# Create a new SSH config for port 2222
sudo tee /etc/ssh/sshd_config.d/99-management.conf > /dev/null << 'EOF'
# Management SSH Configuration
Port 2222
ListenAddress 0.0.0.0
ListenAddress ::

# Security settings
PermitRootLogin prohibit-password
PubkeyAuthentication yes
PasswordAuthentication no
PermitEmptyPasswords no
ChallengeResponseAuthentication no
UsePAM yes

# Keep connection alive
ClientAliveInterval 120
ClientAliveCountMax 3
EOF

# Remove any existing Port directive from main config
sudo sed -i '/^[[:space:]]*Port[[:space:]]/d' /etc/ssh/sshd_config
sudo sed -i '/^[[:space:]]*#Port[[:space:]]/d' /etc/ssh/sshd_config

# Test SSH configuration
echo "Testing SSH configuration..."
sudo sshd -t -f /etc/ssh/sshd_config
if [ $? -ne 0 ]; then
    echo "SSH configuration test failed! Restoring backup..."
    sudo cp /etc/ssh/sshd_config.backup.$(date +%Y%m%d) /etc/ssh/sshd_config
    exit 1
fi

# Restart SSH service
sudo systemctl restart sshd
echo "SSH service restarted on port 2222"

# Verify SSH is listening on 2222
sleep 2
if sudo ss -tlnp | grep -q ":2222"; then
    echo "✓ SSH is now listening on port 2222"
else
    echo "⚠️  Warning: SSH might not be listening on port 2222"
fi

# Step 4: Update system and install dependencies
echo ""
echo "Step 4: Installing dependencies..."
sudo dnf update -y
sudo dnf install -y git curl tar wget

# Step 5: Install Go
echo ""
echo "Step 5: Installing Go..."
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
    echo "Go installed successfully"
fi

# Step 6: Create termidar user
echo ""
echo "Step 6: Creating termidar user..."
if ! id -u $TERMIDAR_USER &>/dev/null; then
    sudo useradd -r -s /bin/false -m $TERMIDAR_USER
    echo "Termidar user created"
fi

# Step 7: Set up directories
echo ""
echo "Step 7: Setting up directories..."
sudo mkdir -p $TERMIDAR_DIR
sudo chown $TERMIDAR_USER:$TERMIDAR_USER $TERMIDAR_DIR

# Step 8: Build Termidar
echo ""
echo "Step 8: Building Termidar..."
cd $TERMIDAR_DIR

# Clone and build as termidar user
sudo -u $TERMIDAR_USER bash << 'BUILD'
export PATH=$PATH:/usr/local/go/bin
cd /opt/termidar-ssh

# Clone repository
if [ -d "termidar" ]; then
    rm -rf termidar
fi
git clone https://github.com/N-Erickson/termidar.git
cd termidar

# Install dependencies
go mod download

# Create SSH server that accepts any connection
mkdir -p cmd/ssh
cat > cmd/ssh/main.go << 'EOSERVER'
package main

import (
    "context"
    "fmt"
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
    "github.com/charmbracelet/lipgloss"
    "github.com/muesli/termenv"
    
    "github.com/N-Erickson/termidar/internal/ui"
)

func main() {
    // Force color output
    lipgloss.SetColorProfile(termenv.ANSI256)
    
    port := os.Getenv("TERMIDAR_PORT")
    if port == "" {
        port = "22"
    }
    
    s, err := wish.NewServer(
        wish.WithAddress("0.0.0.0:"+port),
        wish.WithHostKeyPath("/opt/termidar-ssh/.ssh/id_ed25519"),
        
        // No authentication required - accept everyone
        wish.WithMiddleware(
            func(next ssh.Handler) ssh.Handler {
                return func(sess ssh.Session) {
                    // Log connection
                    log.Printf("New connection from %s (user: %s)", 
                        sess.RemoteAddr(), sess.User())
                    
                    // Clear screen and show welcome
                    fmt.Fprintf(sess, "\033[2J\033[H")
                    fmt.Fprintf(sess, "🌦️  Welcome to Termidar!\n")
                    fmt.Fprintf(sess, "Loading radar interface...\n\n")
                    time.Sleep(500 * time.Millisecond)
                    
                    next(sess)
                }
            },
            bubbletea.Middleware(teaHandler),
            activeterm.Middleware(),
        ),
        
        // Accept any auth method without checking
        wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
            return true  // Accept any password
        }),
        wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
            return true  // Accept any public key
        }),
        wish.WithKeyboardInteractiveAuth(func(ctx ssh.Context, challenge ssh.KeyboardInteractiveChallenge) bool {
            // Just accept without challenge
            return true
        }),
    )
    if err != nil {
        log.Fatalf("Could not create server: %s", err)
    }

    // Handle shutdown gracefully
    done := make(chan os.Signal, 1)
    signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
    
    log.Printf("Termidar SSH server starting on port %s", port)
    log.Printf("Accepting all connections without authentication")
    
    go func() {
        if err = s.ListenAndServe(); err != nil {
            log.Fatalf("Server failed to start: %s", err)
        }
    }()

    <-done
    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := s.Shutdown(ctx); err != nil {
        log.Printf("Could not gracefully shutdown: %s", err)
    }
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    pty, _, active := s.Pty()
    
    if !active {
        log.Printf("Warning: No PTY requested from %s", s.RemoteAddr())
        // Continue anyway
    }
    
    // Force color support
    os.Setenv("COLORTERM", "truecolor")
    if pty.Term != "" {
        os.Setenv("TERM", pty.Term)
    } else {
        os.Setenv("TERM", "xterm-256color")
    }
    
    m := ui.InitialModel()
    
    return m, []tea.ProgramOption{
        tea.WithAltScreen(),
        tea.WithInput(s),
        tea.WithOutput(s),
        tea.WithMouseCellMotion(),
        tea.WithANSICompressor(),
    }
}
EOSERVER

# Build
echo "Building Termidar SSH server..."
go build -buildvcs=false -ldflags="-s -w" -o /opt/termidar-ssh/termidar-ssh ./cmd/ssh
chmod +x /opt/termidar-ssh/termidar-ssh
echo "Build complete"
BUILD

# Step 9: Generate SSH host key
echo ""
echo "Step 9: Generating SSH host key..."
sudo -u $TERMIDAR_USER mkdir -p $TERMIDAR_DIR/.ssh
if [ ! -f $TERMIDAR_DIR/.ssh/id_ed25519 ]; then
    sudo -u $TERMIDAR_USER ssh-keygen -t ed25519 -f $TERMIDAR_DIR/.ssh/id_ed25519 -N ""
    echo "SSH host key generated"
fi

# Step 10: Create systemd service with proper dependencies
echo ""
echo "Step 10: Creating systemd service..."
sudo tee /etc/systemd/system/termidar-ssh.service > /dev/null << 'EOF'
[Unit]
Description=Termidar SSH Server
After=network-online.target sshd.service
Wants=network-online.target
Conflicts=sshd.socket

[Service]
Type=simple
User=termidar
Group=termidar
WorkingDirectory=/opt/termidar-ssh
Environment="PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
Environment="TERMIDAR_PORT=22"

# Wait a bit to ensure network is really up
ExecStartPre=/bin/sleep 5

# Start the service
ExecStart=/opt/termidar-ssh/termidar-ssh

# Restart configuration
Restart=always
RestartSec=10
StartLimitInterval=60
StartLimitBurst=3

# Capabilities for binding to port 22
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

# Logging
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Step 11: Prevent SSH from taking port 22 back
echo ""
echo "Step 11: Configuring service priorities..."

# Disable SSH socket activation which can claim port 22
sudo systemctl stop sshd.socket 2>/dev/null || true
sudo systemctl disable sshd.socket 2>/dev/null || true
sudo systemctl mask sshd.socket 2>/dev/null || true

# Make sure SSH doesn't try to bind to port 22
sudo mkdir -p /etc/systemd/system/sshd.service.d/
sudo tee /etc/systemd/system/sshd.service.d/override.conf > /dev/null << 'EOF'
[Unit]
After=network.target termidar-ssh.service

[Service]
# Ensure we're really on 2222
ExecStartPre=/bin/sleep 2
RestartSec=5
EOF

# Step 12: Start services in correct order
echo ""
echo "Step 12: Starting services..."

# Reload systemd
sudo systemctl daemon-reload

# Stop everything first
sudo systemctl stop termidar-ssh 2>/dev/null || true
sudo systemctl stop sshd

# Start SSH on 2222 first
sudo systemctl start sshd
sleep 2

# Verify SSH is on 2222
if sudo ss -tlnp | grep -q ":2222"; then
    echo "✓ Management SSH is running on port 2222"
else
    echo "⚠️  Warning: Management SSH may not be running on port 2222"
fi

# Now start Termidar on 22
sudo systemctl enable termidar-ssh
sudo systemctl start termidar-ssh
sleep 3

# Step 13: Verify everything
echo ""
echo "===================================="
echo "Verification"
echo "===================================="

# Check ports
echo "Checking ports..."
if sudo ss -tlnp | grep -q ":22 "; then
    echo "✅ Port 22: Termidar is listening"
else
    echo "❌ Port 22: Not listening (Termidar may have failed)"
fi

if sudo ss -tlnp | grep -q ":2222"; then
    echo "✅ Port 2222: SSH management is listening"
else
    echo "❌ Port 2222: Not listening (SSH may have failed)"
fi

# Check services
if systemctl is-active --quiet termidar-ssh; then
    echo "✅ Termidar service is active"
else
    echo "❌ Termidar service is not active"
    echo "   Check logs: sudo journalctl -u termidar-ssh -n 50"
fi

if systemctl is-active --quiet sshd; then
    echo "✅ SSH service is active"
else
    echo "❌ SSH service is not active"
    echo "   Check logs: sudo journalctl -u sshd -n 50"
fi

echo ""
echo "===================================="
echo "Installation Complete!"
echo "===================================="
echo ""
echo "Access methods:"
echo "  🌦️  Termidar (no password): ssh ${HOSTNAME} (or ssh <your-ip>)"
echo "  🔧 Management SSH: ssh -i your-key.pem -p 2222 opc@<your-ip>"
echo ""
echo "⚠️  IMPORTANT REMINDERS:"
echo "  1. Update OCI Security List to allow ports 22 and 2222"
echo "  2. Your SSH key is still required for port 2222 management"
echo "  3. Termidar on port 22 accepts any connection (no auth)"
echo ""
echo "Test connection: ssh localhost (should show Termidar)"
echo ""