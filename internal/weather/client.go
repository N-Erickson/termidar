package weather

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Alert represents a weather alert
type Alert struct {
	Event       string
	Severity    string
	Urgency     string
	Headline    string
	Description string
	Expires     time.Time
}

// GetEmoji returns the appropriate emoji for weather conditions
func GetEmoji(conditions string) string {
	if conditions == "" {
		return ""
	}

	cond := strings.ToLower(conditions)

	switch {
	case strings.Contains(cond, "thunder") || strings.Contains(cond, "storm"):
		return "⛈️"
	case strings.Contains(cond, "snow") || strings.Contains(cond, "blizzard"):
		return "🌨️"
	case strings.Contains(cond, "rain") || strings.Contains(cond, "shower"):
		if strings.Contains(cond, "heavy") {
			return "🌧️"
		}
		return "🌦️"
	case strings.Contains(cond, "drizzle") || strings.Contains(cond, "mist"):
		return "🌫️"
	case strings.Contains(cond, "cloud"):
		if strings.Contains(cond, "partly") || strings.Contains(cond, "few") {
			return "⛅"
		}
		return "☁️"
	case strings.Contains(cond, "clear") || strings.Contains(cond, "sunny"):
		hour := time.Now().Hour()
		if hour >= 6 && hour < 18 {
			return "☀️"
		}
		return "🌙"
	case strings.Contains(cond, "fog"):
		return "🌫️"
	case strings.Contains(cond, "wind"):
		return "💨"
	case strings.Contains(cond, "hail"):
		return "🌨️"
	default:
		return "🌤️"
	}
}

// GetAlertDisplay returns emoji, color, and text for weather alerts
func GetAlertDisplay(alerts []Alert) (emoji string, color lipgloss.Color, text string) {
	if len(alerts) == 0 {
		return "", lipgloss.Color(""), ""
	}

	// Find the most severe alert
	var mostSevere Alert
	severityRank := map[string]int{
		"Extreme":  4,
		"Severe":   3,
		"Moderate": 2,
		"Minor":    1,
		"Unknown":  0,
	}

	maxSeverity := -1
	for _, alert := range alerts {
		rank := severityRank[alert.Severity]
		if rank > maxSeverity {
			maxSeverity = rank
			mostSevere = alert
		}
	}

	// Determine emoji and color based on event type and severity
	switch {
	case strings.Contains(strings.ToLower(mostSevere.Event), "tornado"):
		emoji = "🌪️"
		color = lipgloss.Color("196")
		text = "TORNADO " + strings.ToUpper(getAlertType(mostSevere.Event))

	case strings.Contains(strings.ToLower(mostSevere.Event), "severe thunderstorm"):
		emoji = "⛈️"
		color = lipgloss.Color("208")
		text = "SEVERE T-STORM " + strings.ToUpper(getAlertType(mostSevere.Event))

	case strings.Contains(strings.ToLower(mostSevere.Event), "flood"):
		emoji = "🌊"
		color = lipgloss.Color("33")
		text = "FLOOD " + strings.ToUpper(getAlertType(mostSevere.Event))

	case strings.Contains(strings.ToLower(mostSevere.Event), "winter") ||
		strings.Contains(strings.ToLower(mostSevere.Event), "snow") ||
		strings.Contains(strings.ToLower(mostSevere.Event), "blizzard"):
		emoji = "❄️"
		color = lipgloss.Color("51")
		text = strings.ToUpper(getAlertType(mostSevere.Event))

	case strings.Contains(strings.ToLower(mostSevere.Event), "heat"):
		emoji = "🔥"
		color = lipgloss.Color("202")
		text = "HEAT " + strings.ToUpper(getAlertType(mostSevere.Event))

	case strings.Contains(strings.ToLower(mostSevere.Event), "wind"):
		emoji = "💨"
		color = lipgloss.Color("226")
		text = "WIND " + strings.ToUpper(getAlertType(mostSevere.Event))

	default:
		emoji = "⚠️"
		if mostSevere.Severity == "Extreme" {
			color = lipgloss.Color("196")
		} else if mostSevere.Severity == "Severe" {
			color = lipgloss.Color("208")
		} else {
			color = lipgloss.Color("226")
		}
		text = strings.ToUpper(getAlertType(mostSevere.Event))
	}

	return emoji, color, text
}

