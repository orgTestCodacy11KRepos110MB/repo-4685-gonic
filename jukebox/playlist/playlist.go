package playlist

import (
	"errors"
	"sync"
	"time"
)

type Item struct {
	id   int
	path string
	seek time.Duration
}

func NewItem(id int, path string) *Item {
	return &Item{id: id, path: path}
}

func (i *Item) ID() int                    { return i.id }
func (i *Item) Path() string               { return i.path }
func (i *Item) Seek() time.Duration        { return i.seek }
func (i *Item) SetSeek(seek time.Duration) { i.seek = seek }

type Playlist struct {
	mu    sync.RWMutex
	index int
	items []*Item
}

func New() *Playlist {
	var pl Playlist
	pl.Reset()
	return &pl
}

var ErrOOB = errors.New("player index out of bounds")

// read funcs
func (p *Playlist) Peek() (int, *Item) {
	defer read(&p.mu)()
	if p.index == -1 {
		return -1, nil
	}
	return p.index, p.items[p.index]
}

func (p *Playlist) Items() []*Item {
	defer read(&p.mu)()
	return append([]*Item(nil), p.items...)
}

// write funcs
func (p *Playlist) Inc() (*Item, error) {
	defer write(&p.mu)()
	if !inbounds(len(p.items), p.index+1) {
		return nil, ErrOOB
	}
	p.index++
	return p.items[p.index], nil
}
func (p *Playlist) IncIfEmpty() (*Item, error) {
	defer write(&p.mu)()
	if p.index != -1 {
		return nil, nil
	}
	if !inbounds(len(p.items), p.index+1) {
		return nil, ErrOOB
	}
	p.index++
	return p.items[p.index], nil
}
func (p *Playlist) Set(i int) (*Item, error) {
	defer write(&p.mu)()
	if !inbounds(len(p.items), i) {
		return nil, ErrOOB
	}
	p.index = i
	return p.items[p.index], nil
}

func (p *Playlist) RemoveItem(i int) error {
	defer write(&p.mu)()
	if !inbounds(len(p.items), i) {
		return ErrOOB
	}
	p.items = append(p.items[:i], p.items[i+1:]...)
	return nil
}

func (p *Playlist) SetItems(items []*Item) {
	defer write(&p.mu)()
	p.index = -1
	p.items = items
}

func (p *Playlist) AppendItems(items []*Item) {
	defer write(&p.mu)()
	p.items = append(p.items, items...)
}

func (p *Playlist) Reset() {
	defer write(&p.mu)()
	p.index = -1
	p.items = []*Item{}
}

func read(mu *sync.RWMutex) func() {
	mu.RLock()
	return mu.RUnlock
}
func write(mu *sync.RWMutex) func() {
	mu.Lock()
	return mu.Unlock
}
func inbounds(length, i int) bool {
	return length > 0 && i >= 0 && i < length
}
