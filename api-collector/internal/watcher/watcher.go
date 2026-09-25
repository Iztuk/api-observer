// Package watcher handles file watching
package watcher

import (
	"context"
	"log"

	"github.com/fsnotify/fsnotify"
)

func Watch(ctx context.Context, dir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				log.Println("event:", event)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				log.Println("error:", err)

			case <-ctx.Done():
				log.Println("API Collector shutting down")
				return
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}

	<-make(chan struct{})

	return nil
}
