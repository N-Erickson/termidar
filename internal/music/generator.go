package music

import (
	"fmt"
	"math/rand"
	"time"
)

// Note represents a musical note
type Note struct {
	Pitch    int       // MIDI note number (0-127)
	Velocity int       // Note velocity (0-127)
	Duration time.Duration // Note duration
	Start    time.Duration // Start time relative to beginning
}

// Chord represents a musical chord
type Chord struct {
	Root     int
	Quality  ChordQuality
	Notes    []int
}

type ChordQuality int

const (
	Major ChordQuality = iota
	Minor
	Dominant7
	Major7
	Minor7
	Diminished
	Suspended2
	Suspended4
)

// Scale represents a musical scale
type Scale struct {
	Root  int
	Type  ScaleType
	Notes []int
}

type ScaleType int

const (
	MajorScale ScaleType = iota
	MinorScale
	Dorian
	Mixolydian
	Pentatonic
)

// Track represents a musical track
type Track struct {
	Name       string
	Channel    int
	Instrument int
	Notes      []Note
	Volume     float64
}

// WeatherChannelGenerator generates smooth jazz/ambient music in the style of Weather Channel
type WeatherChannelGenerator struct {
	tempo      int
	key        int  // Root key (0-11, C=0)
	timeSignature [2]int // [numerator, denominator]
	tracks     []Track
	length     time.Duration
	style      string  // Musical style: "classic", "upbeat", "ambient", "latin", "night"
	intensity  float64 // Overall intensity/complexity (0.0 to 1.0)
	rand       *rand.Rand
}

