package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const DefaultSnippetLength = 60

type SessionMeta struct {
	ID      string
	Path    string
	Snippet string
	ModTime time.Time
}

type Session struct {
	SessionMeta
	history []Message
	store   *Store
}

func (s *Session) Append(m Message) error {
	return s.store.AppendMessage(m)
}

func (s *Session) Close() error {
	return s.store.Close()
}

func (s *Session) History() []Message {
	return s.history
}

type SessionManager struct {
	dir string

	mu     sync.Mutex
	active *Session
}

func NewSessionManager(dir string) (*SessionManager, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}
	return &SessionManager{dir: dir}, nil
}

func (sm *SessionManager) List() ([]SessionMeta, error) {
	entries, err := os.ReadDir(sm.dir)
	if err != nil {
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}
	out := make([]SessionMeta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(sm.dir, e.Name())

		snippet, err := firstUserSnippet(path, DefaultSnippetLength)
		if err != nil {
			snippet = "Couldn't load snippet"
		}

		out = append(out, SessionMeta{
			ID:      strings.TrimSuffix(e.Name(), ".jsonl"),
			Path:    path,
			Snippet: snippet,
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ModTime.After(out[j].ModTime)
	})
	return out, nil
}

func firstUserSnippet(path string, maxChars int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("open store %s: %w", path, err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for {
		var m Message
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				return "", nil
			}
			return "", fmt.Errorf("decode message: %w", err)
		}
		if m.Role == RoleUser && m.Content != "" {
			return truncateSnippet(m.Content, maxChars), nil
		}
	}
}

func truncateSnippet(s string, maxChars int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= maxChars {
		return s
	}
	return string(r[:maxChars]) + "…"
}

func (sm *SessionManager) Create() (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.closeActiveLocked(); err != nil {
		return nil, err
	}
	id := time.Now().Format("20060102-150405.000")
	path := filepath.Join(sm.dir, id+".jsonl")
	store := NewStore(path)
	session := &Session{
		SessionMeta: SessionMeta{ID: id, Path: path, ModTime: time.Now()},
		store:       store,
	}
	sm.active = session
	return session, nil
}

func (sm *SessionManager) Open(id string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.closeActiveLocked(); err != nil {
		return nil, err
	}
	path := filepath.Join(sm.dir, id+".jsonl")
	history, err := LoadHistory(path)
	if err != nil {
		return nil, err
	}

	store := NewStore(path)
	modTime := time.Now()
	if info, err := os.Stat(path); err == nil {
		modTime = info.ModTime()
	}
	session := &Session{
		SessionMeta: SessionMeta{ID: id, Path: path, ModTime: modTime},
		store:       store,
		history:     history,
	}
	sm.active = session
	return session, nil
}

func (sm *SessionManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.closeActiveLocked()
}

func (sm *SessionManager) closeActiveLocked() error {
	if sm.active == nil {
		return nil
	}
	err := sm.active.Close()
	sm.active = nil
	return err
}
