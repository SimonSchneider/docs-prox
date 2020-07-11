package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SimonSchneider/docs-prox/openapi"
	"github.com/fsnotify/fsnotify"
)

type fileSpec struct {
	path string
}

func (s *fileSpec) JSONSpec() (interface{}, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	var result interface{}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&result)
	return result, err
}

// Configure the store to add the path for json files with prefix
func Configure(ctx context.Context, store openapi.ApiStore, path, prefix string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fileRepository: unable to start filewatcher: %w", err)
	}
	dirWatcher := &dirWatcher{
		source:  fmt.Sprintf("dirWatcher-%s", path),
		prefix:  prefix,
		watcher: watcher,
		store:   store,
	}
	withCancel, cancel := context.WithCancel(ctx)
	go dirWatcher.start(withCancel)
	err = dirWatcher.add(path)
	if err != nil {
		cancel()
		return fmt.Errorf("fileRepository: unable to add path %s to directory Watcher: %w", path, err)
	}
	go func() {
		<-ctx.Done()
		fmt.Printf("fileRepository: stopping directory watcher\n")
		watcher.Close()
	}()
	return nil
}

type dirWatcher struct {
	source  string
	prefix  string
	watcher *fsnotify.Watcher
	store   openapi.ApiStore
}

type ChangeType int32

const (
	add ChangeType = iota
	remove
)

func (d *dirWatcher) add(path string) error {
	err := d.watcher.Add(path)
	if err != nil {
		return fmt.Errorf("fileRepository: could not access path %s: %w", path, err)
	}
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		d.change(p, add)
		return nil
	})
	if err != nil {
		return fmt.Errorf("fileRepository: could not walk path %s: %w", path, err)
	}
	return nil
}

func (d *dirWatcher) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("stopping directory processor\n")
			return
		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
				d.change(event.Name, remove)
			} else if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				d.change(event.Name, add)
			}
		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("error: %s\n", err)
		}
	}
}

func (d *dirWatcher) change(path string, cType ChangeType) {
	if key, ok := d.getKey(path); ok {
		switch cType {
		case add:
			d.store.Put(d.source, key, &fileSpec{path})
		case remove:
			d.store.Remove(d.source, key)
		}
	}
}

func (d *dirWatcher) getKey(path string) (string, bool) {
	fileName := filepath.Base(path)
	if strings.HasPrefix(fileName, d.prefix) && filepath.Ext(fileName) == ".json" {
		key := strings.TrimSuffix(strings.TrimPrefix(fileName, d.prefix), ".json")
		return key, true
	}
	return "", false
}
