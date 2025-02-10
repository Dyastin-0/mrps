package config

import (
	"context"
	"fmt"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

func Watch(ctx context.Context, filename string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("watcher")
	}
	defer watcher.Close()

	err = watcher.Add(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("watcher")
	}

	log.Info().Str("status", "running").Str("target", filename).Msg("watcher")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {

				if err := config.Load(ctx, filename); err != nil {
					log.Error().Err(fmt.Errorf("failed to reload")).Msg("watcher")
				} else {
					log.Info().Str("status", "reloaded").Str("target", filename).Msg("watcher")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("watcher")
		}
	}
}
