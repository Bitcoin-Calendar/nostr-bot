package bot

import (
	"encoding/json"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"os"
	"path/filepath"
	"syscall"
)

type Record struct {
	Event     nostr.Event `json:"event"`
	Delivered bool        `json:"delivered"`
}
type Store struct {
	dir  string
	lock *os.File
}

func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("another Nostr run is active")
	}
	return &Store{dir: dir, lock: f}, nil
}
func (s *Store) Close() { syscall.Flock(int(s.lock.Fd()), syscall.LOCK_UN); s.lock.Close() }
func (s *Store) file(pubkey, date string, id uint) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s-%s-%d.json", pubkey, date, id))
}
func (s *Store) Load(pubkey, date string, id uint) (*Record, error) {
	data, err := os.ReadFile(s.file(pubkey, date, id))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r Record
	if err = json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	ok, err := r.Event.CheckSignature()
	if err != nil || !ok || r.Event.ID != r.Event.GetID() || r.Event.PubKey != pubkey || r.Event.Kind != 1 {
		return nil, fmt.Errorf("invalid saved event for %d", id)
	}
	return &r, nil
}
func (s *Store) Save(pubkey, date string, id uint, r *Record) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".pending-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, s.file(pubkey, date, id)); err != nil {
		return err
	}
	d, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
