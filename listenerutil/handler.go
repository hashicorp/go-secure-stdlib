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
func WrapCustomHeadersHandler(h http.Handler, config *ListenerConfig, isUiRequest uiRequestFunc) (http.Handler, error) {
	// TODO: maybe we should perform some preparsing here on the headers? check for duplicates,
	// headers that aren't allowed, etc.

	// Perform some basic parsing to convert status codes from strings to int to avoid costly string
	// comparisons for every request.
	uiHeaders := map[int]map[string]string{}

	for status, headers := range config.CustomUiResponseHeaders {
		if status == "default" {
			uiHeaders[0] = headers
			continue
		}

		intStatus, err := strconv.Atoi(status)
		if err != nil {
			if _, err = fmt.Sscanf(status, "%dxx", &intStatus); err != nil {
				return nil, fmt.Errorf("status does not match expected format. should be a valid status code or formatted \"%%dxx\". was: %s", status)
			}
			if intStatus > 5 || intStatus < 1 {
				return nil, fmt.Errorf("status is not within valid range, must be between 1xx and 5xx. was: %s", status)
			}
		}
		if intStatus >= 600 || intStatus < 100 {
			return nil, fmt.Errorf("status is not within valid range, must be between 100 and 599. was: %s", status)
		}
		uiHeaders[intStatus] = headers
	}

	apiHeaders := map[int]map[string]string{}

	for status, headers := range config.CustomApiResponseHeaders {
		if status == "default" {
			apiHeaders[0] = headers
			continue
		}

		intStatus, err := strconv.Atoi(status)
		if err != nil {
			if _, err = fmt.Sscanf(status, "%dxx", &intStatus); err != nil {
				return nil, fmt.Errorf("status does not match expected format. should be a valid status code or formatted \"%%dxx\". was: %s", status)
			}
			if intStatus > 5 || intStatus < 1 {
				return nil, fmt.Errorf("status is not within valid range, must be between 1xx and 5xx. was: %s", status)
			}
		}
		if intStatus >= 600 || intStatus < 100 {
			return nil, fmt.Errorf("status is not within valid range, must be between 100 and 599. was: %s", status)
		}
		apiHeaders[intStatus] = headers
	}

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
	}), nil
}
