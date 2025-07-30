package radar

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/N-Erickson/termidar/internal/config"
	"github.com/N-Erickson/termidar/internal/weather"
)

// Data represents radar data with frames and metadata
type Data struct {
	Frames      []Frame
	Location    string
	Station     string
	LastUpdated time.Time
	IsRealData  bool
	Temperature int
	Conditions  string
	Alerts      []weather.Alert
}

// Frame represents a single radar frame
type Frame struct {
	Data      [][]int
	Timestamp time.Time
	Product   string
}

// Messages for tea.Cmd communication
type LoadedMsg struct {
	Radar Data
}

type ErrorMsg struct {
	Err error
}

// LoadData loads radar data for a given ZIP code
func LoadData(zipCode string) tea.Cmd {
	return func() tea.Msg {
		// Create a custom logger that discards output during loading
		// This prevents console spam from interfering with the display
		oldOutput := log.Writer()
		log.SetOutput(io.Discard)
		defer log.SetOutput(oldOutput)

		lat, lon, city, state, err := weather.GeocodeZip(zipCode)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to geocode ZIP: %w", err)}
		}

		station, err := weather.GetNearestRadarStation(lat, lon)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to get radar station: %w", err)}
		}

		temperature, conditions := weather.FetchCurrentConditions(lat, lon)
		alerts := weather.FetchAlerts(lat, lon)

		frames, isRealData, err := fetchRealRadarData(station, lat, lon)
		if err != nil {
			frames = generateRadarFrames(station, config.MaxFrames)
			isRealData = false
		}

		location := fmt.Sprintf("%s, %s", city, state)

		return LoadedMsg{
			Radar: Data{
				Frames:      frames,
				Location:    location,
				Station:     station,
				LastUpdated: time.Now(),
				IsRealData:  isRealData,
				Temperature: temperature,
				Conditions:  conditions,
				Alerts:      alerts,
			},
		}
	}
}

