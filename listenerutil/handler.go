package listenerutil

import (
	"net/http"
)

type ResponseWriter struct {
	wrapped http.ResponseWriter
	// headers contain a map of response code to header map such that
	// headers[status][header name] = header value
	// this map also contains values for hundred-level values in the format 1: "1xx", 2: "2xx", etc
	// defaults are set to 0
	headers       map[int]map[string]string
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
	if val, ok := sch[0]; ok {
		setter(val)
	}

	// Then setting the generic hundred-level headers
	// Note: integer division always rounds down, so 499/100 = 4
	if val, ok := sch[statusCode/100]; ok {
		setter(val)
	}

	// Finally setting the status-specific headers
	if val, ok := sch[statusCode]; ok {
		setter(val)
	}
}

type uiRequestFunc func(*http.Request) bool

// WrapCustomHeadersHandler wraps the handler to pass a custom ResponseWriter struct to all
// later wrappers and handlers to assign custom headers by status code. This wrapper must
// be the outermost wrapper to function correctly.
func WrapCustomHeadersHandler(h http.Handler, config *ListenerConfig, isUiRequest uiRequestFunc) http.Handler {
	// TODO: maybe we should perform some preparsing here on the headers? check for duplicates,
	// headers that aren't allowed, etc.

	uiHeaders := config.CustomUiResponseHeaders
	apiHeaders := config.CustomApiResponseHeaders

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// this function is extremely generic as all we want to do is wrap the http.ResponseWriter
		// in our own ResponseWriter above, which will then perform all the logic we actually want

		var headers map[int]map[string]string

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
