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

// WindData holds wind speed and direction
type WindData struct {
	Speed     float64 // in mph
	Direction int     // in degrees (0=N, 90=E, 180=S, 270=W)
	Arrow     string  // unicode arrow
}

// GetWindArrow returns the unicode arrow for a compass direction in degrees
func GetWindArrow(degrees int) string {
	arrows := []string{"↓", "↙", "←", "↖", "↑", "↗", "→", "↘"}
	// Wind direction is where wind comes FROM, arrow shows where it blows TO
	idx := ((degrees + 22) / 45) % 8
	return arrows[idx]
}

// FetchCurrentConditions fetches current weather conditions for the given coordinates
func FetchCurrentConditions(lat, lon float64) (int, string, WindData) {
	client := &http.Client{Timeout: 5 * time.Second}

	pointURL := fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", lat, lon)

	noWind := WindData{}

	resp, err := client.Get(pointURL)
	if err != nil {
		log.Printf("Failed to get NWS point data: %v", err)
		return 0, "", noWind
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("NWS point API returned status: %d", resp.StatusCode)
		return 0, "", noWind
	}

	var pointData struct {
		Properties struct {
			ForecastURL    string `json:"forecast"`
			ObservationURL string `json:"observationStations"`
		} `json:"properties"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pointData); err != nil {
		log.Printf("Failed to decode NWS point data: %v", err)
		return 0, "", noWind
	}

	stationsResp, err := client.Get(pointData.Properties.ObservationURL)
	if err != nil {
		log.Printf("Failed to get observation stations: %v", err)
		return 0, "", noWind
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
		return 0, "", noWind
	}

	if len(stationsData.Features) == 0 {
		log.Printf("No observation stations found")
		return 0, "", noWind
	}

	stationID := stationsData.Features[0].Properties.StationIdentifier
	obsURL := fmt.Sprintf("https://api.weather.gov/stations/%s/observations/latest", stationID)

	obsResp, err := client.Get(obsURL)
	if err != nil {
		log.Printf("Failed to get observations: %v", err)
		return 0, "", noWind
	}
	defer obsResp.Body.Close()

	var obsData struct {
		Properties struct {
			Temperature struct {
				Value    float64 `json:"value"`
				UnitCode string  `json:"unitCode"`
			} `json:"temperature"`
			WindSpeed struct {
				Value    float64 `json:"value"`
				UnitCode string  `json:"unitCode"`
			} `json:"windSpeed"`
			WindDirection struct {
				Value float64 `json:"value"`
			} `json:"windDirection"`
			TextDescription string `json:"textDescription"`
		} `json:"properties"`
	}

	if err := json.NewDecoder(obsResp.Body).Decode(&obsData); err != nil {
		log.Printf("Failed to decode observation data: %v", err)
		return 0, "", noWind
	}

	temp := obsData.Properties.Temperature.Value
	unitCode := obsData.Properties.Temperature.UnitCode

	// Check for Celsius in various formats the API might return
	if strings.Contains(strings.ToLower(unitCode), "degc") ||
		strings.Contains(strings.ToLower(unitCode), "celsius") ||
		unitCode == "wmoUnit:degC" ||
		unitCode == "unit:degC" {
		temp = temp*9/5 + 32
	}

	conditions := obsData.Properties.TextDescription
	if conditions == "" {
		conditions = "Clear"
	}

	// Parse wind data
	wind := WindData{}
	windSpeedKmh := obsData.Properties.WindSpeed.Value
	if windSpeedKmh > 0 {
		wind.Speed = windSpeedKmh * 0.621371 // km/h to mph
		wind.Direction = int(obsData.Properties.WindDirection.Value)
		wind.Arrow = GetWindArrow(wind.Direction)
	}

	return int(temp), conditions, wind
}

// ForecastPeriod represents a single forecast period
type ForecastPeriod struct {
	Name         string
	Temperature  int
	TempUnit     string
	WindSpeed    string
	WindDir      string
	ShortFcast   string
	IsDaytime    bool
}

// FetchForecast fetches a compact forecast for the given coordinates
func FetchForecast(lat, lon float64) []ForecastPeriod {
	client := &http.Client{Timeout: 5 * time.Second}

	pointURL := fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", lat, lon)
	resp, err := client.Get(pointURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var pointData struct {
		Properties struct {
			Forecast string `json:"forecast"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pointData); err != nil || pointData.Properties.Forecast == "" {
		return nil
	}

	fResp, err := client.Get(pointData.Properties.Forecast)
	if err != nil {
		return nil
	}
	defer fResp.Body.Close()

	if fResp.StatusCode != http.StatusOK {
		return nil
	}

	var forecastData struct {
		Properties struct {
			Periods []struct {
				Name            string `json:"name"`
				Temperature     int    `json:"temperature"`
				TemperatureUnit string `json:"temperatureUnit"`
				WindSpeed       string `json:"windSpeed"`
				WindDirection    string `json:"windDirection"`
				ShortForecast   string `json:"shortForecast"`
				IsDaytime       bool   `json:"isDaytime"`
			} `json:"periods"`
		} `json:"properties"`
	}

	if err := json.NewDecoder(fResp.Body).Decode(&forecastData); err != nil {
		return nil
	}

	var periods []ForecastPeriod
	limit := 6
	if len(forecastData.Properties.Periods) < limit {
		limit = len(forecastData.Properties.Periods)
	}
	for _, p := range forecastData.Properties.Periods[:limit] {
		periods = append(periods, ForecastPeriod{
			Name:        p.Name,
			Temperature: p.Temperature,
			TempUnit:    p.TemperatureUnit,
			WindSpeed:   p.WindSpeed,
			WindDir:     p.WindDirection,
			ShortFcast:  p.ShortForecast,
			IsDaytime:   p.IsDaytime,
		})
	}
	return periods
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
		// Northeast
		{"KOKX", 40.8653, -72.8639},  // New York City
		{"KBOX", 41.9558, -71.1369},  // Boston
		{"KENX", 42.5864, -74.0639},  // Albany
		{"KBUF", 42.9489, -78.7369},  // Buffalo
		{"KBGM", 42.1997, -75.9847},  // Binghamton
		{"KCBW", 46.0392, -67.8067},  // Caribou (ME)
		{"KGYX", 43.8914, -70.2564},  // Portland (ME)
		{"KDIX", 39.9472, -74.4108},  // Philadelphia
		{"KCCX", 40.9228, -78.0039},  // State College (PA)
		{"KPBZ", 40.5317, -80.2181},  // Pittsburgh
		{"KDOX", 38.8256, -75.4400},  // Dover (DE)
		{"KLWX", 38.9753, -77.4778},  // Washington DC
		{"KAKQ", 36.9839, -77.0078},  // Wakefield (VA)
		{"KFCX", 37.0242, -80.2739},  // Roanoke (VA)
		{"KRLX", 38.3111, -81.7228},  // Charleston (WV)
		// Southeast
		{"KAMX", 25.6111, -80.4128},  // Miami
		{"KMLB", 28.1133, -80.6542},  // Melbourne (FL)
		{"KJAX", 30.4847, -81.7019},  // Jacksonville
		{"KTLH", 30.3975, -84.3289},  // Tallahassee
		{"KTBW", 27.7056, -82.4017},  // Tampa Bay
		{"KEVX", 30.5644, -85.9214},  // Eglin AFB (FL)
		{"KBMX", 33.1722, -86.7697},  // Birmingham (AL)
		{"KHTX", 34.9306, -86.0833},  // Huntsville (AL)
		{"KMOB", 30.6797, -88.2397},  // Mobile (AL)
		{"KFFC", 33.3636, -84.5658},  // Atlanta
		{"KJGX", 32.6753, -83.3511},  // Robins AFB (GA)
		{"KVAX", 30.8903, -83.0019},  // Moody AFB (GA)
		{"KCAE", 33.9486, -81.1186},  // Columbia (SC)
		{"KCLX", 32.6556, -81.0422},  // Charleston (SC)
		{"KGSP", 34.8833, -82.2200},  // Greenville (SC)
		{"KRAX", 35.6653, -78.4897},  // Raleigh
		{"KMHX", 34.7761, -76.8764},  // Morehead City (NC)
		{"KLTX", 33.9894, -78.4292},  // Wilmington (NC)
		// Central
		{"KLOT", 41.6045, -88.0847},  // Chicago
		{"KILX", 40.1506, -89.3369},  // Lincoln (IL)
		{"KLSX", 38.6986, -90.6828},  // St. Louis
		{"KSGF", 37.2355, -93.4003},  // Springfield (MO)
		{"KEAX", 38.8103, -94.2644},  // Kansas City
		{"KLVX", 37.9753, -85.9439},  // Louisville
		{"KJKL", 37.5908, -83.3131},  // Jackson (KY)
		{"KHPX", 36.7369, -87.2850},  // Fort Campbell (KY)
		{"KILN", 39.4203, -83.8217},  // Wilmington (OH)
		{"KCLE", 41.4131, -81.8597},  // Cleveland
		{"KIND", 39.7075, -86.2803},  // Indianapolis
		{"KIWX", 41.3586, -85.7000},  // North Webster (IN)
		{"KVWX", 38.2603, -87.7247},  // Evansville (IN)
		{"KDTX", 42.6997, -83.4717},  // Detroit
		{"KAPX", 44.9072, -84.7197},  // Gaylord (MI)
		{"KGRR", 42.8939, -85.5450},  // Grand Rapids
		{"KMQT", 46.5314, -87.5486},  // Marquette (MI)
		{"KARX", 43.8228, -91.1911},  // La Crosse (WI)
		{"KGRB", 44.4986, -88.1111},  // Green Bay
		{"KMKX", 42.9678, -88.5506},  // Milwaukee
		{"KDMX", 41.7311, -93.7228},  // Des Moines
		{"KDVN", 41.6117, -90.5808},  // Davenport (IA)
		{"KMPX", 44.8489, -93.5653},  // Minneapolis
		{"KDLH", 46.8369, -92.2097},  // Duluth
		// Southern
		{"KFWS", 32.5731, -97.3031},  // Dallas/Fort Worth
		{"KEWX", 29.7039, -98.0286},  // San Antonio
		{"KHGX", 29.4719, -95.0792},  // Houston
		{"KCRP", 27.7842, -97.5111},  // Corpus Christi
		{"KBRO", 25.9161, -97.4189},  // Brownsville
		{"KSJT", 31.3714, -100.4925}, // San Angelo (TX)
		{"KLBB", 33.6542, -101.8142}, // Lubbock
		{"KMAF", 31.9433, -102.1894}, // Midland (TX)
		{"KAMA", 35.2333, -101.7092}, // Amarillo
		{"KDYX", 32.5386, -99.2542},  // Dyess AFB (TX)
		{"KEPZ", 31.8731, -106.6981}, // El Paso
		{"KSHV", 32.4508, -93.8411},  // Shreveport
		{"KLCH", 30.1253, -93.2156},  // Lake Charles (LA)
		{"KLIX", 30.3367, -89.8256},  // New Orleans
		{"KPOE", 31.1556, -92.9761},  // Fort Polk (LA)
		{"KDGX", 32.2800, -89.9844},  // Jackson (MS)
		{"KGWX", 33.8967, -88.3292},  // Columbus AFB (MS)
		{"KNQA", 35.3447, -89.8733},  // Memphis
		{"KOHX", 36.2472, -86.5625},  // Nashville
		{"KMRX", 36.1686, -83.4017},  // Knoxville
		// Plains
		{"KUEX", 40.3208, -98.4417},  // Hastings (NE)
		{"KOAX", 41.3203, -96.3667},  // Omaha
		{"KLNX", 41.9578, -100.5761}, // North Platte (NE)
		{"KABR", 45.4558, -98.4131},  // Aberdeen (SD)
		{"KUDX", 44.1250, -102.8297}, // Rapid City
		{"KFSD", 43.5878, -96.7292},  // Sioux Falls
		{"KBIS", 46.7706, -100.7606}, // Bismarck
		{"KMBX", 48.3925, -100.8644}, // Minot AFB (ND)
		{"KMVX", 47.5278, -97.3256},  // Grand Forks (ND)
		{"KICT", 37.6544, -97.4431},  // Wichita
		{"KDDC", 37.7608, -99.9686},  // Dodge City
		{"KTWX", 38.9969, -96.2325},  // Topeka
		{"KINX", 36.1750, -95.5644},  // Tulsa
		{"KTLX", 35.3331, -97.2775},  // Oklahoma City
		{"KVNX", 36.7406, -98.1275},  // Vance AFB (OK)
		{"KFDR", 34.3622, -98.9764},  // Frederick (OK)
		// Mountain West
		{"KTFX", 47.4595, -111.3855}, // Great Falls (MT)
		{"KGGW", 48.2064, -106.6253}, // Glasgow (MT)
		{"KBLX", 45.8536, -108.6069}, // Billings
		{"KMSX", 47.0411, -113.9864}, // Missoula
		{"KCYS", 41.1519, -104.8061}, // Cheyenne
		{"KRIW", 43.0661, -108.4772}, // Riverton (WY)
		{"KFTG", 39.7867, -104.5458}, // Denver
		{"KPUX", 38.4595, -104.1817}, // Pueblo (CO)
		{"KGJX", 39.0622, -108.2139}, // Grand Junction
		{"KABX", 35.1497, -106.8239}, // Albuquerque
		{"KFDX", 34.6353, -103.6297}, // Cannon AFB (NM)
		{"KHDX", 33.0769, -106.1231}, // Holloman AFB (NM)
		{"KSLC", 40.9722, -111.9300}, // Salt Lake City
		{"KMTX", 41.2628, -112.4478}, // Salt Lake City (Promontory)
		{"KICX", 37.5908, -112.8622}, // Cedar City (UT)
		{"KPHX", 33.4301, -112.0128}, // Phoenix
		{"KIWA", 33.2892, -111.6700}, // Phoenix Mesa
		{"KEMX", 31.8936, -110.6303}, // Tucson
		{"KYUX", 32.4953, -114.6567}, // Yuma
		{"KFSX", 34.5744, -111.1983}, // Flagstaff
		// Pacific West
		{"KATX", 48.1945, -122.4958}, // Seattle
		{"KLGX", 47.1158, -124.1069}, // Langley Hill (WA)
		{"KOTX", 47.6803, -117.6267}, // Spokane
		{"KRTX", 45.7150, -122.9650}, // Portland (OR)
		{"KPDT", 45.6906, -118.8528}, // Pendleton (OR)
		{"KMAX", 42.0811, -122.7172}, // Medford (OR)
		{"KCBX", 43.4908, -116.2364}, // Boise
		{"KSFX", 43.1058, -112.6861}, // Pocatello (ID)
		{"KRGX", 39.7542, -119.4622}, // Reno
		{"KLRX", 40.7397, -116.8025}, // Elko (NV)
		{"KESX", 35.7011, -114.8917}, // Las Vegas
		{"KBBX", 39.4961, -121.6317}, // Sacramento
		{"KDAX", 38.5011, -121.6778}, // Sacramento (Davis)
		{"KMUX", 37.1553, -121.8983}, // San Francisco
		{"KHNX", 36.3142, -119.6319}, // Hanford (CA)
		{"KVTX", 34.4117, -119.1792}, // Los Angeles
		{"KSOX", 33.8178, -117.6358}, // Santa Ana Mtns (CA)
		{"KNKX", 32.9189, -117.0419}, // San Diego
		{"KVBX", 34.8383, -120.3978}, // Vandenberg (CA)
		// Alaska & Hawaii
		{"PACG", 56.8525, -135.5292}, // Juneau (AK)
		{"PAPD", 65.0350, -147.5014}, // Fairbanks (AK)
		{"PAHG", 60.7258, -151.3514}, // Kenai (AK)
		{"PAKC", 58.6794, -156.6294}, // King Salmon (AK)
		{"PABC", 60.7919, -161.8764}, // Bethel (AK)
		{"PAEC", 64.5114, -165.2950}, // Nome (AK)
		{"PHKI", 21.8939, -159.5522}, // Kauai (HI)
		{"PHKM", 20.1253, -155.7781}, // Kohala (HI)
		{"PHMO", 21.1328, -157.1803}, // Molokai (HI)
		{"PHWA", 19.0950, -155.5689}, // South Shore (HI)
		// Caribbean
		{"TJUA", 18.1156, -66.0781},  // San Juan (PR)
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