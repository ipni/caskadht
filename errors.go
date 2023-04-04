package caskadht

import "net/http"

type errHttpResponse struct {
	message string
	status  int
}

func (e errHttpResponse) Error() string {
	return e.message
}

func (e errHttpResponse) WriteTo(w http.ResponseWriter) {
	http.Error(w, e.message, e.status)
}
