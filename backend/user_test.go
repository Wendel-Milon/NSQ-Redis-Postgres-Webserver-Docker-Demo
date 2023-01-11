package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

var server Server

func init() {
	pgconn, err := ConnectPostgre()
	if err != nil {
		log.Fatalln(err)
	}

	mux := CreateRouter()

	server = Server{
		pg:  pgconn,
		mux: mux,
	}

	AttachAllPaths(&server)
}

// executeRequest, creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
func executeRequest(req *http.Request, s *Server) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.mux.ServeHTTP(rr, req)

	return rr
}

// checkResponseCode is a simple utility to check the response code
// of the response
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func readResponse(buffer *bytes.Buffer) string {
	bodyBytes, err := io.ReadAll(buffer)
	if err != nil {
		log.Fatal(err)
	}
	return string(bodyBytes)
}

func TestCreateUserGet(t *testing.T) {

	req, _ := http.NewRequest("GET", "/create", nil)
	resp := executeRequest(req, &server)

	checkResponseCode(t, http.StatusOK, resp.Code)
}

func TestCreateUserPost(t *testing.T) {

	// Empty Form
	req, _ := http.NewRequest("POST", "/create", nil)
	resp := executeRequest(req, &server)
	checkResponseCode(t, http.StatusBadRequest, resp.Code)

	// Only userid
	form := url.Values{}
	form.Add("userid", "test")

	req, _ = http.NewRequest("POST", "/create", nil)
	req.PostForm = form
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp = executeRequest(req, &server)
	checkResponseCode(t, http.StatusBadRequest, resp.Code)
	str := readResponse(resp.Body)
	if str != ErrNoUserID.Error() {
		t.Errorf("Body not correct! got: %s, want: %s", str, ErrNoUserID)
	}

	form.Add("passwd", "test")
	req.PostForm = form
	resp = executeRequest(req, &server)
	checkResponseCode(t, http.StatusBadRequest, resp.Code)
	str = readResponse(resp.Body)
	if str != ErrNoPassWd.Error() {
		t.Errorf("Body not correct! got: %s, want: %s", str, ErrNoPassWd)
	}

}
