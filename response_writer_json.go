package cascadht

import (
	"encoding/json"
	"mime"
	"net/http"
)

const (
	mediaTypeNDJson = "application/x-ndjson"
	mediaTypeJson   = "application/json"
	mediaTypeAny    = "*/*"
)

var (
	_ acceptor            = (*jsonAcceptor)(nil)
	_ http.ResponseWriter = (*jsonAcceptor)(nil)
)

type jsonAcceptor struct {
	w       http.ResponseWriter
	f       http.Flusher
	encoder *json.Encoder
	nd      bool
}

func newJsonAcceptor(w http.ResponseWriter) jsonAcceptor {
	return jsonAcceptor{
		w:       w,
		encoder: json.NewEncoder(w),
	}
}

func (i *jsonAcceptor) Accept(r *http.Request) error {
	accepts := r.Header.Values("Accept")
	var okJson bool
	for _, accept := range accepts {
		mt, _, err := mime.ParseMediaType(accept)
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
			i.nd = true
			okJson = true
		}
		if i.nd && okJson {
			break
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
		return errHttpResponse{message: "media type not supported", status: http.StatusBadRequest}
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

func (i *jsonAcceptor) Header() http.Header {
	return i.w.Header()
}

func (i *jsonAcceptor) Write(b []byte) (int, error) {
	return i.w.Write(b)
}

func (i *jsonAcceptor) WriteHeader(code int) {
	i.w.WriteHeader(code)
}
