package tritonhttp

// approximate contribution: 127 in server.go
import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// DocRoot specifies the path to the directory to serve static files from.
	DocRoot string
}

const (
	responseProto = "HTTP/1.1"

	statusOK         = 200
	statusBadRequest = 400
	statusNotFound   = 404
)

var statusText = map[int]string{
	statusOK:         "OK",
	statusBadRequest: "Bad Request",
	statusNotFound:   "Not Found",
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.
func (s *Server) ListenAndServe() error {
	// @credit: Week4 TA Section

	// Hint: call HandleConnection
	if s.ValidateServerSetup() != nil {
		return fmt.Errorf("server is not setup correctly %v", s.ValidateServerSetup())
	}
	fmt.Println("Server setup valid!")

	// start listening
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	fmt.Println("Listening on", ln.Addr())

	// close after exiting
	defer func() {
		err = ln.Close()
		if err != nil {
			fmt.Printf("Closing listener failed: %v", err)
		}
	}()

	// accept connection
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		fmt.Println("Accept connection from ", conn.RemoteAddr())
		go s.HandleConnection(conn)
	}
}

func (s *Server) ValidateServerSetup() error {
	// @credit: Week4 TA Section

	// Validating the doc root of the server
	fi, err := os.Stat(s.DocRoot)

	if os.IsNotExist(err) {
		return err
	}

	if !fi.IsDir() {
		return fmt.Errorf("doc root %q is not a directory", s.DocRoot)
	}

	return nil
}

// HandleConnection reads requests from the accepted conn and handles them.
func (s *Server) HandleConnection(conn net.Conn) {
	// @credit: Week4 TA Section
	// Hint: use the other methods below

	br := bufio.NewReader(conn)
	for {
		// Set timeout
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
			log.Printf("Failed to set timeout for connection %v", conn)
			_ = conn.Close()
			return
		}

		// Try to read next request
		req, bytesReceived, err := ReadRequest(br)

		// Handle EOF
		if errors.Is(err, io.EOF) {
			fmt.Printf("Connection closed by the client %v", conn.RemoteAddr())
			if bytesReceived {
				res := &Response{}
				res.HandleBadRequest()
				_ = res.Write(conn)
			}

			_ = conn.Close()
			return
		}

		// Handle timeout
		// need more manipulation
		if err, ok := err.(net.Error); ok && err.Timeout() {
			fmt.Printf("Connection to %s timed out", conn.RemoteAddr())
			if bytesReceived {
				res := &Response{}
				res.HandleBadRequest()
				_ = res.Write(conn)
			}
			_ = conn.Close()
			return
		}

		// Handle bad request
		if (err != nil && bytesReceived) || req.URL[0] != '/' {
			fmt.Printf("Handle bad request for error: %v", conn.RemoteAddr())
			res := &Response{}
			res.HandleBadRequest()
			_ = res.Write(conn)
			_ = conn.Close()
			return
		}

		// Handle good request
		fmt.Printf("Handle good request: %v", req)
		res := s.HandleGoodRequest(req)
		err = res.Write(conn)
		if err != nil {
			fmt.Println(err)
		}

		// Close conn if requested
		if req.Close {
			conn.Close()
		}
	}
}

// HandleGoodRequest handles the valid req and generates the corresponding res.
func (s *Server) HandleGoodRequest(req *Request) (res *Response) {
	res = &Response{}
	// fmt.Println(req.URL)
	if strings.HasSuffix(req.URL, "/") {
		req.URL += "index.html"
	}
	respath := filepath.Clean(filepath.Join(s.DocRoot, req.URL))
	// fmt.Println("*******PATH*********", respath)

	_, err := os.Stat(respath)
	// fmt.Println("****************", err)
	if err != nil {
		// fmt.Println("*******ERR*********", os.IsNotExist(err))
		if os.IsNotExist(err) {
			res.HandleNotFound(req)
			return
		}
	}
	res.HandleOK(req, respath)
	fmt.Println("*******HANDLEOK*********")
	return res
	// Hint: use the other methods below
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, fpath string) {
	res.StatusCode = statusOK
	res.FilePath = fpath
	res.Proto = responseProto
	res.Header = make(map[string]string)
	res.Header["Date"] = FormatTime(time.Now())
	if req.Close {
		res.Header["Connection"] = "close"
	}

	fistatus, err := os.Stat(fpath)

	if os.IsNotExist(err) {
		fmt.Println("Get file info failed")
	}

	res.Header["Last-Modified"] = FormatTime(fistatus.ModTime())
	res.Header["Content-Type"] = MIMETypeByExtension(path.Ext(req.URL))
	res.Header["Content-Length"] = strconv.FormatInt(fistatus.Size(), 10)
}

// HandleBadRequest prepares res to be a 400 Bad Request response
// ready to be written back to client.
func (res *Response) HandleBadRequest() {
	res.StatusCode = statusBadRequest
	res.Header = make(map[string]string)
	res.Proto = responseProto
	res.FilePath = ""
	res.Header["Connection"] = "close"
	res.Header["Date"] = FormatTime(time.Now())

}

// HandleNotFound prepares res to be a 404 Not Found response
// ready to be written back to client.
func (res *Response) HandleNotFound(req *Request) {
	// panic("todo")
	res.Proto = responseProto
	res.StatusCode = statusNotFound
	res.Header = make(map[string]string)
	res.Header["Date"] = FormatTime(time.Now())
	if req.Close {
		res.Header["Connection"] = "close"
	}
}
