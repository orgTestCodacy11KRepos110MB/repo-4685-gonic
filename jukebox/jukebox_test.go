package jukebox_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matryer/is"
	"go.senan.xyz/gonic/jukebox"
	"go.senan.xyz/gonic/jukebox/playlist"
	"go.senan.xyz/gonic/transcode"
)

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		os.Exit(0)
		return
	}
	os.Exit(m.Run())
}

func TestPlay(t *testing.T) {
	t.Parallel()
	j := newMockJukebox(t)
	is := is.New(t)

	go j.WatchForItems()

	j.SetItems([]*playlist.Item{
		playlist.NewItem(0, "testdata/10s.mp3"),
		playlist.NewItem(0, "testdata/10s.mp3"),
	})
	j.Play()
	is.NoErr(withTimeout(1*time.Second, waitUntil(func() bool {
		return j.GetStatus().Playing
	})))

	is.Equal(j.GetStatus().CurrentIndex, 0)
	is.Equal(j.GetStatus().Playing, true)
	is.Equal(j.GetStatus().Position, 0)
	is.Equal(j.GetStatus().Gain, 1.0)

	j.Play()
	is.Equal(j.GetStatus().CurrentIndex, 0)
	is.Equal(j.GetStatus().Playing, true)

	j.Pause()
	is.Equal(j.GetStatus().CurrentIndex, 0)
	is.Equal(j.GetStatus().Playing, false)

	j.Play()
	is.Equal(j.GetStatus().CurrentIndex, 0)
	is.Equal(j.GetStatus().Playing, true)
	is.Equal(j.GetStatus().Position, 0)

	// read the whole the first 10s track
	r := secsToBytes(j.profile, 10)
	j.player.ReadN(r)
	time.Sleep(100 * time.Millisecond) // let process exit, skip to next

	is.Equal(j.GetStatus().CurrentIndex, 1)
	is.Equal(j.GetStatus().Playing, true)
	is.Equal(j.GetStatus().Position, 0)

	// then half the second
	j.player.ReadN(secsToBytes(j.profile, 5))
	// check(ch{i: 1, secs: 5, playing: true})

	// then the other half
	j.player.ReadN(secsToBytes(j.profile, 5))
	time.Sleep(100 * time.Millisecond) // let process exit, skip to next

	// check(ch{i: 0, secs: 0})
}

func TestSkip(t *testing.T) {
	t.Parallel()
	j := newMockJukebox(t)
	is := is.New(t)

	go j.WatchForItems()

	is.NoErr(withTimeout(1*time.Second, func() {
		j.Skip(10, 10)
	}))

	is.Equal(j.GetStatus().CurrentIndex, 0) // no change, out of bounds
	is.True(!j.GetStatus().Playing)         // no change, out of bounds

	j.SetItems([]*playlist.Item{
		playlist.NewItem(0, "testdata/5s.mp3"),
		playlist.NewItem(0, "testdata/5s.mp3"),
		playlist.NewItem(0, "testdata/5s.mp3"),
	})
	j.Play()

	j.player.ReadN(secsToBytes(j.profile, 1))
	is.True(j.GetStatus().Playing)      // first track, read 1 sec, playing
	is.Equal(j.GetStatus().Position, 1) // first track, read 1 sec, position 1 sec

	is.NoErr(withTimeout(1*time.Second, func() {
		j.Skip(0, 2)
	}))

	j.player.ReadN(secsToBytes(j.profile, 1))
	is.True(j.GetStatus().Playing)      // first track, seek 2 secs, read 1 sec, playing
	is.Equal(j.GetStatus().Position, 4) // first track, seek 2 secs, read 1 sec, position 3 secs

	is.NoErr(withTimeout(1*time.Second, func() {
		j.Skip(1, 0)
	}))

	j.player.ReadN(secsToBytes(j.profile, 1))
	is.True(j.GetStatus().Playing)      // second track track, read 1 sec, playing
	is.Equal(j.GetStatus().Position, 1) // second track track, read 1 sec, position 1 secs
}

func TestQuit(t *testing.T) {
	t.Parallel()
	j := newMockJukebox(t)
	is := is.New(t)

	go j.WatchForItems()
	j.SetItems([]*playlist.Item{playlist.NewItem(0, "testdata/10s.mp3")})
	j.Play()

	// we should be able to quit even if we're in the middle of transcoding
	is.NoErr(withTimeout(1*time.Second, func() {
		j.Quit()
	}))
}