// getAlertType returns the alert type string (private helper)
func getAlertType(event string) string {
	switch {
	case strings.Contains(event, "Warning"):
		return "WARNING"
	case strings.Contains(event, "Watch"):
		return "WATCH"
	case strings.Contains(event, "Advisory"):
		return "ADVISORY"
	default:
		return "ALERT"
	}
}

// FetchAlerts fetches weather alerts for the given coordinates
func FetchAlerts(lat, lon float64) []Alert {
	client := &http.Client{Timeout: 5 * time.Second}

	alertsURL := fmt.Sprintf("https://api.weather.gov/alerts/active?point=%.4f,%.4f", lat, lon)

	resp, err := client.Get(alertsURL)
	if err != nil {
		log.Printf("Failed to fetch weather alerts: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var alertsData struct {
		Features []struct {
			Properties struct {
				Event       string    `json:"event"`
				Severity    string    `json:"severity"`
				Urgency     string    `json:"urgency"`
				Headline    string    `json:"headline"`
				Description string    `json:"description"`
				Expires     time.Time `json:"expires"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&alertsData); err != nil {
		log.Printf("Failed to decode alerts: %v", err)
		return nil
	}

	var alerts []Alert
	for _, feature := range alertsData.Features {
		alert := Alert{
			Event:       feature.Properties.Event,
			Severity:    feature.Properties.Severity,
			Urgency:     feature.Properties.Urgency,
			Headline:    feature.Properties.Headline,
			Description: feature.Properties.Description,
			Expires:     feature.Properties.Expires,
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// FetchCurrentConditions fetches current weather conditions for the given coordinates
func FetchCurrentConditions(lat, lon float64) (int, string) {
	client := &http.Client{Timeout: 5 * time.Second}

	pointURL := fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", lat, lon)

	resp, err := client.Get(pointURL)
	if err != nil {
		log.Printf("Failed to get NWS point data: %v", err)
		return 0, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("NWS point API returned status: %d", resp.StatusCode)
		return 0, ""
	}

	var pointData struct {
		Properties struct {
			ForecastURL    string `json:"forecast"`
			ObservationURL string `json:"observationStations"`
		} `json:"properties"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pointData); err != nil {
		log.Printf("Failed to decode NWS point data: %v", err)
		return 0, ""
	}

	stationsResp, err := client.Get(pointData.Properties.ObservationURL)
	if err != nil {
		log.Printf("Failed to get observation stations: %v", err)
		return 0, ""
	}
	defer stationsResp.Body.Close()

	var stationsData struct {
		Features []struct {
			Properties struct {
				StationIdentifier string `json:"stationIdentifier"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.NewDecoder(stationsResp.Body).Decode(&stationsData); err != nil {
		log.Printf("Failed to decode stations data: %v", err)
		return 0, ""
	}

	if len(stationsData.Features) == 0 {
		log.Printf("No observation stations found")
		return 0, ""
	}

	stationID := stationsData.Features[0].Properties.StationIdentifier
	obsURL := fmt.Sprintf("https://api.weather.gov/stations/%s/observations/latest", stationID)

	obsResp, err := client.Get(obsURL)
	if err != nil {
		log.Printf("Failed to get observations: %v", err)
		return 0, ""
	}
	defer obsResp.Body.Close()

	var obsData struct {
		Properties struct {
			Temperature struct {
				Value    float64 `json:"value"`
				UnitCode string  `json:"unitCode"`
			} `json:"temperature"`
			TextDescription string `json:"textDescription"`
		} `json:"properties"`
	}

	if err := json.NewDecoder(obsResp.Body).Decode(&obsData); err != nil {
		log.Printf("Failed to decode observation data: %v", err)
		return 0, ""
	}

	temp := obsData.Properties.Temperature.Value
	unitCode := obsData.Properties.Temperature.UnitCode
	
	// Log for debugging
	log.Printf("Temperature value: %f, unit: %s", temp, unitCode)
	
	// Check for Celsius in various formats the API might return
	if strings.Contains(strings.ToLower(unitCode), "degc") || 
	   strings.Contains(strings.ToLower(unitCode), "celsius") ||
	   unitCode == "wmoUnit:degC" ||
	   unitCode == "unit:degC" {
		temp = temp*9/5 + 32
		log.Printf("Converted from Celsius to Fahrenheit: %f", temp)
	}

	conditions := obsData.Properties.TextDescription
	if conditions == "" {
		conditions = "Clear"
	}

	return int(temp), conditions
}

// GeocodeZip converts a ZIP code to coordinates and location information
func GeocodeZip(zipCode string) (float64, float64, string, string, error) {
	url := fmt.Sprintf("https://api.zippopotam.us/us/%s", zipCode)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return geocodeZipAlternative(zipCode)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return geocodeZipAlternative(zipCode)
	}

	var result struct {
		PostCode    string `json:"post code"`
		Country     string `json:"country"`
		CountryCode string `json:"country abbreviation"`
		Places      []struct {
			PlaceName  string  `json:"place name"`
			State      string  `json:"state"`
			StateCode  string  `json:"state abbreviation"`
			Latitude   string  `json:"latitude"`
			Longitude  string  `json:"longitude"`
		} `json:"places"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return geocodeZipAlternative(zipCode)
	}

	if len(result.Places) == 0 {
		return geocodeZipAlternative(zipCode)
	}

	place := result.Places[0]

	lat, err := strconv.ParseFloat(place.Latitude, 64)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("invalid latitude for ZIP %s", zipCode)
	}

	lon, err := strconv.ParseFloat(place.Longitude, 64)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("invalid longitude for ZIP %s", zipCode)
	}

	return lat, lon, place.PlaceName, place.StateCode, nil
}

// geocodeZipAlternative provides a fallback geocoding service (private helper)
func geocodeZipAlternative(zipCode string) (float64, float64, string, string, error) {
	url := fmt.Sprintf("https://api.geocod.io/v1.7/geocode?q=%s&api_key=demo", zipCode)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("failed to geocode ZIP %s: %w", zipCode, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, "", "", fmt.Errorf("unable to find location for ZIP %s", zipCode)
	}

	var result struct {
		Results []struct {
			AddressComponents struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"address_components"`
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, "", "", fmt.Errorf("failed to decode geocoding response: %w", err)
	}

	if len(result.Results) == 0 {
		return 0, 0, "", "", fmt.Errorf("no results found for ZIP %s", zipCode)
	}

	r := result.Results[0]
	return r.Location.Lat, r.Location.Lng, r.AddressComponents.City, r.AddressComponents.State, nil
}

// GetNearestRadarStation returns the nearest NWS radar station for given coordinates
func GetNearestRadarStation(lat, lon float64) (string, error) {
	stations := []struct {
		id   string
		lat  float64
		lon  float64
	}{
		{"KOKX", 40.8653, -72.8639},  // New York
		{"KLOT", 41.6045, -88.0847},  // Chicago
		{"KAMX", 25.6111, -80.4128},  // Miami
		{"KATX", 48.1945, -122.4958}, // Seattle
		{"KFWS", 32.5731, -97.3031},  // Dallas
		{"KLVX", 37.9753, -85.9439},  // Louisville
		{"KTFX", 47.4595, -111.3855}, // Great Falls
		{"KSGF", 37.2355, -93.4003},  // Springfield
		{"KLAS", 36.0558, -115.1622}, // Las Vegas
		{"KPHX", 33.4301, -112.0128}, // Phoenix
	}

	minDist := 999999.0
	nearest := "KOKX"

	for _, s := range stations {
		dist := math.Sqrt(math.Pow(lat-s.lat, 2) + math.Pow(lon-s.lon, 2))
		if dist < minDist {
			minDist = dist
			nearest = s.id
		}
	}

	return nearest, nil
}