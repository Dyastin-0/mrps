package config

import (
	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

func Watch(filename string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("Watcher")
	}
	defer watcher.Close()

	err = watcher.Add(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("Watcher - Add file")
	}

	log.Info().Str("Status", "running").Str("Target", filename).Msg("Watcher")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				log.Printf("Config file changed: %s", event.Name)

				if err := config.Load(filename); err != nil {
					log.Printf("Failed to reload config: %v", err)
				} else {
					log.Info().Str("Status", "reloaded").Str("Target", filename).Msg("Watcher")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("Watcher")
		}
	}
}
