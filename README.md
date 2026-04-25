# Termidar

Real-time weather radar in your terminal.

https://github.com/user-attachments/assets/86537adf-1c31-4ebc-9bda-59b3e88fb57c

## Install

```bash
# macOS / Linux (one-liner)
curl -sSfL https://raw.githubusercontent.com/N-Erickson/termidar/main/install.sh | sh

# Homebrew
brew install N-Erickson/tap/termidar

# Go
go install github.com/N-Erickson/termidar@latest
```

Pre-built binaries for Linux (.deb, .rpm, .apk), macOS, and Windows are available on the [releases page](https://github.com/N-Erickson/termidar/releases).

## Usage

```bash
termidar
```

Enter a US ZIP code to view live animated radar for that location.

## Controls

| Key | Action |
|-----|--------|
| `Enter` | Submit ZIP code |
| `Space` | Play / Pause |
| `←` `→` | Previous / Next frame |
| `+` `-` | Speed up / Slow down |
| `Z` `X` | Zoom in / Zoom out |
| `R` | Refresh data |
| `?` | Toggle help |
| `ESC` | New location |
| `Q` | Quit |

Mouse hover over the radar to inspect precipitation intensity.

## Features

- Live NEXRAD radar data with animated frame playback
- 160+ radar stations covering the full US
- True color precipitation display with intensity legend
- Geographic overlays — state borders, rivers, mountains, coastlines
- Current temperature, wind speed/direction, and weather conditions
- Rotating 6-period NWS forecast
- Severe weather alerts with visual indicators
- Zoom in/out on the radar view
- Fully responsive — adapts to any terminal size
- Auto-refresh every 5 minutes

## Data Sources

- [RainViewer](https://www.rainviewer.com) — precipitation composites
- [Iowa State Mesonet](https://mesonet.agron.iastate.edu/) — NEXRAD imagery
- [National Weather Service](https://www.weather.gov) — conditions, forecasts, alerts

## Requirements

- Terminal with Unicode and 256-color support (most modern terminals)
- Internet connection

## License

[MIT](LICENSE)
