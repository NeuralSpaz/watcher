package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type FileStateStore struct {
	sync.RWMutex
	files map[string]FileState
	count int
	save  chan FileState
	done  chan bool
}

type FileState struct {
	Path         string    `json:"path"`
	Hash         string    `json:"hash"`
	LastModified time.Time `json:"lastmodified"`
}

func NewFileStateStore(filestorepath string) *FileStateStore {
	s := &FileStateStore{files: make(map[string]FileState)}
	if filestorepath != "" {
		s.save = make(chan FileState)
		if err := s.load(filestorepath); err != nil {
			log.Println("FileStateStore load:", err)
		}
		go s.saveloop(filestorepath)
	}
	return s
}

func (s *FileStateStore) Set(f FileState) (bool, error) {
	s.Lock()
	defer s.Unlock()
	old, present := s.files[f.Path]
	if present {
		changed := old.Compare(f)
		if changed {
			s.files[f.Path] = f
		}
		return changed, errKeyExists
	}
	s.files[f.Path] = f
	return false, nil

}

var (
	errKeyExists = errors.New("key already exists")
)

func (f FileState) Compare(new FileState) bool {
	changed := false
	if f.Hash != new.Hash {
		log.Printf("file %s Has changed: old hash=%s new hash=%s\n", f.Path, f.Hash, new.Hash)
		changed = true
	}
	// if f.LastModified != new.LastModified {
	// 	log.Printf("file %s Has changed: old Modtime=%s new ModTime=%s\n", f.Path, f.LastModified, new.LastModified)
	// 	changed = true
	// }
	return changed
}

func (s *FileStateStore) Put(f FileState) error {

	changed, err := s.Set(f)
	if err == errKeyExists && changed {
		log.Println("stats changed")
		if s.save != nil {
			s.save <- f
		}
	}
	if err == errKeyExists {
		// log.Println("should igonore f")
		return nil
	}
	if s.save != nil {
		s.save <- f
	}

	return nil
}

func (s *FileStateStore) saveloop(filestorepath string) {
	f, err := os.OpenFile(filestorepath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Println("FileStateStore Write:", err)
		return
	}
	b := bufio.NewWriter(f)
	e := json.NewEncoder(b)
	defer f.Close()
	defer b.Flush()
	for {
		var err error
		select {
		case <-s.done:
			return
		case r, ok := <-s.save:
			if !ok {
				return
			}
			log.Printf("Updating Database with stats on file %s\n", r.Path)
			err = e.Encode(r)
			b.Flush()
		}
		if err != nil {
			log.Println("FileStateStore SaveLoop:", err)
		}
	}
}

func (s *FileStateStore) load(filestorepath string) error {
	f, err := os.Open(filestorepath)
	if err != nil {
		return err
	}
	defer f.Close()
	b := bufio.NewReader(f)
	d := json.NewDecoder(b)
	for {
		var fs FileState
		if err := d.Decode(&fs); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if _, err = s.Set(fs); err != nil {
			return err
		}
	}
	return nil
}
