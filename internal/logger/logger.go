package logger

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/nxadm/tail"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Subscribers       = sync.Map{}
	LeftBehind        = sync.Map{}
	offsetBytes int64 = -20
)

func Init() {
	logFile := &lumberjack.Logger{
		Filename:   "./logs/mrps.log",
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   true,
	}

	zerolog.TimeFieldFormat = time.RFC3339
	logger := zerolog.New(logFile).With().Timestamp().Logger()
	log.Logger = logger
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (rw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not support Hijack")
}

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		lrw := newLoggingResponseWriter(w)

		next.ServeHTTP(lrw, r)

		log.Info().
			Str("method", r.Method).
			Str("host", r.Host).
			Int("code", lrw.statusCode).
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

func CatchUp(key string, readyChan chan bool) {
	ready := <-readyChan
	close(readyChan)

	if !ready {
		log.Error().Err(fmt.Errorf("failed to load logs")).Str("client", string(key[len(key)-10:])).Msg("websocket")
		return
	}

	t, err := tail.TailFile("./logs/mrps.log", tail.Config{
		Follow: false,
		Logger: tail.DiscardingLogger,
		// location fails sometimes - fix later
		Location: &tail.SeekInfo{
			Offset: offsetBytes,
			Whence: 2,
		},
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
	log.Info().Str("status", "updated").Str("offset", fmt.Sprint(offsetBytes*-1)+"bytes").Msg("logger")
	Subscribers.Store(key, true)
}
