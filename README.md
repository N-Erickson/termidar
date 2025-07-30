# termidar 🌦️

A beautiful, real-time weather radar in your terminal. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Updated video:


https://github.com/user-attachments/assets/0442c9d7-a3d9-4e64-814f-0a6037d88c2e


https://github.com/user-attachments/assets/0969d801-d653-49ed-8358-672fde4269df


## Features


- 🎯 **Real-time weather radar** - Fetches live NEXRAD data
- 🌍 **ZIP code lookup** - Enter any US ZIP code
- 🎬 **Animated radar loop** - Watch weather patterns move
- 🔄 **Auto-refresh** - Updates every 5 minutes
- ⚡ **Interactive controls** - Play, pause, navigate frames
- 🎨 **Beautiful TUI** - Smooth animations and styled interface
- 📡 **Live radar sweep** - Authentic radar visualization
- 🌈 **Precipitation intensity** - Color-coded from light to severe

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/N-Erickson/termidar.git
cd termiradar

# Build and install
go install

# Or run directly
go run main.go
```

### Requirements

- Go 1.19 or higher
- Terminal with Unicode support
- Internet connection for radar data

## Usage

```bash
termidar
```

### Controls

| Key | Action |
|-----|--------|
| `Enter` | Submit ZIP code |
| `Space` | Play/Pause animation |
| `←` / `→` | Previous/Next frame |
| `+` / `-` | Increase/Decrease speed |
| `R` | Refresh radar data |
| `ESC` | Return to ZIP input |
| `Q` | Quit |

### Supported ZIP Codes

termidar works with any valid US ZIP code. Some examples:

- `10001` - New York, NY
- `60601` - Chicago, IL
- `98101` - Seattle, WA
- `33101` - Miami, FL
- `90210` - Beverly Hills, CA
- `02108` - Boston, MA
- `75201` - Dallas, TX
- `80202` - Denver, CO

## How It Works

termidar fetches real weather radar data from multiple sources:

1. **Iowa State University Mesonet** - NEXRAD radar imagery
2. **RainViewer API** - Global precipitation data
3. **NWS API** - Radar station information

The radar images are processed and converted to ASCII art for terminal display, with color-coded precipitation intensity:

- 🟢 Light precipitation
- 🟡 Moderate precipitation
- 🟠 Heavy precipitation
- 🔴 Severe precipitation

## Development

### Building

```bash
# Get dependencies
go mod download

# Build binary
go build -o termidar

# Run tests
go test ./...
```

### Architecture

- **Bubble Tea** - Terminal UI framework
- **Lipgloss** - Styling and layout
- **Image processing** - Converts radar PNGs to terminal display
- **Concurrent updates** - Separate timers for animation, sweep, and refresh


## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

- [Charm](https://charm.sh) for the amazing Bubble Tea framework
- [Iowa State University](https://mesonet.agron.iastate.edu/) for radar data access
- [RainViewer](https://www.rainviewer.com/api.html) for precipitation API
- [National Weather Service](https://www.weather.gov) for weather data

---

Made with ❤️ and ☕ using [Bubble Tea](https://github.com/charmbracelet/bubbletea)
