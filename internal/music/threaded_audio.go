package music

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

// ThreadedAudioPlayer runs audio in a completely separate thread
type ThreadedAudioPlayer struct {
	initialized  bool
	playing      bool
	volume       float64
	currentTrack int
	tracks       [][]Track
	mutex        sync.RWMutex
	stopChan     chan bool
	startChan    chan int // Send track number to start playing
	skipChan     chan bool
	volumeChan   chan float64
}

// NewThreadedAudioPlayer creates a new threaded audio player
func NewThreadedAudioPlayer() *ThreadedAudioPlayer {
	player := &ThreadedAudioPlayer{
		initialized: false,
		playing:     false,
		volume:      0.3,
		stopChan:    make(chan bool, 1),
		startChan:   make(chan int, 1),
		skipChan:    make(chan bool, 1),
		volumeChan:  make(chan float64, 10),
	}

	// Start audio thread immediately (non-blocking)
	go player.audioThread()

	return player
}

// audioThread runs in a separate goroutine and handles all audio
func (p *ThreadedAudioPlayer) audioThread() {
	// Initialize audio system in this thread
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	if err != nil {
		log.Printf("Audio initialization failed: %v", err)
		return
	}

	p.mutex.Lock()
	p.initialized = true
	p.mutex.Unlock()

	log.Println("Audio system initialized in background thread")

	// Audio event loop
	for {
		select {
		case <-p.stopChan:
			speaker.Clear()
			p.mutex.Lock()
			p.playing = false
			p.mutex.Unlock()

		case trackNum := <-p.startChan:
			p.mutex.Lock()
			p.currentTrack = trackNum
			p.playing = true
			tracks := p.tracks
			volume := p.volume
			p.mutex.Unlock()

			if len(tracks) > trackNum && len(tracks[trackNum]) > 0 {
				p.playTrackAudio(tracks[trackNum], volume, sampleRate)
			}

		case <-p.skipChan:
			speaker.Clear()
			p.mutex.Lock()
			if len(p.tracks) > 0 {
				p.currentTrack = (p.currentTrack + 1) % len(p.tracks)
				trackNum := p.currentTrack
				tracks := p.tracks
				volume := p.volume
				p.mutex.Unlock()

				if len(tracks) > trackNum && len(tracks[trackNum]) > 0 {
					p.playTrackAudio(tracks[trackNum], volume, sampleRate)
				}
			} else {
				p.mutex.Unlock()
			}

		case newVolume := <-p.volumeChan:
			p.mutex.Lock()
			p.volume = newVolume
			if p.volume < 0 {
				p.volume = 0
			} else if p.volume > 1 {
				p.volume = 1
			}
			p.mutex.Unlock()
		}
	}
}

// playTrackAudio plays a specific track's audio with full musical arrangement
func (p *ThreadedAudioPlayer) playTrackAudio(tracks []Track, volume float64, sampleRate beep.SampleRate) {
	var streamers []beep.Streamer

	// Create full musical arrangement for each instrument track
	for trackIdx, track := range tracks {
		if len(track.Notes) == 0 {
			continue
		}

		// Create sequenced audio for this instrument track
		trackStreamer := p.createInstrumentStreamer(track, trackIdx, sampleRate)
		if trackStreamer != nil {
			// Apply track-specific volume
			trackVol := track.Volume * volume * 0.3
			if trackVol > 0 {
				volumeCtrl := &VolumeControl{
					Streamer: trackStreamer,
					Volume:   trackVol,
				}
				streamers = append(streamers, volumeCtrl)
			}
		}
	}

	if len(streamers) > 0 {
		mixed := beep.Mix(streamers...)
		speaker.Play(mixed)
	}
}

// createInstrumentStreamer creates a full musical sequence for an instrument
func (p *ThreadedAudioPlayer) createInstrumentStreamer(track Track, trackIdx int, sampleRate beep.SampleRate) beep.Streamer {
	var noteStreamers []beep.Streamer

	// Create different instrument sounds based on track index
	for _, note := range track.Notes {
		freq := midiNoteToFrequency(note.Pitch)
		duration := note.Duration
		if duration > time.Second*8 {
			duration = time.Second * 8 // Allow longer notes for atmosphere
		}

		var noteStreamer beep.Streamer

		switch trackIdx {
		case 0: // Piano - rich harmonic tone
			noteStreamer = p.createPianoTone(freq, duration, sampleRate)
		case 1: // Bass - deep fundamental
			noteStreamer = p.createBassTone(freq, duration, sampleRate)
		case 2: // Saxophone - warm melodic tone
			noteStreamer = p.createSaxTone(freq, duration, sampleRate)
		case 3: // Pad - atmospheric background
			noteStreamer = p.createPadTone(freq, duration, sampleRate)
		default: // String-like tone
			noteStreamer = p.createStringTone(freq, duration, sampleRate)
		}

		// Add note timing (silence before note starts)
		if note.Start > 0 {
			silence := beep.Silence(sampleRate.N(note.Start))
			combined := beep.Seq(silence, noteStreamer)
			noteStreamers = append(noteStreamers, combined)
		} else {
			noteStreamers = append(noteStreamers, noteStreamer)
		}
	}

	if len(noteStreamers) > 0 {
		// Mix all notes for this instrument
		return beep.Mix(noteStreamers...)
	}

	return nil
}

