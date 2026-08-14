package memory

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

type Store struct {
	mu   sync.Mutex
	path string
	f    *os.File
	w    *bufio.Writer
	enc  *json.Encoder
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) ensureOpenLocked() error {
	if s.f != nil {
		return nil
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open store %s: %w", s.path, err)
	}
	s.f = f
	s.w = bufio.NewWriter(f)
	s.enc = json.NewEncoder(s.w)
	return nil
}

func (s *Store) AppendMessage(m Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureOpenLocked(); err != nil {
		return err
	}
	if err := s.enc.Encode(m); err != nil {
		return fmt.Errorf("encode message: %w", err)
	}
	return s.w.Flush()
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.f == nil {
		return nil
	}
	if err := s.w.Flush(); err != nil {
		_ = s.f.Close()
		return fmt.Errorf("flush store: %w", err)
	}
	return s.f.Close()
}

func LoadHistory(path string) ([]Message, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open store %s: %w", path, err)
	}
	defer f.Close()

	var history []Message
	dec := json.NewDecoder(f)
	for {
		var m Message
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode message: %w", err)
		}
		history = append(history, m)
	}
	return history, nil
}
