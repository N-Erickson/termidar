package geography

import (
	"math"

	"github.com/charmbracelet/lipgloss"

	"github.com/N-Erickson/termidar/internal/config"
	"github.com/N-Erickson/termidar/internal/weather"
)

// DrawGeographicBoundaries draws state borders, rivers, mountains, and coastlines on the radar display
func DrawGeographicBoundaries(display [][]string, centerX, centerY int, zipCode string) {
	// Get lat/lon to determine what features to draw
	lat, lon, _, _, err := weather.GeocodeZip(zipCode)
	if err != nil {
		// If geocoding fails, just draw the center marker
		if centerY >= 0 && centerY < len(display) && centerX >= 0 && centerX < len(display[0]) {
			display[centerY][centerX] = lipgloss.NewStyle().
				Foreground(lipgloss.Color("226")).
				Bold(true).
				Render("★")
		}
		return
	}

	boundaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	waterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	mountainStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("94"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Scale: approximately 1 character = 4-5 miles
	milesPerCharX := 250.0 / float64(config.RadarWidth)
	milesPerCharY := 150.0 / float64(config.RadarHeight)

	// Helper functions first
	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	abs := func(a int) int {
		if a < 0 {
			return -a
		}
		return a
	}

	// Safe bounds checking function
	inBounds := func(x, y int) bool {
		return y >= 0 && y < len(display) && x >= 0 && x < len(display[0])
	}

	// Helper function to convert lat/lon to display coordinates
	latLonToDisplay := func(targetLat, targetLon float64) (int, int) {
		milesPerDegreeLat := 69.0
		milesPerDegreeLon := 69.0 * math.Cos(lat*math.Pi/180)

		deltaLat := targetLat - lat
		deltaLon := targetLon - lon

		milesNorth := deltaLat * milesPerDegreeLat
		milesEast := deltaLon * milesPerDegreeLon

		x := centerX + int(milesEast/milesPerCharX)
		y := centerY - int(milesNorth/milesPerCharY)

		return x, y
	}

	// Safe drawing helper that checks bounds
	safeDrawPoint := func(x, y int, char string, style *lipgloss.Style) {
		if inBounds(x, y) {
			display[y][x] = style.Render(char)
		}
	}

	// Safe line drawing function
	drawLine := func(x1, y1, x2, y2 int, char string, style *lipgloss.Style, skipExisting bool) {
		// Clip line to display bounds
		if (x1 < 0 && x2 < 0) || (x1 >= config.RadarWidth && x2 >= config.RadarWidth) ||
			(y1 < 0 && y2 < 0) || (y1 >= config.RadarHeight && y2 >= config.RadarHeight) {
			return
		}

		// Simple clipping
		x1 = max(0, min(config.RadarWidth-1, x1))
		x2 = max(0, min(config.RadarWidth-1, x2))
		y1 = max(0, min(config.RadarHeight-1, y1))
		y2 = max(0, min(config.RadarHeight-1, y2))

		if x1 == x2 { // Vertical line
			if y1 > y2 {
				y1, y2 = y2, y1
			}
			for y := y1; y <= y2; y++ {
				if skipExisting && inBounds(x1, y) && display[y][x1] != " " {
					continue
				}
				safeDrawPoint(x1, y, char, style)
			}
		} else if y1 == y2 { // Horizontal line
			if x1 > x2 {
				x1, x2 = x2, x1
			}
			for x := x1; x <= x2; x++ {
				if skipExisting && inBounds(x, y1) && display[y1][x] != " " {
					continue
				}
				safeDrawPoint(x, y1, char, style)
			}
		} else { // Diagonal line
			dx := abs(x2 - x1)
			dy := abs(y2 - y1)
			sx := 1
			sy := 1
			if x1 > x2 {
				sx = -1
			}
			if y1 > y2 {
				sy = -1
			}
			err := dx - dy

			x, y := x1, y1
			for {
				if skipExisting && inBounds(x, y) && display[y][x] != " " {
					// Skip this point
				} else {
					safeDrawPoint(x, y, char, style)
				}

				if x == x2 && y == y2 {
					break
				}

				e2 := 2 * err
				if e2 > -dy {
					err -= dy
					x += sx
				}
				if e2 < dx {
					err += dx
					y += sy
				}
			}
		}
	}

	// Draw state borders using actual state boundary data
	drawStateBorders := func() {
		// Define key state boundary points (simplified)
		borders := [][]float64{
			// === WESTERN STATES ===
			// Washington borders
			{49.0, -117.03, 49.0, -123.0}, // WA-Canada border
			{49.0, -123.0, 48.5, -124.7},  // WA Pacific coast
			{48.5, -124.7, 46.0, -124.0},  // WA Pacific coast
			{46.0, -124.0, 46.0, -117.03}, // WA-OR border
			{46.0, -117.03, 49.0, -117.03}, // WA-ID border
		
			// Oregon borders  
			{46.0, -117.03, 46.0, -124.0}, // OR-WA border
			{46.0, -124.0, 42.0, -124.4},  // OR Pacific coast
			{42.0, -124.4, 42.0, -117.02}, // OR-CA border
			{42.0, -117.02, 45.5, -117.02}, // OR-ID border
			{45.5, -117.02, 46.0, -117.03}, // OR-ID border (northeast)
		
			// California borders
			{42.0, -120.0, 42.0, -124.4},  // CA-OR border
			{42.0, -120.0, 39.0, -120.0},  // CA-NV border (north)
			{39.0, -120.0, 35.0, -119.5},  // CA Sierra Nevada line
			{35.0, -119.5, 35.0, -114.6},  // CA-NV border (south)
			{35.0, -114.6, 32.5, -114.5},  // CA-AZ border
			{32.5, -114.5, 32.5, -117.1},  // CA-Mexico border
			{32.5, -117.1, 42.0, -124.4},  // CA Pacific coast (simplified)
		
			// Idaho borders
			{49.0, -117.03, 49.0, -116.05}, // ID-Canada border
			{49.0, -116.05, 44.5, -111.05}, // ID-MT border
			{44.5, -111.05, 42.0, -111.05}, // ID-WY border
			{42.0, -111.05, 42.0, -114.0},  // ID-UT border
			{42.0, -114.0, 42.0, -117.02},  // ID-NV/OR border
			{42.0, -117.02, 49.0, -117.03}, // ID-WA border
		
			// Nevada borders
			{42.0, -120.0, 42.0, -114.0},  // NV-OR/ID border
			{42.0, -114.0, 37.0, -114.0},  // NV-UT border
			{37.0, -114.0, 35.0, -114.6},  // NV-AZ border
			{35.0, -114.6, 35.0, -120.0},  // NV-CA border (south)
			{35.0, -120.0, 39.0, -120.0},  // NV-CA border (west)
			{39.0, -120.0, 42.0, -120.0},  // NV-CA border (north)
		
			// Utah borders
			{42.0, -114.0, 42.0, -111.05}, // UT-ID border
			{42.0, -111.05, 41.0, -111.05}, // UT-WY border
			{41.0, -111.05, 41.0, -109.05}, // UT-WY border (south)
			{41.0, -109.05, 37.0, -109.05}, // UT-CO border
			{37.0, -109.05, 37.0, -114.0},  // UT-AZ border
			{37.0, -114.0, 42.0, -114.0},   // UT-NV border
		
			// Arizona borders
			{37.0, -114.0, 37.0, -109.05},  // AZ-UT border
			{37.0, -109.05, 31.33, -109.05}, // AZ-NM border
			{31.33, -109.05, 31.33, -111.07}, // AZ-Mexico border (east)
			{31.33, -111.07, 31.33, -114.81}, // AZ-Mexico border
			{31.33, -114.81, 32.5, -114.5},   // AZ-CA border (junction)
			{32.5, -114.5, 35.0, -114.6},     // AZ-CA border
			{35.0, -114.6, 37.0, -114.0},     // AZ-NV border
		
			// === MOUNTAIN STATES ===
			// Montana borders
			{49.0, -116.05, 49.0, -104.03}, // MT-Canada border
			{49.0, -104.03, 45.0, -104.03}, // MT-ND border
			{45.0, -104.03, 45.0, -111.05}, // MT-WY border
			{45.0, -111.05, 48.5, -116.05}, // MT-ID border
			{48.5, -116.05, 49.0, -116.05}, // MT-ID border (north)
		
			// Wyoming borders
			{45.0, -111.05, 45.0, -104.05}, // WY-MT border
			{45.0, -104.05, 41.0, -104.05}, // WY-SD/NE border
			{41.0, -104.05, 41.0, -111.05}, // WY-CO/UT border
			{41.0, -111.05, 45.0, -111.05}, // WY-ID border
		
			// Colorado borders
			{41.0, -109.05, 41.0, -102.05}, // CO-WY/NE border
			{41.0, -102.05, 37.0, -102.05}, // CO-NE/KS border
			{37.0, -102.05, 37.0, -109.05}, // CO-KS/OK/NM border
			{37.0, -109.05, 41.0, -109.05}, // CO-UT border
		
			// New Mexico borders
			{37.0, -109.05, 37.0, -103.0},  // NM-CO border
			{37.0, -103.0, 32.0, -103.0},   // NM-OK/TX border
			{32.0, -103.0, 32.0, -106.5},   // NM-TX border
			{32.0, -106.5, 31.78, -106.5},  // NM-TX border (El Paso)
			{31.78, -106.5, 31.78, -108.2}, // NM-Mexico border
			{31.78, -108.2, 31.33, -109.05}, // NM-Mexico border
			{31.33, -109.05, 37.0, -109.05}, // NM-AZ border
		
			// === MIDWEST STATES ===
			// North Dakota borders
			{49.0, -104.03, 49.0, -97.23},  // ND-Canada border
			{49.0, -97.23, 45.94, -96.56},  // ND-MN border
			{45.94, -96.56, 45.94, -104.03}, // ND-SD border
			{45.94, -104.03, 49.0, -104.03}, // ND-MT border
		
			// South Dakota borders
			{45.94, -104.03, 45.94, -96.44}, // SD-ND border
			{45.94, -96.44, 43.5, -96.44},   // SD-MN/IA border
			{43.5, -96.44, 43.0, -96.44},    // SD-IA border
			{43.0, -96.44, 43.0, -104.05},   // SD-NE border
			{43.0, -104.05, 45.94, -104.03}, // SD-WY/MT border
		
			// Nebraska borders
			{43.0, -104.05, 43.0, -96.44},  // NE-SD border
			{43.0, -96.44, 40.0, -95.31},   // NE-IA border
			{40.0, -95.31, 40.0, -102.05},  // NE-KS border
			{40.0, -102.05, 41.0, -102.05}, // NE-CO border
			{41.0, -102.05, 41.0, -104.05}, // NE-WY border
			{41.0, -104.05, 43.0, -104.05}, // NE-WY border
		
			// Kansas borders
			{40.0, -102.05, 40.0, -94.62},  // KS-NE border
			{40.0, -94.62, 39.0, -94.62},   // KS-MO border
			{39.0, -94.62, 37.0, -94.62},   // KS-MO border
			{37.0, -94.62, 37.0, -102.05},  // KS-OK border
			{37.0, -102.05, 40.0, -102.05}, // KS-CO border
		
			// Minnesota borders
			{49.0, -97.23, 49.0, -95.15},   // MN-Canada border
			{49.0, -95.15, 49.0, -89.53},   // MN-Canada border (east)
			{48.0, -89.53, 47.5, -92.3},    // MN-Lake Superior
			{47.5, -92.3, 46.5, -92.3},     // MN-WI border
			{46.5, -92.3, 45.5, -92.3},     // MN-WI border
			{45.5, -92.3, 43.5, -91.22},    // MN-WI border
			{43.5, -91.22, 43.5, -96.44},   // MN-IA border
			{43.5, -96.44, 45.94, -96.56},  // MN-SD border
			{45.94, -96.56, 49.0, -97.23},  // MN-ND border
		
			// Iowa borders
			{43.5, -96.44, 43.5, -91.22},   // IA-MN border
			{43.5, -91.22, 42.5, -90.64},   // IA-WI border
			{42.5, -90.64, 40.38, -91.41},  // IA-IL border
			{40.38, -91.41, 40.58, -95.77}, // IA-MO border
			{40.58, -95.77, 43.0, -96.44},  // IA-NE border
			{43.0, -96.44, 43.5, -96.44},   // IA-SD border
		
			// Missouri borders
			{40.58, -95.77, 40.38, -91.41}, // MO-IA border
			{40.38, -91.41, 36.5, -89.5},   // MO-IL border
			{36.5, -89.5, 36.0, -89.5},     // MO-KY/TN border
			{36.0, -89.5, 36.5, -90.37},    // MO-AR border
			{36.5, -90.37, 36.5, -94.62},   // MO-AR border
			{36.5, -94.62, 37.0, -94.62},   // MO-OK border
			{37.0, -94.62, 39.0, -94.62},   // MO-KS border
			{39.0, -94.62, 40.58, -95.77},  // MO-KS/NE border
		
			// Wisconsin borders
			{46.5, -92.3, 45.5, -92.3},     // WI-MN border
			{45.5, -92.3, 43.5, -91.22},    // WI-MN border
			{43.5, -91.22, 42.5, -90.64},   // WI-IA border
			{42.5, -90.64, 42.5, -87.02},   // WI-IL border
			{42.5, -87.02, 45.0, -87.0},    // WI-Lake Michigan
			{45.0, -87.0, 45.5, -88.0},     // WI-MI border
			{45.5, -88.0, 46.5, -90.0},     // WI-MI border
			{46.5, -90.0, 46.5, -92.3},     // WI-Lake Superior
		
			// Illinois borders
			{42.5, -90.64, 42.5, -87.02},   // IL-WI border
			{42.5, -87.02, 41.76, -87.53},  // IL-Lake Michigan
			{41.76, -87.53, 39.0, -87.5},   // IL-IN border
			{39.0, -87.5, 37.0, -88.1},     // IL-IN border
			{37.0, -88.1, 37.0, -89.15},    // IL-KY border
			{37.0, -89.15, 36.5, -89.5},    // IL-MO border
			{36.5, -89.5, 40.38, -91.41},   // IL-MO border
			{40.38, -91.41, 42.5, -90.64},  // IL-IA border
		
			// Michigan borders
			{45.0, -87.0, 45.5, -88.0},     // MI-WI border
			{45.5, -88.0, 46.5, -90.0},     // MI-WI border
			{46.5, -90.0, 47.5, -89.0},     // MI-Lake Superior
			{42.0, -86.5, 45.0, -87.0},     // MI-Lake Michigan
			{41.76, -84.8, 42.0, -83.0},    // MI-OH border
			{42.0, -83.0, 42.5, -82.4},     // MI-Canada border
		
			// Indiana borders
			{41.76, -87.53, 41.76, -84.8},  // IN-MI border
			{41.76, -84.8, 39.0, -84.8},    // IN-OH border
			{39.0, -84.8, 38.0, -86.0},     // IN-KY border
			{38.0, -86.0, 37.0, -88.1},     // IN-KY border
			{37.0, -88.1, 39.0, -87.5},     // IN-IL border
			{39.0, -87.5, 41.76, -87.53},   // IN-IL border
		
			// Ohio borders
			{41.76, -84.8, 41.97, -80.52},  // OH-MI/PA border
			{41.97, -80.52, 40.64, -80.52}, // OH-PA border
			{40.64, -80.52, 39.0, -81.0},   // OH-WV border
			{39.0, -81.0, 38.5, -82.0},     // OH-WV border
			{38.5, -82.0, 38.5, -84.8},     // OH-KY border
			{38.5, -84.8, 39.0, -84.8},     // OH-KY border
			{39.0, -84.8, 41.76, -84.8},    // OH-IN border
		
			// === SOUTHERN STATES ===
			// Texas borders
			{36.5, -103.0, 36.5, -100.0},   // TX-OK border (panhandle)
			{36.5, -100.0, 34.0, -100.0},   // TX-OK border (Red River)
			{34.0, -100.0, 33.5, -94.04},   // TX-OK border (Red River)
			{33.5, -94.04, 31.17, -94.04},  // TX-LA border
			{31.17, -94.04, 29.5, -93.84},  // TX-LA border (Sabine)
			{29.5, -93.84, 26.0, -97.14},   // TX Gulf Coast
			{26.0, -97.14, 25.84, -97.14},  // TX-Mexico border (Gulf)
			{25.84, -97.14, 31.78, -106.5}, // TX-Mexico border (Rio Grande)
			{31.78, -106.5, 32.0, -106.5},  // TX-NM border
			{32.0, -106.5, 32.0, -103.0},   // TX-NM border
			{32.0, -103.0, 36.5, -103.0},   // TX-NM/OK border
		
			// Oklahoma borders
			{37.0, -103.0, 37.0, -94.62},   // OK-KS border
			{37.0, -94.62, 36.5, -94.62},   // OK-MO border
			{36.5, -94.62, 35.0, -94.43},   // OK-AR border
			{35.0, -94.43, 33.5, -94.04},   // OK-AR border
			{33.5, -94.04, 34.0, -100.0},   // OK-TX border (Red River)
			{34.0, -100.0, 36.5, -100.0},   // OK-TX border
			{36.5, -100.0, 36.5, -103.0},   // OK-TX border (panhandle)
			{36.5, -103.0, 37.0, -103.0},   // OK-NM/CO border
		
			// Arkansas borders
			{36.5, -94.62, 36.5, -90.37},   // AR-MO border
			{36.5, -90.37, 35.0, -90.0},    // AR-TN border
			{35.0, -90.0, 35.0, -91.0},     // AR-MS border
			{35.0, -91.0, 33.0, -91.2},     // AR-LA border
			{33.0, -91.2, 33.0, -94.04},    // AR-LA border
			{33.0, -94.04, 35.0, -94.43},   // AR-TX border
			{35.0, -94.43, 36.5, -94.62},   // AR-OK border
		
			// Louisiana borders
			{33.0, -94.04, 33.0, -91.2},    // LA-AR border
			{33.0, -91.2, 31.0, -91.5},     // LA-MS border
			{31.0, -91.5, 30.0, -89.5},     // LA-MS border
			{30.0, -89.5, 29.0, -89.0},     // LA Gulf Coast
			{29.0, -89.0, 29.5, -93.84},    // LA Gulf Coast
			{29.5, -93.84, 31.17, -94.04},  // LA-TX border
			{31.17, -94.04, 33.0, -94.04},  // LA-TX border
		
			// Mississippi borders
			{35.0, -91.0, 35.0, -88.2},     // MS-TN border
			{35.0, -88.2, 31.0, -88.47},    // MS-AL border
			{31.0, -88.47, 30.0, -89.5},    // MS Gulf Coast
			{30.0, -89.5, 31.0, -91.5},     // MS-LA border
			{31.0, -91.5, 35.0, -91.0},     // MS-LA border
		
			// Alabama borders
			{35.0, -88.2, 35.0, -85.0},     // AL-TN border
			{35.0, -85.0, 32.9, -85.0},     // AL-GA border
			{32.9, -85.0, 31.0, -85.0},     // AL-FL border
			{31.0, -85.0, 30.0, -87.5},     // AL-FL border
			{30.0, -87.5, 30.0, -88.47},    // AL Gulf Coast
			{30.0, -88.47, 31.0, -88.47},   // AL-MS border
			{31.0, -88.47, 35.0, -88.2},    // AL-MS border
		
			// Tennessee borders
			{36.5, -90.37, 36.5, -81.65},   // TN-KY/VA border
			{36.5, -81.65, 35.0, -84.32},   // TN-NC border
			{35.0, -84.32, 35.0, -85.0},    // TN-GA border
			{35.0, -85.0, 35.0, -88.2},     // TN-AL border
			{35.0, -88.2, 35.0, -90.0},     // TN-MS border
			{35.0, -90.0, 36.5, -90.37},    // TN-AR border
		
			// Kentucky borders
			{39.0, -84.8, 38.5, -84.8},     // KY-OH border
			{38.5, -84.8, 38.5, -82.0},     // KY-OH border
			{38.5, -82.0, 37.5, -82.5},     // KY-WV border
			{37.5, -82.5, 36.5, -83.68},    // KY-VA border
			{36.5, -83.68, 36.5, -89.5},    // KY-TN border
			{36.5, -89.5, 37.0, -89.15},    // KY-MO border
			{37.0, -89.15, 37.0, -88.1},    // KY-IL border
			{37.0, -88.1, 38.0, -86.0},     // KY-IN border
			{38.0, -86.0, 39.0, -84.8},     // KY-IN border
		
			// Florida borders
			{31.0, -87.5, 31.0, -85.0},     // FL-AL border
			{31.0, -85.0, 30.5, -84.86},    // FL-GA border
			{30.5, -84.86, 30.0, -82.0},    // FL-GA border
			{30.0, -82.0, 25.0, -80.0},     // FL Atlantic coast
			{25.0, -80.0, 24.5, -81.8},     // FL Keys
			{24.5, -81.8, 30.0, -87.5},     // FL Gulf coast
			{30.0, -87.5, 31.0, -87.5},     // FL-AL border
		
			// Georgia borders
			{35.0, -85.0, 35.0, -83.5},     // GA-TN/NC border
			{35.0, -83.5, 32.0, -81.0},     // GA-SC border
			{32.0, -81.0, 30.5, -81.5},     // GA Atlantic coast
			{30.5, -81.5, 30.0, -82.0},     // GA-FL border
			{30.0, -82.0, 30.5, -84.86},    // GA-FL border
			{30.5, -84.86, 32.9, -85.0},    // GA-AL border
			{32.9, -85.0, 35.0, -85.0},     // GA-AL border
		
			// South Carolina borders
			{35.0, -83.5, 35.2, -80.5},     // SC-NC border
			{35.2, -80.5, 33.5, -79.0},     // SC Atlantic coast
			{33.5, -79.0, 32.0, -81.0},     // SC-GA border
			{32.0, -81.0, 35.0, -83.5},     // SC-GA border
		
			// North Carolina borders
			{36.5, -83.68, 36.5, -75.5},    // NC-VA border
			{36.5, -75.5, 35.5, -75.5},     // NC Atlantic coast
			{35.5, -75.5, 33.5, -79.0},     // NC Atlantic coast
			{33.5, -79.0, 35.2, -80.5},     // NC-SC border
			{35.2, -80.5, 35.0, -84.32},    // NC-TN border
			{35.0, -84.32, 36.5, -83.68},   // NC-TN/VA border
		
			// Virginia borders
			{39.0, -77.52, 39.0, -75.5},    // VA-MD border
			{39.0, -75.5, 38.0, -75.5},     // VA Atlantic coast
			{38.0, -75.5, 36.5, -75.5},     // VA Atlantic coast
			{36.5, -75.5, 36.5, -83.68},    // VA-NC border
			{36.5, -83.68, 37.5, -82.5},    // VA-KY border
			{37.5, -82.5, 39.0, -80.52},    // VA-WV border
			{39.0, -80.52, 39.0, -77.52},   // VA-MD border
		
			// West Virginia borders
			{40.64, -80.52, 39.72, -79.48}, // WV-PA border
			{39.72, -79.48, 39.0, -77.52},  // WV-MD border
			{39.0, -77.52, 39.0, -80.52},   // WV-VA border
			{39.0, -80.52, 37.5, -82.5},    // WV-VA border
			{37.5, -82.5, 38.5, -82.0},     // WV-KY border
			{38.5, -82.0, 39.0, -81.0},     // WV-OH border
			{39.0, -81.0, 40.64, -80.52},   // WV-OH/PA border
		
			// === NORTHEASTERN STATES ===
			// Pennsylvania borders
			{42.0, -80.52, 42.0, -79.76},   // PA-NY border (west)
			{42.0, -79.76, 41.99, -75.35},  // PA-NY border
			{41.99, -75.35, 41.0, -75.1},   // PA-NJ border
			{41.0, -75.1, 39.72, -75.79},   // PA-DE border
			{39.72, -75.79, 39.72, -79.48}, // PA-MD border
			{39.72, -79.48, 40.64, -80.52}, // PA-WV border
			{40.64, -80.52, 42.0, -80.52},  // PA-OH border
		
			// New York borders
			{45.01, -74.75, 45.01, -71.5},  // NY-Canada border
			{45.01, -71.5, 42.73, -71.5},   // NY-VT border
			{42.73, -71.5, 42.0, -73.35},   // NY-MA/CT border
			{42.0, -73.35, 41.0, -73.9},    // NY-CT border
			{41.0, -73.9, 40.7, -74.0},     // NY Atlantic coast
			{40.7, -74.0, 41.0, -75.1},     // NY-NJ border
			{41.0, -75.1, 41.99, -75.35},   // NY-PA border
			{41.99, -75.35, 42.0, -79.76},  // NY-PA border
			{42.0, -79.76, 45.01, -74.75},  // NY-Canada border
		
			// New Jersey borders
			{41.36, -74.7, 41.0, -73.9},    // NJ-NY border
			{41.0, -73.9, 40.7, -74.0},     // NJ-NY border
			{40.7, -74.0, 39.0, -74.5},     // NJ Atlantic coast
			{39.0, -74.5, 38.8, -75.2},     // NJ-DE border
			{38.8, -75.2, 39.72, -75.79},   // NJ-DE/PA border
			{39.72, -75.79, 41.0, -75.1},   // NJ-PA border
			{41.0, -75.1, 41.36, -74.7},    // NJ-NY border
		
			// Delaware borders
			{39.84, -75.79, 39.72, -75.79}, // DE-PA border
			{39.72, -75.79, 38.8, -75.2},   // DE-NJ border
			{38.8, -75.2, 38.45, -75.05},   // DE Atlantic coast
			{38.45, -75.05, 38.45, -75.79}, // DE-MD border
			{38.45, -75.79, 39.84, -75.79}, // DE-MD border
		
			// Maryland borders
			{39.72, -79.48, 39.72, -75.79}, // MD-PA border
			{39.72, -75.79, 38.45, -75.79}, // MD-DE border
			{38.45, -75.79, 38.0, -76.0},   // MD Chesapeake Bay
			{38.0, -76.0, 38.0, -77.0},     // MD-VA border
			{38.0, -77.0, 39.0, -77.52},    // MD-VA/WV border
			{39.0, -77.52, 39.72, -79.48},  // MD-WV border
		
			// Connecticut borders
			{42.05, -73.48, 42.05, -71.8},  // CT-MA border
			{42.05, -71.8, 41.3, -71.85},   // CT-RI border
			{41.3, -71.85, 41.0, -72.0},    // CT Long Island Sound
			{41.0, -72.0, 41.0, -73.9},     // CT-NY border
			{41.0, -73.9, 42.0, -73.35},    // CT-NY border
			{42.0, -73.35, 42.05, -73.48},  // CT-MA border
		
			// Rhode Island borders
			{42.01, -71.38, 42.01, -71.12}, // RI-MA border
			{42.01, -71.12, 41.3, -71.12},  // RI Atlantic coast
			{41.3, -71.12, 41.3, -71.85},   // RI-CT border
			{41.3, -71.85, 42.01, -71.8},   // RI-CT border
			{42.01, -71.8, 42.01, -71.38},  // RI-MA border
		
			// Massachusetts borders
			{42.88, -73.26, 42.75, -71.0},  // MA-NH/VT border
			{42.75, -71.0, 42.88, -70.5},   // MA-NH border
			{42.88, -70.5, 42.0, -70.0},    // MA Atlantic coast
			{42.0, -70.0, 41.5, -71.12},    // MA Atlantic coast
			{41.5, -71.12, 42.01, -71.38},  // MA-RI border
			{42.01, -71.38, 42.05, -71.8},  // MA-CT border
			{42.05, -71.8, 42.05, -73.48},  // MA-CT border
			{42.05, -73.48, 42.88, -73.26}, // MA-NY border
		
			// Vermont borders
			{45.01, -71.5, 45.01, -73.35},  // VT-Canada border
			{45.01, -73.35, 42.73, -73.26}, // VT-NY border
			{42.73, -73.26, 42.73, -72.46}, // VT-MA border
			{42.73, -72.46, 42.73, -71.5},  // VT-NH border
			{42.73, -71.5, 45.01, -71.5},   // VT-NH border
		
			// New Hampshire borders
			{45.3, -71.08, 45.3, -71.0},    // NH-Canada border
			{45.3, -71.0, 42.88, -70.5},    // NH-ME border
			{42.88, -70.5, 42.75, -71.0},   // NH-MA border
			{42.75, -71.0, 42.73, -72.46},  // NH-MA border
			{42.73, -72.46, 45.01, -71.5},  // NH-VT border
			{45.01, -71.5, 45.3, -71.08},   // NH-Canada border
		
			// Maine borders
			{47.46, -69.23, 45.3, -71.08},  // ME-Canada border
			{45.3, -71.08, 45.3, -71.0},    // ME-NH border
			{45.3, -71.0, 42.88, -70.5},    // ME-NH border
			{42.88, -70.5, 43.5, -70.0},    // ME Atlantic coast
			{43.5, -70.0, 45.0, -67.0},     // ME Atlantic coast
			{45.0, -67.0, 47.46, -69.23},   // ME-Canada border
		
			// === NON-CONTIGUOUS STATES ===
			// Alaska (simplified box)
			{71.5, -156.5, 71.5, -141.0},  // AK north border
			{71.5, -141.0, 54.5, -130.0},  // AK east border
			{54.5, -130.0, 54.5, -173.0},  // AK south border
			{54.5, -173.0, 71.5, -156.5},  // AK west border
		
			// Hawaii (simplified boxes for main islands)
			{22.2, -159.8, 22.2, -159.3},  // Kauai
			{22.2, -159.3, 21.8, -159.3},
			{21.8, -159.3, 21.8, -159.8},
			{21.8, -159.8, 22.2, -159.8},
		
			{21.1, -156.3, 21.1, -155.9},  // Maui
			{21.1, -155.9, 20.5, -155.9},
			{20.5, -155.9, 20.5, -156.7},
			{20.5, -156.7, 21.1, -156.3},
		
			{21.7, -158.3, 21.7, -157.6},  // Oahu
			{21.7, -157.6, 21.2, -157.6},
			{21.2, -157.6, 21.2, -158.3},
			{21.2, -158.3, 21.7, -158.3},
		
			{19.7, -156.1, 19.7, -154.8},  // Big Island
			{19.7, -154.8, 18.9, -154.8},
			{18.9, -154.8, 18.9, -156.1},
			{18.9, -156.1, 19.7, -156.1},
		}

		// Draw each border segment
		for _, border := range borders {
			startX, startY := latLonToDisplay(border[0], border[1])
			endX, endY := latLonToDisplay(border[2], border[3])

			// Determine if vertical or horizontal
			if abs(startX-endX) < abs(startY-endY) {
				drawLine(startX, startY, endX, endY, "│", &borderStyle, false)
			} else {
				drawLine(startX, startY, endX, endY, "─", &borderStyle, false)
			}
		}

		// Add state abbreviations
		stateLabels := []struct {
			lat, lon float64
			label    string
		}{
			{44.5, -100.0, "SD"}, {41.5, -99.0, "NE"}, {42.0, -93.5, "IA"},
			{46.0, -94.5, "MN"}, {43.0, -89.5, "WI"}, {40.0, -89.0, "IL"},
			{38.5, -98.5, "KS"}, {39.0, -105.5, "CO"}, {44.0, -107.5, "WY"},
			{47.0, -110.0, "MT"}, {46.5, -100.5, "ND"}, {38.5, -92.5, "MO"},
			{35.0, -97.5, "OK"}, {31.0, -99.0, "TX"}, {40.5, -112.0, "UT"},
			{39.0, -119.5, "NV"}, {37.5, -119.5, "CA"}, {44.0, -120.5, "OR"},
			{47.5, -120.5, "WA"}, {43.5, -114.0, "ID"}, {34.5, -106.0, "NM"},
			{34.5, -112.0, "AZ"}, {42.5, -72.5, "VT"}, {43.5, -71.5, "NH"},
			{42.3, -71.8, "MA"}, {41.7, -71.5, "RI"}, {41.6, -72.7, "CT"},
			{43.0, -75.5, "NY"}, {40.5, -74.5, "NJ"}, {41.0, -77.5, "PA"},
			{39.0, -75.5, "DE"}, {39.0, -76.5, "MD"}, {38.0, -79.5, "VA"},
			{35.5, -79.5, "NC"}, {34.0, -81.0, "SC"}, {33.0, -83.5, "GA"},
			{30.5, -84.5, "FL"}, {32.5, -86.5, "AL"}, {32.5, -90.0, "MS"},
			{31.0, -92.0, "LA"}, {35.0, -86.0, "TN"}, {37.5, -84.5, "KY"},
			{40.0, -82.5, "OH"}, {40.0, -86.0, "IN"}, {42.0, -84.5, "MI"},
			{38.5, -81.0, "WV"}, {35.5, -92.5, "AR"},
		}

		// Draw visible state labels
		for _, state := range stateLabels {
			x, y := latLonToDisplay(state.lat, state.lon)
			if inBounds(x, y) && x+len(state.label)-1 < len(display[0]) {
				for i, ch := range state.label {
					if inBounds(x+i, y) {
						display[y][x+i] = boundaryStyle.Render(string(ch))
					}
				}
			}
		}
	}

	// Draw state borders first
	drawStateBorders()

	// Then draw geographic features on top
	// Draw major rivers
	rivers := []struct {
		name string
		path [][]float64
	}{
		{
			"Mississippi",
			[][]float64{
				{47.5, -94.5}, {46.0, -94.0}, {44.0, -92.0}, {42.0, -90.5},
				{40.0, -90.0}, {38.0, -89.5}, {36.0, -89.5}, {34.0, -90.5},
				{32.0, -91.0}, {30.0, -91.0}, {29.0, -89.5},
			},
		},
		{
			"Missouri",
			[][]float64{
				{46.0, -111.5}, {45.5, -110.0}, {44.5, -108.0}, {43.5, -104.0},
				{42.5, -100.0}, {41.5, -96.0}, {40.0, -95.0}, {39.0, -93.5},
				{38.5, -90.5},
			},
		},
		{
			"Colorado",
			[][]float64{
				{36.0, -114.5}, {35.5, -113.0}, {34.5, -111.0}, {33.5, -109.0},
				{32.5, -107.5}, {31.5, -105.5},
			},
		},
		{
			"Rio Grande",
			[][]float64{
				{37.0, -107.0}, {36.0, -106.0}, {34.0, -106.5}, {32.0, -106.5},
				{30.0, -104.0}, {28.0, -102.0}, {26.0, -99.0}, {25.8, -97.2},
			},
		},
	}

	// Draw rivers
	for _, river := range rivers {
		for i := 0; i < len(river.path)-1; i++ {
			x1, y1 := latLonToDisplay(river.path[i][0], river.path[i][1])
			x2, y2 := latLonToDisplay(river.path[i+1][0], river.path[i+1][1])

			steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
			if steps > 0 {
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					x := int(float64(x1) + t*float64(x2-x1))
					y := int(float64(y1) + t*float64(y2-y1))

					if inBounds(x, y) && display[y][x] == " " {
						display[y][x] = waterStyle.Render("~")
					}
				}
			}
		}
	}

	// Draw mountain ranges
	mountains := []struct {
		name string
		path [][]float64
	}{
		{
			"Rockies",
			[][]float64{
				{49.0, -114.0}, {47.0, -113.5}, {45.0, -112.5}, {43.0, -109.0},
				{41.0, -105.5}, {39.0, -105.5}, {37.0, -105.0}, {35.0, -106.0},
			},
		},
		{
			"Cascades",
			[][]float64{
				{49.0, -121.5}, {47.5, -121.5}, {46.0, -121.7}, {44.0, -122.0},
				{42.0, -122.2}, {40.5, -122.0},
			},
		},
		{
			"Sierra Nevada",
			[][]float64{
				{40.5, -121.0}, {39.0, -120.5}, {37.5, -119.0}, {36.0, -118.0},
			},
		},
		{
			"Appalachians",
			[][]float64{
				{44.0, -71.5}, {42.0, -73.5}, {40.0, -75.5}, {38.0, -78.5},
				{36.0, -81.5}, {34.5, -83.5},
			},
		},
	}

	// Draw mountains
	for _, mountain := range mountains {
		for _, point := range mountain.path {
			x, y := latLonToDisplay(point[0], point[1])
			if inBounds(x, y) && display[y][x] == " " {
				display[y][x] = mountainStyle.Render("^")
			}
			// Add some width to mountain ranges
			if inBounds(x-1, y) && display[y][x-1] == " " {
				display[y][x-1] = mountainStyle.Render("^")
			}
			if inBounds(x+1, y) && display[y][x+1] == " " {
				display[y][x+1] = mountainStyle.Render("^")
			}
		}
	}

	// Draw coastlines
	// Atlantic Coast
	if lon > -85 {
		coastPoints := [][]float64{
			{45.0, -67.0}, {44.0, -68.0}, {42.5, -70.0}, {41.0, -71.0},
			{40.5, -73.5}, {39.0, -74.0}, {37.5, -75.5}, {36.0, -76.0},
			{34.0, -78.0}, {32.0, -80.0}, {30.0, -81.0}, {28.0, -80.5},
			{25.5, -80.0}, {24.5, -81.5},
		}

		for i := 0; i < len(coastPoints)-1; i++ {
			x1, y1 := latLonToDisplay(coastPoints[i][0], coastPoints[i][1])
			x2, y2 := latLonToDisplay(coastPoints[i+1][0], coastPoints[i+1][1])

			steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
			if steps > 0 {
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					x := int(float64(x1) + t*float64(x2-x1))
					y := int(float64(y1) + t*float64(y2-y1))

					if inBounds(x, y) && display[y][x] == " " {
						display[y][x] = waterStyle.Render("≈")
					}
				}
			}
		}
	}

	// Pacific Coast
	if lon < -115 {
		coastPoints := [][]float64{
			{48.5, -124.7}, {47.0, -124.0}, {45.0, -124.0}, {43.0, -124.4},
			{41.0, -124.2}, {39.0, -123.8}, {37.0, -122.5}, {35.0, -121.0},
			{33.5, -118.0}, {32.5, -117.2},
		}

		for i := 0; i < len(coastPoints)-1; i++ {
			x1, y1 := latLonToDisplay(coastPoints[i][0], coastPoints[i][1])
			x2, y2 := latLonToDisplay(coastPoints[i+1][0], coastPoints[i+1][1])

			steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
			if steps > 0 {
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					x := int(float64(x1) + t*float64(x2-x1))
					y := int(float64(y1) + t*float64(y2-y1))

					if inBounds(x, y) && display[y][x] == " " {
						display[y][x] = waterStyle.Render("≈")
					}
				}
			}
		}
	}

	// Gulf Coast
	if lat < 33 && lon > -98 {
		coastPoints := [][]float64{
			{30.0, -87.5}, {29.5, -89.0}, {29.0, -91.0}, {28.5, -93.0},
			{27.5, -95.0}, {26.5, -97.0}, {25.8, -97.2},
		}

		for i := 0; i < len(coastPoints)-1; i++ {
			x1, y1 := latLonToDisplay(coastPoints[i][0], coastPoints[i][1])
			x2, y2 := latLonToDisplay(coastPoints[i+1][0], coastPoints[i+1][1])

			steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
			if steps > 0 {
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					x := int(float64(x1) + t*float64(x2-x1))
					y := int(float64(y1) + t*float64(y2-y1))

					if inBounds(x, y) && display[y][x] == " " {
						display[y][x] = waterStyle.Render("≈")
					}
				}
			}
		}
	}

	// Great Lakes
	if lon > -93 && lon < -75 && lat > 41 && lat < 49 {
		// Lake Superior
		if lat > 46 {
			lakePoints := [][]float64{
				{48.0, -89.5}, {47.5, -91.0}, {46.5, -92.0}, {46.5, -94.0},
				{47.0, -92.5}, {47.5, -90.5}, {48.0, -89.5},
			}
			for _, point := range lakePoints {
				x, y := latLonToDisplay(point[0], point[1])
				if inBounds(x, y) {
					display[y][x] = waterStyle.Render("≈")
				}
			}
		}

		// Lake Michigan
		if lon > -88 && lon < -85 {
			for dlat := -2.0; dlat <= 2.0; dlat += 0.5 {
				x, y := latLonToDisplay(lat+dlat, lon+1.5)
				if inBounds(x, y) {
					display[y][x] = waterStyle.Render("≈")
				}
			}
		}
	}

	// Add city marker for the center (on top of everything)
	if centerY >= 0 && centerY < len(display) && centerX >= 0 && centerX < len(display[0]) {
		display[centerY][centerX] = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true).
			Render("★")
	}
}

// DrawDistanceMarkers draws simple distance marker rings on the radar display
func DrawDistanceMarkers(display [][]string, centerX, centerY int) {
	markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	// 50 mile ring
	radius := 12
	for angle := 0.0; angle < 360.0; angle += 10 {
		x := int(float64(centerX) + float64(radius)*math.Cos(angle*math.Pi/180))
		y := int(float64(centerY) + float64(radius)*math.Sin(angle*math.Pi/180))

		if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
			if display[y][x] == " " {
				display[y][x] = markerStyle.Render("·")
			}
		}
	}

	// 100 mile ring
	radius = 22
	for angle := 0.0; angle < 360.0; angle += 15 {
		x := int(float64(centerX) + float64(radius)*math.Cos(angle*math.Pi/180))
		y := int(float64(centerY) + float64(radius)*math.Sin(angle*math.Pi/180)*0.5)

		if y >= 0 && y < len(display) && x >= 0 && x < len(display[0]) {
			if display[y][x] == " " {
				display[y][x] = markerStyle.Render("·")
			}
		}
	}
}