// author: AlexKraak (https://github.com/alexkraak/)
// author: sentriz (https://github.com/sentriz/)

package jukebox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"time"

	"go.senan.xyz/gonic/countrw"
	"go.senan.xyz/gonic/jukebox/playlist"
	"go.senan.xyz/gonic/transcode"
)

type Jukebox struct {
	transcoder transcode.Transcoder
	profile    transcode.Profile
	pcmw       io.Writer
	pcmr       *countrw.CountReader

	player Player

	playlist *playlist.Playlist
	next     chan *playlist.Item
	quit     chan struct{}
}

func New(transcoder transcode.Transcoder, profile transcode.Profile, playerFunc PlayerFunc) (*Jukebox, error) {
	var j Jukebox
	j.transcoder = transcoder
	j.profile = profile

	pcmr, pcmw := io.Pipe()
	j.pcmw = pcmw
	j.pcmr = countrw.NewCountReader(pcmr)

	var err error
	j.player, err = playerFunc(j.pcmr, profile)
	if err != nil {
		return nil, fmt.Errorf("create player: %w", err)
	}

	j.playlist = playlist.New()
	j.next = make(chan *playlist.Item)
	j.quit = make(chan struct{})

	return &j, nil
}

func (j *Jukebox) WatchForItems() {
	var prevCancel context.CancelFunc
	for {
		select {
		case item := <-j.next:
			j.player.Pause()
			if prevCancel != nil {
				prevCancel()
			}
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				j.decodeToStream(ctx, item)
				cancel()
			}()
			prevCancel = cancel
		case <-j.quit:
			if prevCancel != nil {
				prevCancel()
			}
			j.clearBuffer()
			j.playlist.Reset()
			break
		}
	}
}

func (j *Jukebox) decodeToStream(ctx context.Context, item *playlist.Item) {
	profile := transcode.WithSeek(j.profile, item.Seek())
	j.clearBuffer()
	j.Play()
	err := j.transcoder.Transcode(ctx, profile, item.Path(), j.pcmw)
	if errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	if err != nil {
		log.Printf("decoding item: %v", err)
	}

	item, err = j.playlist.Inc()
	if err != nil {
		return
	}
	j.next <- item
}

func (j *Jukebox) GetItems() []*playlist.Item         { return j.playlist.Items() }
func (j *Jukebox) SetItems(items []*playlist.Item)    { j.playlist.SetItems(items) }
func (j *Jukebox) RemoveItem(i int) error             { return j.playlist.RemoveItem(i) }
func (j *Jukebox) AppendItems(items []*playlist.Item) { j.playlist.AppendItems(items) }

func (j *Jukebox) Skip(i int, offsetSecs int) {
	item, err := j.playlist.Set(i)
	if err != nil {
		return
	}
	item.SetSeek(time.Duration(offsetSecs) * time.Second)
	j.next <- item
}

func (j *Jukebox) ClearItems() {
	j.playlist.Reset()
	j.clearBuffer()
}

func (j *Jukebox) Quit() {
	j.playlist.Reset()
	j.clearBuffer()
	close(j.quit)
}

func (j *Jukebox) Pause() { j.player.Pause() }
func (j *Jukebox) Play() {
	j.player.Play()
	if item, _ := j.playlist.IncIfEmpty(); item != nil {
		j.next <- item
	}
}

func (j *Jukebox) SetGain(v float64) { j.player.SetVolume(v) }
func (j *Jukebox) GetGain() float64  { return j.player.Volume() }

func (j *Jukebox) clearBuffer() {
	j.player.Reset() // clear oto's buffer
	j.pcmr.Reset()   // clear our counter of how many bytes oto has read
}

type Status struct {
	CurrentIndex int
	Playing      bool
	Gain         float64
	Position     int
}

func (j *Jukebox) GetStatus() Status {
	var status Status
	if i, item := j.playlist.Peek(); item != nil {
		playedBits := (j.pcmr.Count() - uint64(j.player.UnplayedBufferSize())) * 8
		playedSecs := float64(playedBits) / float64(j.profile.BitRate())
		seekedSecs := item.Seek().Seconds()

		status.Position = int(math.Round(playedSecs + seekedSecs))
		status.CurrentIndex = i
	}
	status.Gain = j.player.Volume()
	status.Playing = j.player.IsPlaying()
	return status
}
