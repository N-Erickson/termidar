package music

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

// AudioPlayer provides real audio playback for generated music
type AudioPlayer struct {
	generator     *WeatherChannelGenerator
	state         PlayerState
	volume        float64
	position      time.Duration
	startTime     time.Time
	mutex         sync.RWMutex
	stopChan      chan bool
	tracks        []Track
	mixer         *beep.Mixer
	sampleRate    beep.SampleRate
	initialized   bool
	activeStreams map[string]*beep.Ctrl
}

// NewAudioPlayer creates a new audio player with real sound output
func NewAudioPlayer() (*AudioPlayer, error) {
	generator := NewWeatherChannelGenerator()
	sampleRate := beep.SampleRate(44100)

	// Initialize speaker
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize speaker: %v", err)
	}

	mixer := &beep.Mixer{}
	speaker.Play(mixer)

	return &AudioPlayer{
		generator:     generator,
		state:         Stopped,
		volume:        0.3, // Start with lower volume
		position:      0,
		stopChan:      make(chan bool, 1),
		mixer:         mixer,
		sampleRate:    sampleRate,
		initialized:   true,
		activeStreams: make(map[string]*beep.Ctrl),
	}, nil
}

// Close closes the audio player and cleans up resources
func (p *AudioPlayer) Close() error {
	p.Stop()
	if p.initialized {
		speaker.Clear()
	}
	return nil
}

// GenerateNewTrack generates a new random track
func (p *AudioPlayer) GenerateNewTrack() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Stop current playback
	p.stopAllStreams()

	// Randomize parameters for variety
	keys := []int{0, 2, 4, 5, 7, 9} // C, D, E, F, G, A
	tempos := []int{65, 70, 72, 75, 78}
	lengths := []time.Duration{
		time.Minute * 2,
		time.Minute*2 + time.Second*30,
		time.Minute * 3,
	}

	now := time.Now()
	p.generator.SetKey(keys[now.Second()%len(keys)])
	p.generator.SetTempo(tempos[now.Nanosecond()%len(tempos)])
	p.generator.SetLength(lengths[now.Minute()%len(lengths)])

	err := p.generator.GenerateTrack()
	if err != nil {
		return fmt.Errorf("failed to generate track: %v", err)
	}

	p.tracks = p.generator.GetTracks()
	p.position = 0

	return nil
}

// Play starts or resumes playback with real audio
func (p *AudioPlayer) Play() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.state == Playing {
		return nil
	}

	if len(p.tracks) == 0 {
		err := p.GenerateNewTrack()
		if err != nil {
			return err
		}
	}

	p.state = Playing
	p.startTime = time.Now().Add(-p.position)

	go p.audioPlaybackLoop()

	return nil
}

// Pause pauses playback
func (p *AudioPlayer) Pause() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.state == Playing {
		p.state = Paused
		p.position = time.Since(p.startTime)
		p.stopAllStreams()
		select {
		case p.stopChan <- true:
		default:
		}
	}
}

// Stop stops playback and resets position
func (p *AudioPlayer) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.state != Stopped {
		p.state = Stopped
		p.position = 0
		p.stopAllStreams()
		select {
		case p.stopChan <- true:
		default:
		}
	}
}

// Skip skips to the next randomly generated track
func (p *AudioPlayer) Skip() error {
	wasPlaying := p.state == Playing

	p.Stop()

	err := p.GenerateNewTrack()
	if err != nil {
		return err
	}

	if wasPlaying {
		return p.Play()
	}

	return nil
}

// SetVolume sets the playback volume (0.0 to 1.0)
func (p *AudioPlayer) SetVolume(volume float64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if volume < 0.0 {
		volume = 0.0
	} else if volume > 1.0 {
		volume = 1.0
	}

	p.volume = volume
}

// GetVolume returns the current volume
func (p *AudioPlayer) GetVolume() float64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.volume
}

// GetState returns the current playback state
func (p *AudioPlayer) GetState() PlayerState {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.state
}

// GetPosition returns the current playback position
func (p *AudioPlayer) GetPosition() time.Duration {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.state == Playing {
		return time.Since(p.startTime)
	}
	return p.position
}

// GetTrackInfo returns information about the current track
func (p *AudioPlayer) GetTrackInfo() string {
	if p.generator == nil {
		return "No track loaded"
	}
	return p.generator.GetInfo()
}

