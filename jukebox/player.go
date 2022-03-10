package jukebox

import (
	"fmt"
	"io"

	"github.com/hajimehoshi/oto/v2"
	"go.senan.xyz/gonic/transcode"
)

type PlayerFunc func(io.Reader, transcode.Profile) (Player, error)

type Player interface {
	Pause()
	Play()
	IsPlaying() bool
	Reset()
	Volume() float64
	SetVolume(volume float64)
	UnplayedBufferSize() int
	Close() error
}

func OtoPlayer(r io.Reader, profile transcode.Profile) (Player, error) {
	otoc, wait, err := oto.NewContext(int(profile.SampleRate()), int(profile.NumChannels()), int(profile.BitsPerSample())/8)
	if err != nil {
		return nil, fmt.Errorf("create oto context: %w", err)
	}
	<-wait

	return otoc.NewPlayer(r), nil
}
