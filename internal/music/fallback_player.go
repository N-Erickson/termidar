package music

import (
	"fmt"
	"sync"
	"time"
)

// PlayerState represents the current state of the music player
type PlayerState int

const (
	Stopped PlayerState = iota
	Playing
	Paused
)

// FallbackPlayer provides music player functionality without MIDI dependencies
type FallbackPlayer struct {
	generator *WeatherChannelGenerator
	state     PlayerState
	volume    float64
	position  time.Duration
	startTime time.Time
	mutex     sync.RWMutex
	stopChan  chan bool
	tracks    []Track
}

// NewFallbackPlayer creates a new fallback music player
func NewFallbackPlayer() (*FallbackPlayer, error) {
	generator := NewWeatherChannelGenerator()

	return &FallbackPlayer{
		generator: generator,
		state:     Stopped,
		volume:    0.8,
		position:  0,
		stopChan:  make(chan bool, 1),
	}, nil
}

// Close closes the fallback player (no-op for fallback)
func (p *FallbackPlayer) Close() error {
	p.Stop()
	return nil
}

// GenerateNewTrack generates a new random track
func (p *FallbackPlayer) GenerateNewTrack() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Randomize some parameters
	keys := []int{0, 2, 4, 5, 7, 9} // C, D, E, F, G, A
	tempos := []int{65, 70, 72, 75, 78}
	lengths := []time.Duration{
		time.Minute * 2,
		time.Minute*2 + time.Second*30,
		time.Minute * 3,
	}

	p.generator.SetKey(keys[time.Now().Second()%len(keys)])
	p.generator.SetTempo(tempos[time.Now().Second()%len(tempos)])
	p.generator.SetLength(lengths[time.Now().Second()%len(lengths)])

	err := p.generator.GenerateTrack()
	if err != nil {
		return fmt.Errorf("failed to generate track: %v", err)
	}

	p.tracks = p.generator.GetTracks()
	p.position = 0

	return nil
}

// Play starts or resumes playback (simulated)
func (p *FallbackPlayer) Play() error {
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

	go p.simulatePlayback()

	return nil
}

// Pause pauses playback
func (p *FallbackPlayer) Pause() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.state == Playing {
		p.state = Paused
		p.position = time.Since(p.startTime)
		select {
		case p.stopChan <- true:
		default:
		}
	}
}

// Stop stops playback and resets position
func (p *FallbackPlayer) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.state != Stopped {
		p.state = Stopped
		p.position = 0
		select {
		case p.stopChan <- true:
		default:
		}
	}
}

// Skip skips to the next randomly generated track
func (p *FallbackPlayer) Skip() error {
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
func (p *FallbackPlayer) SetVolume(volume float64) {
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
func (p *FallbackPlayer) GetVolume() float64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.volume
}

// GetState returns the current playback state
func (p *FallbackPlayer) GetState() PlayerState {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.state
}

// GetPosition returns the current playback position
func (p *FallbackPlayer) GetPosition() time.Duration {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.state == Playing {
		return time.Since(p.startTime)
	}
	return p.position
}

// GetTrackInfo returns information about the current track
func (p *FallbackPlayer) GetTrackInfo() string {
	if p.generator == nil {
		return "No track loaded"
	}
	return p.generator.GetInfo()
}

// GetStateString returns a human-readable state string
func (p *FallbackPlayer) GetStateString() string {
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

// simulatePlayback simulates audio playback timing
func (p *FallbackPlayer) simulatePlayback() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

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
			tracks := p.tracks
			p.mutex.RUnlock()

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
						p.Skip()
					}()
				}
			}
		}
	}
}