package config

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func Watch(filename string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(filename)
	if err != nil {
		log.Fatalf("Failed to watch file %s: %v", filename, err)
	}

	log.Printf("Watching config file: %s", filename)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				log.Printf("Config file changed: %s", event.Name)

				if err := Load(filename); err != nil {
					log.Printf("Failed to reload config: %v", err)
				} else {
					log.Println("Configuration reloaded successfully.")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
