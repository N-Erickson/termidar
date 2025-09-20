package music

import "time"

// Player interface defines the common methods for both audio and fallback players
type Player interface {
	Close() error
	GenerateNewTrack() error
	Play() error
	Pause()
	Stop()
	Skip() error
	SetVolume(volume float64)
	GetVolume() float64
	GetState() PlayerState
	GetPosition() time.Duration
	GetTrackInfo() string
	GetStateString() string
}

// AudioPlayerWrapper wraps FallbackPlayer to implement the Player interface for AudioPlayer
type AudioPlayerWrapper struct {
	Fallback *FallbackPlayer
}

func (w *AudioPlayerWrapper) Close() error {
	return w.Fallback.Close()
}

func (w *AudioPlayerWrapper) GenerateNewTrack() error {
	return w.Fallback.GenerateNewTrack()
}

func (w *AudioPlayerWrapper) Play() error {
	return w.Fallback.Play()
}

func (w *AudioPlayerWrapper) Pause() {
	w.Fallback.Pause()
}

func (w *AudioPlayerWrapper) Stop() {
	w.Fallback.Stop()
}

func (w *AudioPlayerWrapper) Skip() error {
	return w.Fallback.Skip()
}

func (w *AudioPlayerWrapper) SetVolume(volume float64) {
	w.Fallback.SetVolume(volume)
}

func (w *AudioPlayerWrapper) GetVolume() float64 {
	return w.Fallback.GetVolume()
}

func (w *AudioPlayerWrapper) GetState() PlayerState {
	return w.Fallback.GetState()
}

func (w *AudioPlayerWrapper) GetPosition() time.Duration {
	return w.Fallback.GetPosition()
}

func (w *AudioPlayerWrapper) GetTrackInfo() string {
	return w.Fallback.GetTrackInfo()
}

func (w *AudioPlayerWrapper) GetStateString() string {
	return w.Fallback.GetStateString()
}