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
	"github.com/hpcloud/tail"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Subscribers = sync.Map{}
var LeftBehind = sync.Map{}

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
			Str("url", r.URL.Path).
			Int("code", lrw.statusCode).
			Msg("Access")
	})
}

type LogData struct {
	Type string `json:"type"`
	Log  string `json:"log"`
}

func InitNotifier(ctx context.Context) {
	log.Info().Str("Status", "Running").Msg("Logger - Notifier")

	t, err := tail.TailFile("./logs/mrps.log", tail.Config{
		Follow: true,
		ReOpen: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("Logger - Notify")
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("Status", "stopping").Msg("Logger - Notifier")
			t.Stop()
			return

		case line := <-t.Lines:
			if line == nil || line.Err != nil {
				log.Error().Err(line.Err).Msg("Logger - Notifier")
				continue
			}

			Subscribers.Range(func(key, value interface{}) bool {
				if _, ok := LeftBehind.Load(key); ok {
					return true
				}

				logData := LogData{
					Type: "log",
					Log:  line.Text,
				}

				marshalLogData, err := json.Marshal(logData)
				if err != nil {
					log.Logger.Error().Err(err).Msg("Logger - Notifier")
				}

				err = ws.SendData(key.(string), marshalLogData)
				if err != nil {
					fmt.Println(err)
				}
				return true
			})
		}
	}
}

func CatchUp(key string) {
	t, err := tail.TailFile("./logs/mrps.log", tail.Config{Follow: false})
	if err != nil {
		log.Error().Err(err).Msg("Logger - CatchUp")
		return
	}
	defer t.Cleanup()

	for line := range t.Lines {
		if line == nil || line.Err != nil {
			log.Error().Err(line.Err).Msg("Logger - CatchUp")
			continue
		}

		logData := LogData{
			Type: "log",
			Log:  line.Text,
		}

		marshalLogData, err := json.Marshal(logData)
		if err != nil {
			log.Logger.Error().Err(err).Msg("Logger - CatchUp")
		}

		ws.SendData(key, marshalLogData)
	}

	LeftBehind.Delete(key)
	log.Info().Str("Status", "caught up").Msg("Logger - CatchUp")
	Subscribers.Store(key, true)
}
