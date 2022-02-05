package tritonhttp

// approximate contribution: 102 in request.go
import (
	"bufio"
	"fmt"
	"strings"
)

type Request struct {
	Method string // e.g. "GET"
	URL    string // e.g. "/path/to/a/file"
	Proto  string // e.g. "HTTP/1.1"

	// Header stores misc headers excluding "Host" and "Connection",
	// which are stored in special fields below.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	Host  string // determine from the "Host" header
	Close bool   // determine from the "Connection" header
}

// ReadRequest tries to read the next valid request from br.
//
// If it succeeds, it returns the valid request read. In this case,
// bytesReceived should be true, and err should be nil.
//
// If an error occurs during the reading, it returns the error,
// and a nil request. In this case, bytesReceived indicates whether or not
// some bytes are received before the error occurs. This is useful to determine
// the timeout with partial request received condition.
func ReadRequest(br *bufio.Reader) (req *Request, bytesReceived bool, err error) {
	// @credit: Week4 TA Section
	startline, err := ReadLine(br)
	// fmt.Println("******req******", startline)
	if err != nil {
		return nil, false, err
	}

	req = &Request{}
	// req.Close = false
	req.Header = make(map[string]string)

	// Read start line

	req.Method, req.URL, req.Proto, err = parseStartLine(startline)
	// fmt.Println("************", req.URL)
	if err != nil {
		return nil, true, err
	}

	if !validMethod(req.Method, req.Proto) {
		return nil, true, badStringError("invalid method", req.Method)
	}
	if req.URL[0] != '/' {
		return nil, true, badStringError("Invalid URL: ", req.URL)
	}

	// Read headers
	for {
		line, err := ReadLine(br)
		if err != nil {
			fmt.Println("Line read failed", err)
			return nil, true, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		// Check required headers
		if strings.HasPrefix(line, "Host") {
			req.Host = strings.SplitN(line, " ", 2)[1]
			continue
		}

		// Handle special headers
		if strings.HasPrefix(line, "Connection") {
			req.Close = (strings.SplitN(line, " ", 2)[1] == "close")
			continue
		}

		key, val, err := sepHeaderline(line)
		if err != nil {
			return nil, true, fmt.Errorf("invalid headerline")
		}
		// key = CanonicalHeaderKey(strings.TrimSpace(key))

		if find := strings.ContainsAny(key, " !#$%&'*+@{}[]:;.^_`|~"); find {
			return nil, true, fmt.Errorf("key congtains invalid char")
		}

		req.Header[key] = val
	}

	// Check required headers
	if req.Host == "" {
		return nil, true, fmt.Errorf("missing valid host")
	}

	// Handle special headers

	return req, true, nil
}

func sepHeaderline(line string) (key string, val string, err error) {
	fields := strings.SplitN(line, ":", 2)
	if len(fields) != 2 {
		return "", "", fmt.Errorf("could not parse the request header line, got fields %v", fields)
	}
	return CanonicalHeaderKey(strings.TrimSpace(fields[0])), strings.TrimSpace(fields[1]), nil
}

func parseStartLine(line string) (string, string, string, error) {
	fields := strings.SplitN(line, " ", 3)
	if len(fields) != 3 {
		return "", "", "", fmt.Errorf("could not parse the request line, got fields %v", fields)
	}
	return fields[0], fields[1], fields[2], nil
}

func validMethod(method string, proto string) bool {
	return method == "GET" && proto == "HTTP/1.1"
}

func badStringError(what, val string) error {
	return fmt.Errorf("%s %q", what, val)
}