func TestPlaylist(t *testing.T) {
	j := newMockJukebox(t)
	is := is.New(t)

	go j.WatchForItems()

	is.Equal(len(j.GetItems()), 0)

	j.SetItems([]*playlist.Item{
		playlist.NewItem(0, "testdata/5s.mp3"),
		playlist.NewItem(1, "testdata/5s.mp3"),
		playlist.NewItem(2, "testdata/5s.mp3"),
	})
	is.Equal(len(j.GetItems()), 3)

	j.AppendItems([]*playlist.Item{
		playlist.NewItem(3, "testdata/5s.mp3"),
	})
	is.Equal(len(j.GetItems()), 4)
	is.Equal(j.GetItems()[0].ID(), 0)
	is.Equal(j.GetItems()[1].ID(), 1)
	is.Equal(j.GetItems()[2].ID(), 2)
	is.Equal(j.GetItems()[3].ID(), 3)

	j.RemoveItem(1)
	is.Equal(len(j.GetItems()), 3)
	is.Equal(j.GetItems()[0].ID(), 0)
	is.Equal(j.GetItems()[1].ID(), 2)
	is.Equal(j.GetItems()[2].ID(), 3)

	j.RemoveItem(10)
	is.Equal(len(j.GetItems()), 3)
}

func TestGain(t *testing.T) {
	j := newMockJukebox(t)
	is := is.New(t)

	is.Equal(j.GetStatus().Gain, 1.0)
	is.Equal(j.GetGain(), 1.0)

	j.SetGain(0)
	is.Equal(j.GetStatus().Gain, 0.0)
	is.Equal(j.GetGain(), 0.0)

	j.SetGain(0.5)
	is.Equal(j.GetStatus().Gain, 0.5)
	is.Equal(j.GetGain(), 0.5)
}

type mockJukebox struct {
	t *testing.T
	*jukebox.Jukebox
	profile  transcode.Profile
	player   *mockPlayer
	quitOnce sync.Once
}

func newMockJukebox(t *testing.T) *mockJukebox {
	transcodeProfile := transcode.PCM16le

	var player *mockPlayer

	j, err := jukebox.New(
		transcode.NewFFmpegTranscoder(),
		transcodeProfile,
		func(r io.Reader, profile transcode.Profile) (jukebox.Player, error) {
			player = &mockPlayer{r: r}
			return player, nil
		},
	)
	if err != nil {
		t.Fatalf("error creating jukebox: %v", err)
	}

	mj := &mockJukebox{t: t, Jukebox: j, player: player, profile: transcodeProfile}
	mj.SetGain(1)
	t.Cleanup(func() {
		mj.Quit()
	})
	return mj
}

func (mj *mockJukebox) Quit() {
	mj.quitOnce.Do(mj.Jukebox.Quit)
}

type mockPlayer struct {
	r       io.Reader
	playing atomic.Bool
	gain    atomic.Uint32
}

func (m *mockPlayer) Pause()                  { m.playing.Store(false) }
func (m *mockPlayer) Play()                   { m.playing.Store(true) }
func (m *mockPlayer) IsPlaying() bool         { return m.playing.Load() }
func (m *mockPlayer) Reset()                  { m.playing.Store(false) }
func (m *mockPlayer) Volume() float64         { return float64(m.gain.Load()) / 10.0 }
func (m *mockPlayer) SetVolume(gain float64)  { m.gain.Store(uint32(gain * 10.0)) }
func (m *mockPlayer) UnplayedBufferSize() int { return 0 }
func (m *mockPlayer) Close() error            { return nil }

func (m *mockPlayer) ReadN(to int) {
	io.ReadAll(io.LimitReader(m.r, int64(to)))
}

var _ jukebox.Player = (*mockPlayer)(nil)

var ErrTimeout = fmt.Errorf("timeout")

func withTimeout(d time.Duration, f func()) error {
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()

	select {
	case <-time.After(d):
		return ErrTimeout
	case <-done:
		return nil
	}
}

func waitUntil(f func() bool) func() {
	return func() {
		for !f() {
		}
	}
}

func secsToBytes(profile transcode.Profile, secs int) int {
	return secs * int(profile.BitRate()/8)
}