// Simple sine wave generator
type SineWaveThreaded struct {
	freq       float64
	sampleRate beep.SampleRate
	pos        float64
}

func NewSineWaveThreaded(sampleRate beep.SampleRate, freq float64) *SineWaveThreaded {
	return &SineWaveThreaded{
		freq:       freq,
		sampleRate: sampleRate,
		pos:        0,
	}
}

func (s *SineWaveThreaded) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		sample := math.Sin(s.pos * 2 * math.Pi * s.freq / float64(s.sampleRate))
		samples[i][0] = sample
		samples[i][1] = sample
		s.pos++
	}
	return len(samples), true
}

func (s *SineWaveThreaded) Err() error {
	return nil
}

// Simple volume control
type VolumeControl struct {
	Streamer beep.Streamer
	Volume   float64
}

func (v *VolumeControl) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = v.Streamer.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= v.Volume
		samples[i][1] *= v.Volume
	}
	return n, ok
}

func (v *VolumeControl) Err() error {
	return v.Streamer.Err()
}

// Public interface methods
func (p *ThreadedAudioPlayer) SetTracks(tracks [][]Track) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.tracks = tracks
}

func (p *ThreadedAudioPlayer) Play() error {
	p.mutex.RLock()
	initialized := p.initialized
	currentTrack := p.currentTrack
	p.mutex.RUnlock()

	if !initialized {
		return fmt.Errorf("audio not initialized")
	}

	select {
	case p.startChan <- currentTrack:
	default:
	}
	return nil
}

func (p *ThreadedAudioPlayer) Stop() {
	select {
	case p.stopChan <- true:
	default:
	}
}

func (p *ThreadedAudioPlayer) Skip() error {
	select {
	case p.skipChan <- true:
	default:
	}
	return nil
}

func (p *ThreadedAudioPlayer) SetVolume(volume float64) {
	select {
	case p.volumeChan <- volume:
	default:
	}
}

func (p *ThreadedAudioPlayer) GetVolume() float64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.volume
}

func (p *ThreadedAudioPlayer) IsInitialized() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.initialized
}

func (p *ThreadedAudioPlayer) IsPlaying() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.playing
}

func (p *ThreadedAudioPlayer) Close() error {
	p.Stop()
	return nil
}

// midiNoteToFrequency converts a MIDI note number to frequency in Hz
func midiNoteToFrequency(midiNote int) float64 {
	// A4 (MIDI note 69) = 440 Hz
	// Frequency = 440 * 2^((midiNote - 69) / 12)
	return 440.0 * math.Pow(2.0, float64(midiNote-69)/12.0)
}

// Advanced instrument tone generators

// createPianoTone creates a rich piano-like tone with harmonics
func (p *ThreadedAudioPlayer) createPianoTone(freq float64, duration time.Duration, sampleRate beep.SampleRate) beep.Streamer {
	// PURE fundamental frequency only - NO harmonics to prevent sharp high notes
	fundamental := NewSineWaveThreaded(sampleRate, freq) // Pure sine wave only
	envelope := NewADSREnvelope(fundamental, duration, sampleRate, 0.05, 0.2, 0.8, 0.3)
	return beep.Take(sampleRate.N(duration), envelope)
}

// createBassTone creates a deep bass tone
func (p *ThreadedAudioPlayer) createBassTone(freq float64, duration time.Duration, sampleRate beep.SampleRate) beep.Streamer {
	// PURE fundamental frequency only - NO harmonics to prevent sharp high notes
	bass := NewSineWaveThreaded(sampleRate, freq) // Pure sine wave only
	envelope := NewADSREnvelope(bass, duration, sampleRate, 0.05, 0.2, 0.8, 0.3)
	return beep.Take(sampleRate.N(duration), envelope)
}

// createSaxTone creates a warm saxophone-like tone
func (p *ThreadedAudioPlayer) createSaxTone(freq float64, duration time.Duration, sampleRate beep.SampleRate) beep.Streamer {
	// PURE fundamental frequency only - NO harmonics to prevent sharp high notes
	sax := NewSineWaveThreaded(sampleRate, freq) // Pure sine wave only
	envelope := NewADSREnvelope(sax, duration, sampleRate, 0.15, 0.2, 0.7, 0.4)
	return beep.Take(sampleRate.N(duration), envelope)
}

