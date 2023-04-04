package caskadht

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"
)

const (
	mediaTypeNDJson = "application/x-ndjson"
	mediaTypeJson   = "application/json"
	mediaTypeAny    = "*/*"
)

var (
	_ selectiveResponseWriter = (*jsonResponseWriter)(nil)
	_ http.ResponseWriter     = (*jsonResponseWriter)(nil)
)

type jsonResponseWriter struct {
	w          http.ResponseWriter
	f          http.Flusher
	encoder    *json.Encoder
	nd         bool
	preferJson bool
}

func newJsonResponseWriter(w http.ResponseWriter, preferJson bool) jsonResponseWriter {
	return jsonResponseWriter{
		w:          w,
		encoder:    json.NewEncoder(w),
		preferJson: preferJson,
	}
}

func (i *jsonResponseWriter) Accept(r *http.Request) error {
	accepts := r.Header.Values("Accept")
	var okJson bool
	for _, accept := range accepts {
		amts := strings.Split(accept, ",")
		for _, amt := range amts {
			mt, _, err := mime.ParseMediaType(amt)
			if err != nil {
				logger.Debugw("Failed to check accepted response media type", "err", err)
				return errHttpResponse{message: "invalid Accept header", status: http.StatusBadRequest}
			}
			switch mt {
			case mediaTypeNDJson:
				i.nd = true
			case mediaTypeJson:
				okJson = true
			case mediaTypeAny:
				i.nd = !i.preferJson
				okJson = true
			}
			if i.nd && okJson {
				break
			}
		}
	}

	var okFlusher bool
	i.f, okFlusher = i.w.(http.Flusher)
	if !okFlusher && !okJson && i.nd {
		// Respond with error if the request only accepts ndjson and the server does not support
		// streaming.
		return errHttpResponse{message: "server does not support streaming response", status: http.StatusBadRequest}
	}
	if !okJson && !i.nd {
		return errHttpResponse{message: fmt.Sprintf("media type not supported: %s", accepts), status: http.StatusBadRequest}
	}

	if i.nd {
		i.w.Header().Set("Content-Type", mediaTypeNDJson)
		i.w.Header().Set("Connection", "Keep-Alive")
		i.w.Header().Set("X-Content-Type-Options", "nosniff")
	} else {
		i.w.Header().Set("Content-Type", mediaTypeJson)
	}
	return nil
}

func (i *jsonResponseWriter) Header() http.Header {
	return i.w.Header()
}

func (i *jsonResponseWriter) Write(b []byte) (int, error) {
	return i.w.Write(b)
}

func (i *jsonResponseWriter) WriteHeader(code int) {
	i.w.WriteHeader(code)
}
