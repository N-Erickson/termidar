#!/bin/bash

echo "==================================="
echo "Termidar Audio Installation"
echo "==================================="
echo ""
echo "This will enable real audio output for the Weather Channel music."
echo ""

# Detect OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Detected macOS"
    echo "Installing PortAudio..."
    if command -v brew &> /dev/null; then
        brew install portaudio
        echo "✅ PortAudio installed"
    else
        echo "❌ Homebrew not found. Please install Homebrew first:"
        echo "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
        exit 1
    fi

elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Detected Linux"
    echo "Installing ALSA development libraries..."

    if command -v apt-get &> /dev/null; then
        sudo apt-get update
        sudo apt-get install -y libasound2-dev
        echo "✅ ALSA libraries installed"
    elif command -v dnf &> /dev/null; then
        sudo dnf install -y alsa-lib-devel
        echo "✅ ALSA libraries installed"
    elif command -v yum &> /dev/null; then
        sudo yum install -y alsa-lib-devel
        echo "✅ ALSA libraries installed"
    else
        echo "❌ Package manager not found. Please install ALSA development libraries:"
        echo "  Ubuntu/Debian: sudo apt-get install libasound2-dev"
        echo "  Fedora: sudo dnf install alsa-lib-devel"
        echo "  CentOS: sudo yum install alsa-lib-devel"
        exit 1
    fi

else
    echo "❌ Unsupported OS: $OSTYPE"
    echo "Audio support is available for macOS and Linux only."
    exit 1
fi

echo ""
echo "✅ Audio dependencies installed!"
echo ""
echo "Now rebuilding Termidar with audio support..."

# Rebuild with audio
go build -tags audio -o termidar

if [ $? -eq 0 ]; then
    echo "✅ Termidar rebuilt with audio support!"
    echo ""
    echo "🎵 You can now enjoy real Weather Channel music:"
    echo "  1. Run: ./termidar"
    echo "  2. Enter a ZIP code"
    echo "  3. Press M to start music"
    echo "  4. Press N to skip tracks"
    echo "  5. Press [ ] for volume control"
    echo ""
    echo "🌦️ Enjoy your enhanced weather radar experience!"
else
    echo "❌ Build failed. The system may not be fully compatible with audio output."
    echo "The simulation mode will continue to work."
fi