func fetchRealRadarData(station string, lat, lon float64) ([]Frame, bool, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	frames := []Frame{}

	// First try RainViewer
	frames, err := fetchFromRainViewer(lat, lon)
	if err == nil && len(frames) > 0 {
		log.Printf("Successfully fetched %d frames from RainViewer", len(frames))
		return frames, true, nil
	}

	// Fallback to Iowa State University
	baseTime := time.Now().UTC()

	for i := 0; i < 24; i++ {
		frameTime := baseTime.Add(time.Duration(-i*5) * time.Minute)

		minutes := frameTime.Minute()
		minutes = (minutes / 5) * 5
		frameTime = time.Date(frameTime.Year(), frameTime.Month(), frameTime.Day(),
			frameTime.Hour(), minutes, 0, 0, time.UTC)

		timeStr := frameTime.Format("200601021504")
		radarURL := fmt.Sprintf("https://mesonet.agron.iastate.edu/cgi-bin/wms/nexrad/n0r.cgi?SERVICE=WMS&VERSION=1.1.1&REQUEST=GetMap&FORMAT=image/png&TRANSPARENT=true&LAYERS=nexrad-n0r&WIDTH=%d&HEIGHT=%d&SRS=EPSG:4326&BBOX=%f,%f,%f,%f&TIME=%s",
			config.RadarWidth*4, config.RadarHeight*4,
			lon-2.5, lat-2.0, lon+2.5, lat+2.0,
			timeStr,
		)

		resp, err := client.Get(radarURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		img, err := png.Decode(resp.Body)
		if err != nil {
			continue
		}

		data := imageToRadarData(img)
		if data != nil {
			frame := Frame{
				Data:      data,
				Timestamp: frameTime,
				Product:   "N0R",
			}
			frames = append(frames, frame)
		}

		if len(frames) >= config.MaxFrames {
			break
		}
	}

	if len(frames) == 0 {
		return nil, false, fmt.Errorf("no radar data available")
	}

	// Reverse frames so oldest is first
	for i := len(frames)/2 - 1; i >= 0; i-- {
		opp := len(frames) - 1 - i
		frames[i], frames[opp] = frames[opp], frames[i]
	}

	log.Printf("Successfully fetched %d frames from Iowa State", len(frames))
	return frames, true, nil
}

func fetchFromRainViewer(lat, lon float64) ([]Frame, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://api.rainviewer.com/public/weather-maps.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiData struct {
		Radar struct {
			Past []struct {
				Time int64  `json:"time"`
				Path string `json:"path"`
			} `json:"past"`
			Nowcast []struct {
				Time int64  `json:"time"`
				Path string `json:"path"`
			} `json:"nowcast"`
		} `json:"radar"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		return nil, err
	}

	frames := []Frame{}

	for _, past := range apiData.Radar.Past {
		zoom := 7
		tileX, tileY := latLonToTile(lat, lon, zoom)

		tileURL := fmt.Sprintf("https://tilecache.rainviewer.com%s/512/%d/%d/%d/6/1_1.png",
			past.Path, zoom, tileX, tileY)

		resp, err := client.Get(tileURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		img, err := png.Decode(resp.Body)
		if err != nil {
			continue
		}

		data := imageToRadarData(img)
		if data != nil {
			frame := Frame{
				Data:      data,
				Timestamp: time.Unix(past.Time, 0),
				Product:   "Composite",
			}
			frames = append(frames, frame)
		}

		if len(frames) >= config.MaxFrames {
			break
		}
	}

	return frames, nil
}

func latLonToTile(lat, lon float64, zoom int) (int, int) {
	n := math.Pow(2, float64(zoom))
	x := int((lon + 180.0) / 360.0 * n)
	latRad := lat * math.Pi / 180.0
	y := int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n)
	return x, y
}

func imageToRadarData(img image.Image) [][]int {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	data := make([][]int, config.RadarHeight)
	for i := range data {
		data[i] = make([]int, config.RadarWidth)
	}

	foundPrecipitation := false

	for y := 0; y < config.RadarHeight; y++ {
		for x := 0; x < config.RadarWidth; x++ {
			imgX := x * width / config.RadarWidth
			imgY := y * height / config.RadarHeight

			c := img.At(imgX, imgY)
			r, g, b, a := c.RGBA()

			if a < 128 {
				continue
			}

			intensity := 0

			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if r8 > 200 && g8 < 100 && b8 < 100 {
				intensity = 8 + int((r8-200)/28)
				foundPrecipitation = true
			} else if r8 > 200 && g8 > 150 && b8 < 100 {
				intensity = 5 + int((r8-200)/50)
				foundPrecipitation = true
			} else if g8 > 150 && r8 < 150 && b8 < 100 {
				intensity = 1 + int((g8-150)/50)
				foundPrecipitation = true
			} else if b8 > 150 && r8 < 100 && g8 < 150 {
				intensity = 1
				foundPrecipitation = true
			} else if r8 > 100 && g8 > 100 && b8 < 50 {
				intensity = 4
				foundPrecipitation = true
			}

			if intensity > 10 {
				intensity = 10
			}
			if intensity < 0 {
				intensity = 0
			}

			data[y][x] = intensity
		}
	}

	if foundPrecipitation {
		log.Printf("Found precipitation in radar image")
	} else {
		log.Printf("No precipitation detected in radar image")
	}

	return data
}

func generateRadarFrames(station string, count int) []Frame {
	frames := make([]Frame, count)

	for i := 0; i < count; i++ {
		data := make([][]int, config.RadarHeight)
		for y := range data {
			data[y] = make([]int, config.RadarWidth)
		}

		numCells := 2 + i%3
		for c := 0; c < numCells; c++ {
			centerX := 10 + (i*3+c*15)%config.RadarWidth
			centerY := 5 + (i*2+c*10)%config.RadarHeight
			intensity := 5 + c*2

			for dy := -5; dy <= 5; dy++ {
				for dx := -5; dx <= 5; dx++ {
					x, y := centerX+dx, centerY+dy
					if x >= 0 && x < config.RadarWidth && y >= 0 && y < config.RadarHeight {
						dist := math.Sqrt(float64(dx*dx + dy*dy))
						if dist < 5 {
							data[y][x] = intensity - int(dist)
							if data[y][x] < 0 {
								data[y][x] = 0
							}
							if data[y][x] > 10 {
								data[y][x] = 10
							}
						}
					}
				}
			}
		}

		frames[i] = Frame{
			Data:      data,
			Timestamp: time.Now().Add(time.Duration(i*10) * time.Minute),
		}
	}

	return frames
}