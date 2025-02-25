package watcher

import (
	"context"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

func Watch(ctx context.Context, filename string, callback func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create file watcher")
	}
	defer watcher.Close()

	err = watcher.Add(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to watch file")
	}

	log.Info().Str("status", "running").Str("target", filename).Msg("watcher")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&(fsnotify.Write) != 0 {
				log.Info().Str("event", "modified").Str("target", filename).Msg("watcher")
				callback()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("watcher")

		case <-ctx.Done():
			log.Info().Str("status", "stopping").Msg("watcher")
			return
		}
	}
}
