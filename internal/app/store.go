package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/terraform-explorer/terraform-explorer/internal/hclparser"
	"github.com/terraform-explorer/terraform-explorer/internal/semantic"
)

const (
	maxSnapshots  = 16
	maxSourceSize = 10 << 20
)

var (
	ErrSnapshotNotFound = errors.New("scan snapshot not found")
	ErrInvalidSource    = errors.New("invalid source file")
)

type Snapshot struct {
	ID        string           `json:"id"`
	CreatedAt time.Time        `json:"createdAt"`
	Analysis  hclparser.Result `json:"analysis"`
	Graph     semantic.Graph   `json:"graph"`
}

type Source struct {
	File    string `json:"file"`
	Content string `json:"content"`
}

type Store struct {
	mu        sync.RWMutex
	snapshots map[string]*Snapshot
	order     []string
}

func NewStore() *Store {
	return &Store{snapshots: make(map[string]*Snapshot), order: make([]string, 0, maxSnapshots)}
}

func (store *Store) Scan(path string) (*Snapshot, error) {
	result, err := hclparser.ScanDir(path)
	if err != nil {
		return nil, err
	}
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	snapshot := &Snapshot{
		ID:        id,
		CreatedAt: time.Now().UTC(),
		Analysis:  result,
		Graph:     semantic.BuildGraph(result),
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.order) == maxSnapshots {
		delete(store.snapshots, store.order[0])
		store.order = store.order[1:]
	}
	store.snapshots[id] = snapshot
	store.order = append(store.order, id)
	return snapshot, nil
}

func (store *Store) Get(id string) (*Snapshot, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	snapshot, ok := store.snapshots[id]
	if !ok {
		return nil, ErrSnapshotNotFound
	}
	return snapshot, nil
}

func (store *Store) ReadSource(id, file string) (Source, error) {
	snapshot, err := store.Get(id)
	if err != nil {
		return Source{}, err
	}
	if file == "" || filepath.IsAbs(file) || filepath.Clean(file) != file || filepath.Dir(file) != "." {
		return Source{}, ErrInvalidSource
	}

	allowed := false
	for _, candidate := range snapshot.Analysis.Files {
		if candidate.Path == file {
			allowed = true
			break
		}
	}
	if !allowed {
		return Source{}, ErrInvalidSource
	}

	path := filepath.Join(snapshot.Analysis.Module.Path, file)
	info, err := os.Lstat(path)
	if err != nil {
		return Source{}, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > maxSourceSize {
		return Source{}, ErrInvalidSource
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return Source{}, err
	}
	return Source{File: file, Content: string(content)}, nil
}

func randomID() (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("create scan id: %w", err)
	}
	return hex.EncodeToString(value), nil
}
