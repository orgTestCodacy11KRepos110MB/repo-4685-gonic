package jukebox_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matryer/is"
	"go.senan.xyz/gonic/jukebox"
	"golang.org/x/sys/unix"
)

func TestPlay(t *testing.T) {
	t.Parallel()
	j, audioPipePath := newJukebox(t)
	is := is.New(t)

	is.NoErr(j.SetItems([]string{
		"testdata/10s.mp3",
		"testdata/10s.mp3",
	}))

	is.NoErr(j.Play())

	status, err := j.GetStatus()
	is.NoErr(err)
	is.Equal(status.Playing, true)

	audioPipe, err := os.OpenFile(audioPipePath, os.O_RDONLY, os.ModeNamedPipe)
	is.NoErr(err)
	defer audioPipe.Close()

	progressSecs := func(n int) {
		reader := io.LimitReader(audioPipe, mpvPipeSecondsToBytes(n))
		for {
			fmt.Printf("+++ ||||||||||\n")
			n, _ := io.CopyN(io.Discard, reader, 1<<16)
			if n < (1 << 16) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	progressSecs(1)

	fmt.Printf("+++ xxxxxxxxxx\n")
	status, err = j.GetStatus()
	fmt.Printf("+++ yyyyyyyyyy\n")
	is.NoErr(err)
	is.Equal(status.Playing, true)
	is.Equal(status.CurrentIndex, 0)
	is.Equal(status.Playing, true)
	is.Equal(status.Position, 0)
	is.Equal(status.GainPct, 100)

	is.NoErr(j.Play())

	fmt.Printf("+++ zzzzzzzzzz\n")
	status, err = j.GetStatus()
	fmt.Printf("+++ {{{{{{{{{{\n")
	is.NoErr(err)
	is.Equal(status.CurrentIndex, 0)
	is.Equal(status.Playing, true)

	is.NoErr(j.Pause())

	status, err = j.GetStatus()
	is.NoErr(err)
	is.Equal(status.CurrentIndex, 0)
	is.Equal(status.Playing, false)

	is.NoErr(j.Play())

	status, err = j.GetStatus()
	is.NoErr(err)
	is.Equal(status.CurrentIndex, 0)
	is.Equal(status.Playing, true)
	is.Equal(status.Position, 0)

	progressSecs(1)

	status, err = j.GetStatus()
	is.NoErr(err)
	is.Equal(status.CurrentIndex, 0)
	is.Equal(status.Playing, true)
	is.Equal(status.Position, 5)

	// // then half the second
	// j.player.ReadN(secsToBytes(j.profile, 5))
	// // check(ch{i: 1, secs: 5, playing: true})

	// // then the other half
	// j.player.ReadN(secsToBytes(j.profile, 5))
	// time.Sleep(100 * time.Millisecond) // let process exit, skip to next

	// check(ch{i: 0, secs: 0})
}

// func TestSkip(t *testing.T) {
// 	t.Parallel()
// 	j := newMockJukebox(t)
// 	is := is.New(t)

// 	is.NoErr(withTimeout(1*time.Second, func() {
// 		j.Skip(10, 10)
// 	}))

// 	is.Equal(j.GetStatus().CurrentIndex, 0) // no change, out of bounds
// 	is.True(!j.GetStatus().Playing)         // no change, out of bounds

// 	j.SetItems([]string{
// 		"testdata/5s.mp3",
// 		"testdata/5s.mp3",
// 		"testdata/5s.mp3",
// 	})
// 	j.Play()

// 	j.player.ReadN(secsToBytes(j.profile, 1))
// 	is.True(j.GetStatus().Playing)      // first track, read 1 sec, playing
// 	is.Equal(j.GetStatus().Position, 1) // first track, read 1 sec, position 1 sec

// 	is.NoErr(withTimeout(1*time.Second, func() {
// 		j.Skip(0, 2)
// 	}))

// 	j.player.ReadN(secsToBytes(j.profile, 1))
// 	is.True(j.GetStatus().Playing)      // first track, seek 2 secs, read 1 sec, playing
// 	is.Equal(j.GetStatus().Position, 4) // first track, seek 2 secs, read 1 sec, position 3 secs

// 	is.NoErr(withTimeout(1*time.Second, func() {
// 		j.Skip(1, 0)
// 	}))

// 	j.player.ReadN(secsToBytes(j.profile, 1))
// 	is.True(j.GetStatus().Playing)      // second track track, read 1 sec, playing
// 	is.Equal(j.GetStatus().Position, 1) // second track track, read 1 sec, position 1 secs
// }

// func TestQuit(t *testing.T) {
// 	t.Parallel()
// 	j := newMockJukebox(t)
// 	is := is.New(t)

// 	j.SetItems([]string{"testdata/10s.mp3"})
// 	j.Play()

// 	// we should be able to quit even if we're in the middle of transcoding
// 	is.NoErr(withTimeout(1*time.Second, func() {
// 		j.Quit()
// 	}))
// }

// func TestPlaylist(t *testing.T) {
// 	j := newMockJukebox(t)
// 	is := is.New(t)

// 	is.Equal(len(j.GetItems()), 0)

// 	j.SetItems([]string{
// 		"testdata/5s.mp3",
// 		"testdata/5s.mp3",
// 		"testdata/5s.mp3",
// 	})
// 	is.Equal(len(j.GetItems()), 3)

// 	j.AppendItems([]string{
// 		"testdata/5s.mp3",
// 	})
// 	is.Equal(len(j.GetItems()), 4)
// 	is.Equal(j.GetItems()[0].ID(), 0)
// 	is.Equal(j.GetItems()[1].ID(), 1)
// 	is.Equal(j.GetItems()[2].ID(), 2)
// 	is.Equal(j.GetItems()[3].ID(), 3)

// 	j.RemoveItem(1)
// 	is.Equal(len(j.GetItems()), 3)
// 	is.Equal(j.GetItems()[0].ID(), 0)
// 	is.Equal(j.GetItems()[1].ID(), 2)
// 	is.Equal(j.GetItems()[2].ID(), 3)

// 	j.RemoveItem(10)
// 	is.Equal(len(j.GetItems()), 3)
// }

// func TestGain(t *testing.T) {
// 	j := newMockJukebox(t)
// 	is := is.New(t)

// 	is.Equal(j.GetStatus().Gain, 1.0)
// 	is.Equal(j.GetGain(), 1.0)

// 	j.SetGain(0)
// 	is.Equal(j.GetStatus().Gain, 0.0)
// 	is.Equal(j.GetGain(), 0.0)

// 	j.SetGain(0.5)
// 	is.Equal(j.GetStatus().Gain, 0.5)
// 	is.Equal(j.GetGain(), 0.5)
// }

const (
	mpvPipeSampleRate    = 44_100
	mpvPipeBitDepthBytes = 2
	mpvPipeNumChannels   = 2
)

func mpvPipeSecondsToBytes(secs int) int64 {
	return int64(secs * mpvPipeSampleRate * mpvPipeBitDepthBytes * mpvPipeNumChannels)
}

func newJukebox(t *testing.T) (*jukebox.Jukebox, string) {
	sockPath := filepath.Join(t.TempDir(), "sock")
	audioPipePath := filepath.Join(t.TempDir(), "audio")

	if err := unix.Mkfifo(audioPipePath, 0666); err != nil {
		t.Fatalf("create audio pipe: %v", err)
	}

	j, err := jukebox.New(
		sockPath,
		[]string{
			jukebox.MPVArg("--audio-samplerate", mpvPipeSampleRate),
			jukebox.MPVArg("--audio-format", "s16"),
			jukebox.MPVArg("--audio-channels", "stereo"),
			jukebox.MPVArg("--ao", "pcm"),
			jukebox.MPVArg("--ao-pcm-file", audioPipePath),
			jukebox.MPVArg("--ao-pcm-waveheader", false),
		})
	if err != nil {
		t.Fatalf("error creating jukebox: %v", err)
	}
	t.Cleanup(func() {
		j.Quit()
	})

	return j, audioPipePath
}

// var ErrTimeout = fmt.Errorf("timeout")

// func withTimeout(d time.Duration, f func()) error {
// 	done := make(chan struct{})
// 	go func() {
// 		f()
// 		close(done)
// 	}()

// 	select {
// 	case <-time.After(d):
// 		return ErrTimeout
// 	case <-done:
// 		return nil
// 	}
// }

// func waitUntil(f func() bool) func() {
// 	return func() {
// 		for !f() {
// 		}
// 	}
// }
