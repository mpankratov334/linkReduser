package main

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

var log = logrus.New()

func setLogger() {
	logfile, err := os.OpenFile("info.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.Out = logfile
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)
}

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func WithLogging(fnHandler func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		fnHandler(&lw, r)
		duration := time.Since(start).Milliseconds()

		log.WithFields(logrus.Fields{
			"uri":      r.RequestURI,
			"method":   r.Method,
			"status":   responseData.status,
			"duration": duration,
			"size":     responseData.size,
		}).Info("request served")
	}
	return logFn
}
