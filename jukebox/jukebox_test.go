package jukebox_test

import (
	"path/filepath"
	"testing"

	"github.com/matryer/is"
	"go.senan.xyz/gonic/jukebox"
)

func TestPlay(t *testing.T) {
	t.Parallel()
	j := newMockJukebox(t)
	is := is.New(t)

	is.NoErr(j.SetItems([]string{
		"testdata/10s.mp3",
		"testdata/10s.mp3",
	}))

	is.NoErr(j.Play())

	status, err := j.GetStatus()
	is.NoErr(err)
	is.Equal(status.Playing, true)
	is.Equal(status.CurrentIndex, 0)
	is.Equal(status.Playing, true)
	is.Equal(status.Position, 0)
	is.Equal(status.GainPct, 100)

	is.NoErr(j.Play())

	status, err = j.GetStatus()
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

	// read the whole the first 10s track

	// j.player.ReadN(secsToBytes(10))
	// time.Sleep(100 * time.Millisecond) // let process exit, skip to next

	// is.Equal(j.GetStatus().CurrentIndex, 1)
	// is.Equal(j.GetStatus().Playing, true)
	// is.Equal(j.GetStatus().Position, 0)

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
	testBitrate = 44_100
)

func newMockJukebox(t *testing.T) *jukebox.Jukebox {
	sockPath := filepath.Join(t.TempDir(), "sock")
	j, err := jukebox.New(sockPath, nil)
	if err != nil {
		t.Fatalf("error creating jukebox: %v", err)
	}
	t.Cleanup(func() {
		j.Quit()
	})
	return j
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

func secsToBytes(secs int) int {
	return secs * int(testBitrate/8)
}
