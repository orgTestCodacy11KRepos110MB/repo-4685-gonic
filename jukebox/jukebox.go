// author: AlexKraak (https://github.com/alexkraak/)
// author: sentriz (https://github.com/sentriz/)

package jukebox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dexterlb/mpvipc"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/slices"
)

type Jukebox struct {
	cmd  *exec.Cmd
	conn *mpvipc.Connection
}

func New(sockPath string, mpvExtraArgs []string) (*Jukebox, error) {
	var mpvArgs []string
	mpvArgs = append(mpvArgs, "--idle", "--no-config", "--no-video", mpvArg("--audio-display", "no"), mpvArg("--input-ipc-server", sockPath))
	mpvArgs = append(mpvArgs, mpvExtraArgs...)
	var j Jukebox
	j.cmd = exec.Command("mpv", mpvArgs...)
	if err := j.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start mpv process: %w", err)
	}
	j.cmd.Stdout = os.Stdout
	j.cmd.Stderr = os.Stderr
	time.Sleep(500 * time.Millisecond)
	j.conn = mpvipc.NewConnection(sockPath)
	if err := j.conn.Open(); err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}
	return &j, nil
}

func (j *Jukebox) GetItems() ([]string, error) {
	var resp mpvPlaylist
	if err := j.getDecode(&resp, "playlist"); err != nil {
		return nil, fmt.Errorf("get playlist: %w", err)
	}
	var items []string
	for _, item := range resp {
		items = append(items, item.Filename)
	}
	return items, nil
}

func (j *Jukebox) SetItems(items []string) error {
	tmp, cleanup, err := tmp()
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer cleanup()
	for _, item := range items {
		item, _ = filepath.Abs(item)
		fmt.Fprintln(tmp, item)
	}
	if _, err := j.conn.Call("loadlist", tmp.Name()); err != nil {
		return fmt.Errorf("load list: %w", err)
	}
	return nil
}

func (j *Jukebox) AppendItems(items []string) error {
	tmp, cleanup, err := tmp()
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer cleanup()
	for _, item := range items {
		fmt.Fprintln(tmp, item)
	}
	if _, err := j.conn.Call("loadlist", tmp.Name(), "append"); err != nil {
		return fmt.Errorf("load list: %w", err)
	}
	return nil
}

func (j *Jukebox) RemoveItem(i int) error {
	if _, err := j.conn.Call("playlist-remove", i); err != nil {
		return fmt.Errorf("playlist remove: %w", err)
	}
	return nil
}

func (j *Jukebox) Skip(i int, offsetSecs int) error {
	if _, err := j.conn.Call("playlist-play-index", i); err != nil {
		return fmt.Errorf("playlist play index: %w", err)
	}
	if _, err := j.conn.Call("seek", offsetSecs, "absolute"); err != nil {
		return fmt.Errorf("seek: %w", err)
	}
	return nil
}

func (j *Jukebox) ClearItems() error {
	if _, err := j.conn.Call("playlist-clear"); err != nil {
		return fmt.Errorf("seek: %w", err)
	}
	return nil
}

func (j *Jukebox) Quit() error {
	if _, err := j.conn.Call("quit"); err != nil {
		return fmt.Errorf("quit: %w", err)
	}
	if err := j.cmd.Wait(); err != nil {
		return fmt.Errorf("wait to quit: %w", err)
	}
	if err := j.conn.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return nil
}

func (j *Jukebox) Pause() error {
	if err := j.conn.Set("pause", true); err != nil {
		return fmt.Errorf("pause: %w", err)
	}
	return nil
}

func (j *Jukebox) Play() error {
	if err := j.conn.Set("pause", false); err != nil {
		return fmt.Errorf("pause: %w", err)
	}
	return nil
}

func (j *Jukebox) SetGainPct(v int) error {
	if err := j.conn.Set("volume", v); err != nil {
		return fmt.Errorf("set volume: %w", err)
	}
	return nil
}

func (j *Jukebox) GetGain() (float64, error) {
	var volume float64
	if err := j.getDecode(&volume, "volume"); err != nil {
		return 0, fmt.Errorf("get volume: %w", err)
	}
	return volume, nil
}

type Status struct {
	CurrentIndex int
	Playing      bool
	GainPct      int
	Position     int
}

func (j *Jukebox) GetStatus() (*Status, error) {
	var status Status
	_ = j.getDecode(&status.Position, "time-pos") // property may not always be there
	_ = j.getDecode(&status.GainPct, "volume")    // property may not always be there

	var paused bool
	_ = j.getDecode(&paused, "pause") // property may not always be there
	status.Playing = !paused

	var playlist mpvPlaylist
	_ = j.getDecode(&playlist, "playlist")

	status.CurrentIndex = slices.IndexFunc(playlist, func(pl mpvPlaylistItem) bool {
		return pl.Current
	})

	return &status, nil
}

func (j *Jukebox) getDecode(dest any, property string) error {
	raw, err := j.conn.Get(property)
	if err != nil {
		return fmt.Errorf("get property: %w", err)
	}
	if err := mapstructure.Decode(raw, dest); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}

type mpvPlaylist []mpvPlaylistItem
type mpvPlaylistItem struct {
	ID       int
	Filename string
	Current  bool
	Playing  bool
}

func tmp() (*os.File, func(), error) {
	tmp, err := os.CreateTemp("", "gonic-jukebox-")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp file: %w", err)
	}
	return tmp, func() {
		os.Remove(tmp.Name())
		tmp.Close()
	}, nil
}

func mpvArg(k, v string) string {
	return fmt.Sprintf("%s=%s", k, v)
}