// createPadTone creates an atmospheric pad sound
func (p *ThreadedAudioPlayer) createPadTone(freq float64, duration time.Duration, sampleRate beep.SampleRate) beep.Streamer {
	// PURE fundamental frequency only - NO harmonics to prevent sharp high notes
	pad := NewSineWaveThreaded(sampleRate, freq) // Pure sine wave only
	envelope := NewADSREnvelope(pad, duration, sampleRate, 1.0, 0.8, 0.9, 1.2)
	return beep.Take(sampleRate.N(duration), envelope)
}

// createStringTone creates a string-like tone
func (p *ThreadedAudioPlayer) createStringTone(freq float64, duration time.Duration, sampleRate beep.SampleRate) beep.Streamer {
	// PURE fundamental frequency only - NO harmonics to prevent sharp high notes
	strings := NewSineWaveThreaded(sampleRate, freq) // Pure sine wave only
	envelope := NewADSREnvelope(strings, duration, sampleRate, 0.2, 0.3, 0.8, 0.5)
	return beep.Take(sampleRate.N(duration), envelope)
}

// HarmonicTone generates rich harmonic content
type HarmonicTone struct {
	freq        float64
	harmonics   []float64 // Amplitude of each harmonic (1x, 2x, 3x, etc.)
	sampleRate  beep.SampleRate
	pos         float64
}

func NewHarmonicTone(sampleRate beep.SampleRate, freq float64, harmonics []float64) *HarmonicTone {
	return &HarmonicTone{
		freq:       freq,
		harmonics:  harmonics,
		sampleRate: sampleRate,
		pos:        0,
	}
}

func (h *HarmonicTone) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		sample := 0.0

		// Add each harmonic
		for harmIdx, amplitude := range h.harmonics {
			if amplitude > 0 {
				harmonicFreq := h.freq * float64(harmIdx+1)
				harmonicSample := amplitude * math.Sin(h.pos * 2 * math.Pi * harmonicFreq / float64(h.sampleRate))
				sample += harmonicSample
			}
		}

		// Normalize to prevent clipping
		sample *= 0.3

		samples[i][0] = sample
		samples[i][1] = sample
		h.pos++
	}
	return len(samples), true
}

func (h *HarmonicTone) Err() error {
	return nil
}

// ADSREnvelope applies Attack, Decay, Sustain, Release envelope
type ADSREnvelope struct {
	streamer    beep.Streamer
	sampleRate  beep.SampleRate
	attackTime  time.Duration
	decayTime   time.Duration
	sustainLevel float64
	releaseTime time.Duration
	totalSamples int64
	currentSample int64
	totalDuration time.Duration
}

func NewADSREnvelope(streamer beep.Streamer, duration time.Duration, sampleRate beep.SampleRate,
	attackSec, decaySec, sustainLevel, releaseSec float64) *ADSREnvelope {

	return &ADSREnvelope{
		streamer:      streamer,
		sampleRate:    sampleRate,
		attackTime:    time.Duration(attackSec * float64(time.Second)),
		decayTime:     time.Duration(decaySec * float64(time.Second)),
		sustainLevel:  sustainLevel,
		releaseTime:   time.Duration(releaseSec * float64(time.Second)),
		totalSamples:  int64(sampleRate.N(duration)),
		currentSample: 0,
		totalDuration: duration,
	}
}

func (e *ADSREnvelope) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = e.streamer.Stream(samples)

	for i := 0; i < n; i++ {
		envelope := e.calculateEnvelope()
		samples[i][0] *= envelope
		samples[i][1] *= envelope
		e.currentSample++
	}

	return n, ok && e.currentSample < e.totalSamples
}

func (e *ADSREnvelope) calculateEnvelope() float64 {
	attackSamples := int64(e.sampleRate.N(e.attackTime))
	decaySamples := int64(e.sampleRate.N(e.decayTime))
	releaseSamples := int64(e.sampleRate.N(e.releaseTime))
	releaseStart := e.totalSamples - releaseSamples

	if e.currentSample < attackSamples {
		// Attack phase: 0 to 1
		return float64(e.currentSample) / float64(attackSamples)
	} else if e.currentSample < attackSamples + decaySamples {
		// Decay phase: 1 to sustain
		decayProgress := float64(e.currentSample - attackSamples) / float64(decaySamples)
		return 1.0 - (1.0 - e.sustainLevel) * decayProgress
	} else if e.currentSample < releaseStart {
		// Sustain phase: constant sustain level
		return e.sustainLevel
	} else {
		// Release phase: sustain to 0
		releaseProgress := float64(e.currentSample - releaseStart) / float64(releaseSamples)
		return e.sustainLevel * (1.0 - releaseProgress)
	}
}

func (e *ADSREnvelope) Err() error {
	return e.streamer.Err()
}