// NewWeatherChannelGenerator creates a new music generator
func NewWeatherChannelGenerator() *WeatherChannelGenerator {
	return &WeatherChannelGenerator{
		tempo:         72, // Slow, relaxed tempo
		key:           0,  // C major
		timeSignature: [2]int{4, 4},
		length:        time.Minute * 2, // 2 minute tracks
		style:         "classic",
		intensity:     0.6,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SetTempo sets the tempo in BPM
func (g *WeatherChannelGenerator) SetTempo(bpm int) {
	g.tempo = bpm
}

// SetKey sets the root key (0-11, C=0)
func (g *WeatherChannelGenerator) SetKey(key int) {
	g.key = key % 12
}

// SetLength sets the track length
func (g *WeatherChannelGenerator) SetLength(length time.Duration) {
	g.length = length
}

// SetStyle sets the musical style
func (g *WeatherChannelGenerator) SetStyle(style string) {
	g.style = style
}

// SetIntensity sets the musical intensity/complexity
func (g *WeatherChannelGenerator) SetIntensity(intensity float64) {
	g.intensity = intensity
}

// GenerateTrack generates a complete smooth jazz track
func (g *WeatherChannelGenerator) GenerateTrack() error {
	g.tracks = []Track{}

	// Generate chord progression
	chords := g.generateChordProgression()

	// Create tracks
	g.addPianoTrack(chords)
	g.addBassTrack(chords)
	g.addSaxTrack(chords)
	g.addDrumTrack()
	g.addPadTrack(chords)

	return nil
}

// generateChordProgression creates video game style progressions
func (g *WeatherChannelGenerator) generateChordProgression() []Chord {
	switch g.style {
	case "upbeat":
		return g.generateUpbeatProgression()
	case "smooth":
		return g.generateSmoothProgression()
	case "gentle":
		return g.generateGentleProgression()
	case "driving":
		return g.generateDrivingProgression()
	case "ambient":
		return g.generateAmbientProgression()
	default:
		return g.generateUpbeatProgression()
	}
}

// Video game style chord progressions

// generateUpbeatProgression - Classic upbeat theme
func (g *WeatherChannelGenerator) generateUpbeatProgression() []Chord {
	// Classic video game progressions - upbeat and memorable
	progressions := [][]struct{root int; quality ChordQuality}{
		{{0, Major}, {5, Minor}, {3, Major}, {7, Major}},              // I-vi-IV-V (classic pop/video game)
		{{0, Major}, {7, Major}, {5, Minor}, {3, Major}},              // I-V-vi-IV (uplifting)
		{{0, Major}, {2, Minor}, {5, Minor}, {7, Major}},              // I-ii-vi-V (smooth)
	}
	return g.chordsFromPattern(progressions[g.rand.Intn(len(progressions))])
}

// generateSmoothProgression - Smooth flowing theme
func (g *WeatherChannelGenerator) generateSmoothProgression() []Chord {
	// Relaxed, flowing progressions
	progressions := [][]struct{root int; quality ChordQuality}{
		{{0, Major7}, {5, Minor7}, {2, Minor7}, {7, Major7}},          // IM7-vi7-ii7-VM7 (smooth jazz)
		{{0, Major}, {3, Major}, {5, Minor}, {7, Major}},              // I-IV-vi-V (gentle)
		{{5, Minor}, {3, Major}, {0, Major}, {7, Major}},              // vi-IV-I-V (emotional)
	}
	return g.chordsFromPattern(progressions[g.rand.Intn(len(progressions))])
}

// generateGentleProgression - Gentle theme
func (g *WeatherChannelGenerator) generateGentleProgression() []Chord {
	// Gentle, flowing progressions
	progressions := [][]struct{root int; quality ChordQuality}{
		{{0, Major}, {5, Minor}, {7, Major}, {0, Major}},              // I-vi-V-I (peaceful resolution)
		{{0, Major}, {3, Major}, {2, Minor}, {7, Major}},              // I-IV-ii-V (classic)
		{{5, Minor}, {7, Major}, {0, Major}, {3, Major}},              // vi-V-I-IV (gentle flow)
	}
	return g.chordsFromPattern(progressions[g.rand.Intn(len(progressions))])
}

// generateDrivingProgression - Driving theme
func (g *WeatherChannelGenerator) generateDrivingProgression() []Chord {
	// Rhythmic, driving progressions
	progressions := [][]struct{root int; quality ChordQuality}{
		{{0, Major}, {10, Major}, {7, Major}, {0, Major}},             // I-bVII-V-I (rock-influenced)
		{{0, Major}, {2, Minor}, {7, Major}, {2, Minor}},              // I-ii-V-ii (driving)
		{{7, Major}, {10, Major}, {0, Major}, {7, Major}},             // V-bVII-I-V (powerful)
	}
	return g.chordsFromPattern(progressions[g.rand.Intn(len(progressions))])
}

// generateAmbientProgression - Ambient theme
func (g *WeatherChannelGenerator) generateAmbientProgression() []Chord {
	// Contemplative, steady progressions
	progressions := [][]struct{root int; quality ChordQuality}{
		{{0, Minor}, {7, Major}, {3, Major}, {0, Minor}},              // i-V-III-i (contemplative minor)
		{{0, Major}, {5, Minor}, {3, Major}, {0, Major}},              // I-vi-IV-I (stable, thoughtful)
		{{2, Minor}, {7, Major}, {0, Major}, {5, Minor}},              // ii-V-I-vi (planning cycle)
	}
	return g.chordsFromPattern(progressions[g.rand.Intn(len(progressions))])
}

// chordsFromPattern converts a pattern to actual chords
func (g *WeatherChannelGenerator) chordsFromPattern(pattern []struct{root int; quality ChordQuality}) []Chord {
	chords := make([]Chord, len(pattern))
	for i, p := range pattern {
		actualRoot := (g.key + p.root) % 12
		chords[i] = g.createChord(actualRoot, p.quality)
	}
	return chords
}

// createChord creates a chord with the given root and quality
func (g *WeatherChannelGenerator) createChord(root int, quality ChordQuality) Chord {
	chord := Chord{
		Root:    root,
		Quality: quality,
	}

	baseNote := 36 + root // PLEASANT LOW C2 + root (COMFORTABLE BASS RANGE)

	switch quality {
	case Major:
		chord.Notes = []int{baseNote, baseNote + 4, baseNote + 7}
	case Minor:
		chord.Notes = []int{baseNote, baseNote + 3, baseNote + 7}
	case Dominant7:
		chord.Notes = []int{baseNote, baseNote + 4, baseNote + 7, baseNote + 10}
	case Major7:
		chord.Notes = []int{baseNote, baseNote + 4, baseNote + 7, baseNote + 11}
	case Minor7:
		chord.Notes = []int{baseNote, baseNote + 3, baseNote + 7, baseNote + 10}
	case Diminished:
		chord.Notes = []int{baseNote, baseNote + 3, baseNote + 6}
	case Suspended2:
		chord.Notes = []int{baseNote, baseNote + 2, baseNote + 7}
	case Suspended4:
		chord.Notes = []int{baseNote, baseNote + 5, baseNote + 7}
	}

	return chord
}

// addPianoTrack adds a smooth jazz piano track
func (g *WeatherChannelGenerator) addPianoTrack(chords []Chord) {
	track := Track{
		Name:       "Piano",
		Channel:    0,
		Instrument: 1, // Acoustic Grand Piano
		Notes:      []Note{},
		Volume:     g.getInstrumentVolume("piano"),
	}

	beatDuration := time.Minute / time.Duration(g.tempo)
	measureDuration := beatDuration * 4

	switch g.style {
	case "upbeat":
		g.addUpbeatPianoNotes(&track, chords, beatDuration, measureDuration)
	case "smooth":
		g.addSmoothPianoNotes(&track, chords, beatDuration, measureDuration)
	case "gentle":
		g.addGentlePianoNotes(&track, chords, beatDuration, measureDuration)
	case "driving":
		g.addDrivingPianoNotes(&track, chords, beatDuration, measureDuration)
	case "ambient":
		g.addAmbientPianoNotes(&track, chords, beatDuration, measureDuration)
	default:
		g.addUpbeatPianoNotes(&track, chords, beatDuration, measureDuration)
	}

	g.tracks = append(g.tracks, track)
}

// addBassTrack adds a walking bass line
func (g *WeatherChannelGenerator) addBassTrack(chords []Chord) {
	track := Track{
		Name:       "Bass",
		Channel:    1,
		Instrument: 33, // Acoustic Bass
		Notes:      []Note{},
		Volume:     g.getInstrumentVolume("bass"),
	}

	beatDuration := time.Minute / time.Duration(g.tempo)
	measureDuration := beatDuration * 4

	currentTime := time.Duration(0)

	for currentTime < g.length {
		chord := chords[int(currentTime/measureDuration)%len(chords)]

		// REAL upright bass walking pattern (authentic jazz bass range)
		bassNotes := []int{
			chord.Root + 36,                    // Root (MIDI 36-48: C2-C3 - REAL upright bass range)
			chord.Root + 36 + 7,               // Fifth
			chord.Root + 36 + 3,               // Third
			chord.Root + 36 + 10,              // Seventh
		}

		for i, note := range bassNotes {
			if i == 0 || g.rand.Float32() < 0.8 {
				track.Notes = append(track.Notes, Note{
					Pitch:    note,
					Velocity: 70 + g.rand.Intn(25), // Much more dynamic bass
					Duration: beatDuration,
					Start:    currentTime + time.Duration(i)*beatDuration,
				})
			}
		}

		currentTime += measureDuration
	}

	g.tracks = append(g.tracks, track)
}

// addSaxTrack adds a smooth saxophone melody
func (g *WeatherChannelGenerator) addSaxTrack(chords []Chord) {
	track := Track{
		Name:       "Saxophone",
		Channel:    2,
		Instrument: 67, // Tenor Sax
		Notes:      []Note{},
		Volume:     g.getInstrumentVolume("sax"),
	}

	beatDuration := time.Minute / time.Duration(g.tempo)
	scale := g.getScale()

	currentTime := time.Duration(0)

	for currentTime < g.length {
		// Soulful saxophone phrases in authentic tenor sax range
		if g.rand.Float32() < 0.4 {
			phraseLength := beatDuration * time.Duration(2+g.rand.Intn(6))

			note := scale.Notes[g.rand.Intn(len(scale.Notes))] + 48 // PLEASANT sax range: MIDI 60-72 (C4-C5) - PLEASANT MELODY RANGE
			track.Notes = append(track.Notes, Note{
				Pitch:    note,
				Velocity: 60 + g.rand.Intn(35), // Much more dynamic range
				Duration: phraseLength,
				Start:    currentTime,
			})
		}

		currentTime += beatDuration * 2
	}

	g.tracks = append(g.tracks, track)
}

// addDrumTrack adds subtle ambient percussion
func (g *WeatherChannelGenerator) addDrumTrack() {
	track := Track{
		Name:       "Drums",
		Channel:    9, // Standard MIDI drum channel
		Instrument: 0, // Drum kit
		Notes:      []Note{},
		Volume:     g.getInstrumentVolume("drums"), // Very subtle
	}

	beatDuration := time.Minute / time.Duration(g.tempo)
	currentTime := time.Duration(0)

	for currentTime < g.length {
		// Soft kick on beats 1 and 3
		if g.rand.Float32() < 0.6 {
			track.Notes = append(track.Notes, Note{
				Pitch:    36, // Bass drum
				Velocity: 30 + g.rand.Intn(15),
				Duration: beatDuration / 4,
				Start:    currentTime,
			})
		}

		// Soft hi-hat on off-beats
		if g.rand.Float32() < 0.4 {
			track.Notes = append(track.Notes, Note{
				Pitch:    42, // Closed hi-hat
				Velocity: 25 + g.rand.Intn(10),
				Duration: beatDuration / 8,
				Start:    currentTime + beatDuration/2,
			})
		}

		currentTime += beatDuration * 2
	}

	g.tracks = append(g.tracks, track)
}

// addPadTrack adds ambient string/pad sounds
func (g *WeatherChannelGenerator) addPadTrack(chords []Chord) {
	track := Track{
		Name:       "Pad",
		Channel:    3,
		Instrument: 89, // Warm Pad
		Notes:      []Note{},
		Volume:     g.getInstrumentVolume("pad"),
	}

	beatDuration := time.Minute / time.Duration(g.tempo)
	measureDuration := beatDuration * 4

	currentTime := time.Duration(0)

	for currentTime < g.length {
		chord := chords[int(currentTime/measureDuration)%len(chords)]

		// Long, sustained chord tones
		for _, note := range chord.Notes {
			track.Notes = append(track.Notes, Note{
				Pitch:    note + 24, // PLEASANT pad range: MIDI 60-72 (C4-C5) - PLEASANT BACKGROUND RANGE
				Velocity: 40 + g.rand.Intn(20),
				Duration: measureDuration,
				Start:    currentTime,
			})
		}

		currentTime += measureDuration
	}

	g.tracks = append(g.tracks, track)
}

// getScale returns the current scale
func (g *WeatherChannelGenerator) getScale() Scale {
	scale := Scale{
		Root: g.key,
		Type: MajorScale,
	}

	// Major scale intervals
	intervals := []int{0, 2, 4, 5, 7, 9, 11}
	scale.Notes = make([]int, len(intervals))

	for i, interval := range intervals {
		scale.Notes[i] = (g.key + interval) % 12
	}

	return scale
}

// getInstrumentVolume returns volume based on style and intensity
func (g *WeatherChannelGenerator) getInstrumentVolume(instrument string) float64 {
	baseVolumes := map[string]float64{
		"piano": 0.7,
		"bass":  0.6,
		"sax":   0.5,
		"pad":   0.4,
		"drums": 0.3,
	}

	base := baseVolumes[instrument]
	return base * g.intensity
}

// Video game style piano patterns

// addUpbeatPianoNotes creates the classic upbeat theme
func (g *WeatherChannelGenerator) addUpbeatPianoNotes(track *Track, chords []Chord, beatDuration, measureDuration time.Duration) {
	currentTime := time.Duration(0)
	for currentTime < g.length {
		chord := chords[int(currentTime/measureDuration)%len(chords)]

		// PLEASANT piano left hand chord voicings (comfortable bass range)
		for i, note := range chord.Notes {
			if i < 3 {
				track.Notes = append(track.Notes, Note{
					Pitch:    note + 12, // MIDI 48-60 (C3-C4) - PLEASANT BASS RANGE
					Velocity: 65 + g.rand.Intn(15),
					Duration: beatDuration * 2,
					Start:    currentTime + time.Duration(i)*beatDuration/8,
				})
			}
		}

		// PLEASANT piano right hand melody (comfortable melody range)
		scale := g.getScale()
		melodyPattern := []int{0, 2, 4, 2, 0, 4, 6, 4}
		for i, scaleStep := range melodyPattern {
			if i < 6 {
				melodyNote := scale.Notes[scaleStep%len(scale.Notes)] + 48 // MIDI 60-72 (C4-C5) - PLEASANT MELODY RANGE
				track.Notes = append(track.Notes, Note{
					Pitch:    melodyNote,
					Velocity: 70 + g.rand.Intn(15),
					Duration: beatDuration,
					Start:    currentTime + time.Duration(i)*beatDuration,
				})
			}
		}

		// PLEASANT bass line (comfortable bass range)
		bassNote := chord.Notes[0] + 0 // MIDI 36-48 (C2-C3) - PLEASANT BASS RANGE
		track.Notes = append(track.Notes, Note{
			Pitch:    bassNote,
			Velocity: 75,
			Duration: measureDuration,
			Start:    currentTime,
		})

		currentTime += measureDuration
	}
}

// addSmoothPianoNotes creates smooth flowing theme
func (g *WeatherChannelGenerator) addSmoothPianoNotes(track *Track, chords []Chord, beatDuration, measureDuration time.Duration) {
	currentTime := time.Duration(0)
	for currentTime < g.length {
		chord := chords[int(currentTime/measureDuration)%len(chords)]

		// PLEASANT jazz piano left hand chord voicings (smooth jazz style)
		for i, note := range chord.Notes {
			if i < 3 {
				track.Notes = append(track.Notes, Note{
					Pitch:    note + 12, // MIDI 48-60 (C3-C4) - PLEASANT BASS RANGE
					Velocity: 55 + g.rand.Intn(15),
					Duration: beatDuration * 3, // Longer sustain for smooth style
					Start:    currentTime + time.Duration(i)*beatDuration/8,
				})
			}
		}

		// PLEASANT jazz piano right hand melody (smooth and flowing)
		scale := g.getScale()
		melodyPattern := []int{0, 4, 2, 6, 4, 0}
		for i, scaleStep := range melodyPattern {
			if i < 4 {
				melodyNote := scale.Notes[scaleStep%len(scale.Notes)] + 48 // MIDI 60-72 (C4-C5) - PLEASANT MELODY RANGE
				track.Notes = append(track.Notes, Note{
					Pitch:    melodyNote,
					Velocity: 50 + g.rand.Intn(20),
					Duration: beatDuration * 2, // Smooth flowing duration
					Start:    currentTime + time.Duration(i)*beatDuration,
				})
			}
		}

		// ULTRA DEEP bass line (like upright bass in smooth jazz)
		bassNote := chord.Notes[0] + 12 // MIDI 12-24 (C0-C1) - SUB-BASS RANGE
		track.Notes = append(track.Notes, Note{
			Pitch:    bassNote,
			Velocity: 60,
			Duration: measureDuration,
			Start:    currentTime,
		})

		currentTime += measureDuration
	}
}

// addGentlePianoNotes creates gentle theme
func (g *WeatherChannelGenerator) addGentlePianoNotes(track *Track, chords []Chord, beatDuration, measureDuration time.Duration) {
	currentTime := time.Duration(0)
	for currentTime < g.length {
		chord := chords[int(currentTime/measureDuration)%len(chords)]

		// PLEASANT jazz piano left hand chord voicings (gentle jazz style)
		for i, note := range chord.Notes {
			if i < 3 {
				track.Notes = append(track.Notes, Note{
					Pitch:    note + 12, // MIDI 48-60 (C3-C4) - PLEASANT BASS RANGE
					Velocity: 45 + g.rand.Intn(15),
					Duration: measureDuration, // Gentle sustained chords
					Start:    currentTime + time.Duration(i)*beatDuration/6,
				})
			}
		}

		// PLEASANT jazz piano right hand melody (simple and peaceful)
		scale := g.getScale()
		melodyPattern := []int{0, 2, 4, 0, 6, 4, 2}
		for i, scaleStep := range melodyPattern {
			if i < 5 {
				melodyNote := scale.Notes[scaleStep%len(scale.Notes)] + 48 // MIDI 60-72 (C4-C5) - PLEASANT MELODY RANGE
				track.Notes = append(track.Notes, Note{
					Pitch:    melodyNote,
					Velocity: 40 + g.rand.Intn(15),
					Duration: beatDuration,
					Start:    currentTime + time.Duration(i)*beatDuration*3/4,
				})
			}
		}

		// ULTRA DEEP bass line (like upright bass in gentle jazz)
		bassNote := chord.Notes[0] + 12 // MIDI 12-24 (C0-C1) - SUB-BASS RANGE
		track.Notes = append(track.Notes, Note{
			Pitch:    bassNote,
			Velocity: 50,
			Duration: measureDuration,
			Start:    currentTime,
		})

		currentTime += measureDuration
	}
}

// addDrivingPianoNotes creates driving theme
func (g *WeatherChannelGenerator) addDrivingPianoNotes(track *Track, chords []Chord, beatDuration, measureDuration time.Duration) {
	currentTime := time.Duration(0)
	for currentTime < g.length {
		chord := chords[int(currentTime/measureDuration)%len(chords)]

		// PLEASANT jazz piano left hand - driving rhythmic chord pattern
		rhythmBeats := []float64{0, 0.5, 1, 2, 2.5, 3}
		for _, beat := range rhythmBeats {
			noteIdx := g.rand.Intn(len(chord.Notes))
			if noteIdx < 3 {
				track.Notes = append(track.Notes, Note{
					Pitch:    chord.Notes[noteIdx] + 24, // MIDI 24-36 (C1-C2) - PLEASANT BASS RANGE
					Velocity: 70 + g.rand.Intn(20),
					Duration: beatDuration / 2,
					Start:    currentTime + time.Duration(beat*float64(beatDuration)),
				})
			}
		}

		// PLEASANT jazz piano right hand - energetic melody
		scale := g.getScale()
		melodyPattern := []int{0, 4, 6, 4, 2, 6, 0}
		for i, scaleStep := range melodyPattern {
			if i < 6 {
				melodyNote := scale.Notes[scaleStep%len(scale.Notes)] + 48 // MIDI 60-72 (C4-C5) - PLEASANT MELODY RANGE
				track.Notes = append(track.Notes, Note{
					Pitch:    melodyNote,
					Velocity: 75 + g.rand.Intn(15),
					Duration: beatDuration / 2,
					Start:    currentTime + time.Duration(i)*beatDuration/2,
				})
			}
		}

		// ULTRA DEEP bass line (like upright bass in driving jazz)
		bassNote := chord.Notes[0] + 12 // MIDI 12-24 (C0-C1) - SUB-BASS RANGE
		track.Notes = append(track.Notes, Note{
			Pitch:    bassNote,
			Velocity: 80,
			Duration: measureDuration,
			Start:    currentTime,
		})

		currentTime += measureDuration
	}
}

// addAmbientPianoNotes creates contemplative ambient theme
func (g *WeatherChannelGenerator) addAmbientPianoNotes(track *Track, chords []Chord, beatDuration, measureDuration time.Duration) {
	currentTime := time.Duration(0)
	for currentTime < g.length {
		chord := chords[int(currentTime/measureDuration)%len(chords)]

		// REAL jazz piano left hand - contemplative chord voicings
		for i, note := range chord.Notes {
			if i < 4 {
				track.Notes = append(track.Notes, Note{
					Pitch:    note + 48, // MIDI 48-60 (C3-C4) - REAL piano range
					Velocity: 35 + g.rand.Intn(20),
					Duration: measureDuration * 2, // Long sustain for ambient
					Start:    currentTime + time.Duration(i)*beatDuration/4,
				})
			}
		}

		// REAL jazz piano right hand - thoughtful melody
		scale := g.getScale()
		melodyPattern := []int{0, 2, 4, 6, 4, 2}
		for i, scaleStep := range melodyPattern {
			if i < 4 {
				melodyNote := scale.Notes[scaleStep%len(scale.Notes)] + 60 // MIDI 60-72 (C4-C5) - REAL melody range
				track.Notes = append(track.Notes, Note{
					Pitch:    melodyNote,
					Velocity: 30 + g.rand.Intn(20),
					Duration: beatDuration * 2,
					Start:    currentTime + time.Duration(i)*beatDuration*2,
				})
			}
		}

		// REAL bass line (like upright bass in ambient jazz)
		bassNote := chord.Notes[0] + 36 // MIDI 36-48 (C2-C3) - REAL bass range
		track.Notes = append(track.Notes, Note{
			Pitch:    bassNote,
			Velocity: 45,
			Duration: measureDuration * 2,
			Start:    currentTime,
		})

		currentTime += measureDuration
	}
}

// GetTracks returns all generated tracks
func (g *WeatherChannelGenerator) GetTracks() []Track {
	return g.tracks
}

// GetInfo returns information about the generated music
func (g *WeatherChannelGenerator) GetInfo() string {
	keyNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	return fmt.Sprintf(
		"Weather Channel Style Music\nKey: %s\nTempo: %d BPM\nLength: %.1f minutes\nTracks: %d",
		keyNames[g.key],
		g.tempo,
		g.length.Minutes(),
		len(g.tracks),
	)
}