package listenerutil

import (
	"fmt"
	"net/http"
	"strconv"
)

type ResponseWriter struct {
	wrapped http.ResponseWriter
	// headers contain a map of response code to header map such that
	// headers[status][header name] = header value
	// this map also contains values for hundred-level values in the format "1xx", "2xx", etc
	headers       map[string]map[string]string
	headerWritten bool
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.headerWritten = true
	w.setCustomResponseHeaders(statusCode)
	w.wrapped.WriteHeader(statusCode)
}

func (w *ResponseWriter) Header() http.Header {
	return w.wrapped.Header()
}

func (w *ResponseWriter) Write(data []byte) (int, error) {
	// The default behavior of http.ResponseWriter.Write is such that if WriteHeader has not
	// yet been called, it calls it with the below line. We will copy that logic so that our
	// WriteHeader function is called rather than http.ResponseWriter.WriteHeader
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.wrapped.Write(data)
}

func (w *ResponseWriter) setCustomResponseHeaders(statusCode int) {
	sch := w.headers
	if sch == nil {
		return
	}

	// Check the validity of the status code
	if statusCode >= 600 || statusCode < 100 {
		return
	}

	// Setter function to set headers
	setter := func(headerMap map[string]string) {
		for header, value := range headerMap {
			w.Header().Set(header, value)
		}
	}

	// Setting the default headers first
	if val, ok := sch["default"]; ok {
		setter(val)
	}

	// Then setting the generic hundred-level headers
	d := fmt.Sprintf("%dxx", statusCode/100)
	if val, ok := sch[d]; ok {
		setter(val)
	}

	// Finally setting the status-specific headers
	if val, ok := sch[strconv.Itoa(statusCode)]; ok {
		setter(val)
	}
}

type uiRequestFunc func(*http.Request) bool

// wrapCustomHeadersHandler wraps the handler to pass a custom ResponseWriter struct to all
// later wrappers and handlers to assign custom headers by status code. This wrapper must
// be the outermost wrapper to function correctly.
func WrapCustomHeadersHandler(h http.Handler, config *ListenerConfig, isUiRequest uiRequestFunc) http.Handler {
	// TODO: maybe we should perform some preparsing here on the headers? check for duplicates,
	// headers that aren't allowed, etc. could also update that map to actually be int -> str
	// rather than str -> str which requires converting status to strings, and may be slower

	uiHeaders := config.CustomUiResponseHeaders
	apiHeaders := config.CustomApiResponseHeaders

	// TODO: we should also set the default headers here. whether or not they are stored in
	// consts/etc is another thing
	// NOTE: this is for boundary specific things. universal things would go in go-secure-stdlib

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// this function is extremely generic as all we want to do is wrap the http.ResponseWriter
		// in our own ResponseWriter above, which will then perform all the logic we actually want

		var headers map[string]map[string]string

		if isUiRequest(req) {
			headers = uiHeaders
		} else {
			headers = apiHeaders
		}

		wrappedWriter := &ResponseWriter{
			wrapped: w,
			headers: headers,
		}
		h.ServeHTTP(wrappedWriter, req)
	})
}