// GetStateString returns a human-readable state string
func (p *AudioPlayer) GetStateString() string {
	state := p.GetState()
	switch state {
	case Playing:
		return "Playing"
	case Paused:
		return "Paused"
	case Stopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// stopAllStreams stops all currently playing audio streams
func (p *AudioPlayer) stopAllStreams() {
	for _, ctrl := range p.activeStreams {
		ctrl.Paused = true
	}
	p.activeStreams = make(map[string]*beep.Ctrl)
	p.mixer.Clear()
}

// midiNoteToFrequency converts MIDI note number to frequency in Hz
func (p *AudioPlayer) midiNoteToFrequency(midiNote int) float64 {
	// A4 (MIDI note 69) = 440 Hz
	// Frequency = 440 * 2^((midiNote - 69) / 12)
	return 440.0 * math.Pow(2.0, float64(midiNote-69)/12.0)
}

// SineWave generates a sine wave at a specific frequency
type SineWave struct {
	freq       float64
	sampleRate beep.SampleRate
	pos        float64
}

func NewSineWave(sampleRate beep.SampleRate, freq float64) *SineWave {
	return &SineWave{
		freq:       freq,
		sampleRate: sampleRate,
		pos:        0,
	}
}

func (s *SineWave) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		sample := math.Sin(s.pos * 2 * math.Pi * s.freq / float64(s.sampleRate))
		samples[i][0] = sample
		samples[i][1] = sample
		s.pos++
	}
	return len(samples), true
}

func (s *SineWave) Err() error {
	return nil
}

// createToneForNote creates a tone generator for a specific note
func (p *AudioPlayer) createToneForNote(note Note, trackVolume float64) (beep.Streamer, error) {
	freq := p.midiNoteToFrequency(note.Pitch)
	duration := note.Duration

	// Apply volume scaling
	amplitude := (float64(note.Velocity) / 127.0) * trackVolume * p.volume * 0.3 // Keep it gentle

	// Create a sine wave tone
	tone := NewSineWave(p.sampleRate, freq)

	// Apply volume
	volumeCtrl := &effects.Volume{
		Streamer: tone,
		Base:     2,
		Volume:   math.Log2(amplitude + 0.01), // Avoid log(0)
	}

	// Limit duration
	limited := beep.Take(p.sampleRate.N(duration), volumeCtrl)

	return limited, nil
}

// audioPlaybackLoop handles real audio playback
func (p *AudioPlayer) audioPlaybackLoop() {
	ticker := time.NewTicker(time.Millisecond * 50) // 20 FPS for smooth playback
	defer ticker.Stop()

	noteStates := make(map[string]time.Duration) // track when notes should end

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.mutex.RLock()
			if p.state != Playing {
				p.mutex.RUnlock()
				continue
			}

			currentPos := time.Since(p.startTime)
			volume := p.volume
			tracks := p.tracks
			p.mutex.RUnlock()

			// Start new notes
			for trackIdx, track := range tracks {
				trackVolume := track.Volume * volume
				for noteIdx, note := range track.Notes {
					noteKey := fmt.Sprintf("t%d_n%d", trackIdx, noteIdx)

					// Check if note should start now
					if note.Start <= currentPos && note.Start+time.Millisecond*100 > currentPos {
						if _, exists := noteStates[noteKey]; !exists {
							// Start this note
							noteStates[noteKey] = currentPos + note.Duration

							// Create tone for this note
							tone, err := p.createToneForNote(note, trackVolume)
							if err == nil {
								ctrl := &beep.Ctrl{Streamer: tone, Paused: false}
								p.activeStreams[noteKey] = ctrl
								p.mixer.Add(ctrl)
							}
						}
					}
				}
			}

			// Clean up finished notes
			for noteKey, endTime := range noteStates {
				if currentPos >= endTime {
					delete(noteStates, noteKey)
					if ctrl, exists := p.activeStreams[noteKey]; exists {
						ctrl.Paused = true
						delete(p.activeStreams, noteKey)
					}
				}
			}

			// Check if track is finished
			if len(tracks) > 0 {
				maxLength := time.Duration(0)
				for _, track := range tracks {
					for _, note := range track.Notes {
						end := note.Start + note.Duration
						if end > maxLength {
							maxLength = end
						}
					}
				}

				if currentPos >= maxLength {
					// Track finished, generate new one
					go func() {
						time.Sleep(time.Second) // Brief pause between tracks
						p.Skip()
					}()
				}
			}
		}
	}
}