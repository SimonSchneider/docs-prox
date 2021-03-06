package file

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SimonSchneider/docs-prox/pkg/openapi"
	"github.com/fsnotify/fsnotify"
)

type fileSpec struct {
	path string
}

func (s *fileSpec) Get() ([]byte, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ioutil.ReadAll(file)
}

func newCachedFileSpec(path string) openapi.Spec {
	return openapi.Cached(&fileSpec{path: path}, 20*time.Second)
}

// Configure the store to add the path for json files with prefix
func Configure(ctx context.Context, store openapi.SpecStore, path, prefix, jsonExt, urlExt string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fileRepository: unable to start filewatcher: %w", err)
	}
	dirWatcher := &dirWatcher{
		source:  fmt.Sprintf("dirWatcher-%s", path),
		prefix:  prefix,
		jsonExt: jsonExt,
		urlExt:  urlExt,
		watcher: watcher,
		store:   store,
	}
	go dirWatcher.start(ctx)
	err = dirWatcher.add(path)
	if err != nil {
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
	source          string
	prefix          string
	jsonExt, urlExt string
	watcher         *fsnotify.Watcher
	store           openapi.SpecStore
}

type changeType int32

const (
	add changeType = iota
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

func (d *dirWatcher) change(path string, cType changeType) {
	if keyType, key, ok := d.getKey(path); ok {
		switch keyType {
		case jsonKey:
			d.changeJSONFile(key, path, cType)
		case urlKey:
			d.changeURLFile(key, path, cType)
		}
	}
}

func (d *dirWatcher) changeJSONFile(key, path string, cType changeType) {
	switch cType {
	case add:
		d.store.Put(d.source, key, newCachedFileSpec(path))
	case remove:
		d.store.Remove(d.source, key)
	}
}

func (d *dirWatcher) changeURLFile(key, path string, cType changeType) {
	source := fmt.Sprintf("%s-%s", d.source, key)
	switch cType {
	case add:
		file, err := os.Open(path)
		if err != nil {
			log.Printf("unable to parse url file %s: %v\n", path, err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		specs := make(map[string]openapi.Spec)
		for scanner.Scan() {
			row := scanner.Text()
			if split := strings.SplitN(row, ": ", 2); len(split) == 2 {
				specs[strings.Trim(split[0], " ")] = openapi.NewCachedRemoteSpec(strings.Trim(split[1], " "), 20*time.Second)
			} else {
				log.Printf("unexpected file formatting in file %s row '%s'\n", path, row)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("error when scanning file %s: %v\n", path, err)
			return
		}
		d.store.ReplaceAllOf(source, specs)
	case remove:
		d.store.RemoveAllOf(source)
	}
}

type keyType int32

const (
	jsonKey keyType = iota
	urlKey
)

func (d *dirWatcher) getKey(path string) (keyType, string, bool) {
	fileName := filepath.Base(path)
	if strings.HasPrefix(fileName, d.prefix) {
		withoutPrefix := strings.TrimPrefix(fileName, d.prefix)
		switch filepath.Ext(fileName) {
		case d.jsonExt:
			return jsonKey, strings.TrimSuffix(withoutPrefix, d.jsonExt), true
		case d.urlExt:
			return urlKey, strings.TrimSuffix(withoutPrefix, d.urlExt), true
		}
	}
	return jsonKey, "", false
}
