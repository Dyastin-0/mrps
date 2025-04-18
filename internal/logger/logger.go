package logger

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/hijack"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/nxadm/tail"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Subscribers = sync.Map{}
	LeftBehind  = sync.Map{}
)

func Init() {
	logFile := &lumberjack.Logger{
		Filename:   "./logs/mrps.log",
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   true,
	}

	zerolog.TimeFieldFormat = time.RFC3339
	logger := zerolog.New(logFile).With().Timestamp().Logger()
	log.Logger = logger
}

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statusCode := hijack.StatusCode(next, w, r)

		log.Info().
			Str("method", r.Method).
			Str("host", r.Host).
			Int("code", statusCode).
			Msg("access")
	})
}

type LogData struct {
	Type string `json:"type"`
	Log  string `json:"log"`
}

func InitNotifier(ctx context.Context) {
	log.Info().Str("status", "running").Msg("logger")

	t, err := tail.TailFile("./logs/mrps.log", tail.Config{
		Follow: true,
		ReOpen: true,
		Logger: tail.DiscardingLogger,
	})
	if err != nil {
		log.Error().Err(err).Msg("logger")
		return
	}

	log.Info().Str("status", "tailing").Str("target", "./logs/mrps.log").Msg("logger")

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("status", "stopping").Msg("logger")
			t.Stop()
			return

		case line := <-t.Lines:
			if line == nil {
				continue
			}
			if line.Err != nil {
				continue
			}

			Subscribers.Range(func(key, value interface{}) bool {
				if _, ok := LeftBehind.Load(key.(string)); ok {
					return true
				}

				if ok := ws.Clients.Exists(key.(string)); !ok {
					return true
				}

				Data := LogData{
					Type: "log",
					Log:  line.Text,
				}

				dataBytes, err := json.Marshal(Data)
				if err != nil {
					return true
				}

				token := key.(string)
				ws.Clients.Send(token, dataBytes)
				return true
			})
		}
	}
}

func CatchUp(key string) {
	t, err := tail.TailFile("./logs/mrps.log", tail.Config{
		Follow: false,
		Logger: tail.DiscardingLogger,
	})

	if err != nil {
		log.Error().Err(err).Msg("logger")
		return
	}
	defer t.Stop()

	for line := range t.Lines {
		if line == nil || line.Err != nil {
			log.Error().Err(line.Err).Msg("logger")
			continue
		}

		Data := LogData{
			Type: "log",
			Log:  line.Text,
		}

		dataBytes, err := json.Marshal(Data)
		if err != nil {
			log.Logger.Error().Err(err).Msg("logger")
		}

		ws.Clients.Send(key, dataBytes)
	}

	LeftBehind.Delete(key)
	log.Info().Str("status", "updated").Msg("logger")
	Subscribers.Store(key, true)
}
