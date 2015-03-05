package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/V1/SHOUT", ProShout).Methods("POST")
	http.Handle("/", r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Printf("LISTENING ON %v", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type ShoutRequest struct {
	Input  string `json:"INPUT"`
	Output string `json:"OUTPUT"`
	Error  error
}

func (s *ShoutRequest) Process() {
	// call regular shoutcloud API for first pass inanity
	var jsonOut []byte
	jsonOut, s.Error = json.Marshal(s)
	if s.Error != nil {
		return
	}
	var req *http.Request
	req, s.Error = http.NewRequest("POST", "http://api.shoutcloud.io/V1/SHOUT", bytes.NewBuffer(jsonOut))
	req.Header.Set("CONTENT-TYPE", "APPLICATION/JSON")

	var resp *http.Response
	resp, s.Error = http.DefaultClient.Do(req)

	// decode api response
	if resp.StatusCode != 200 {
		var upstreamErr error
		body, upstreamErr := ioutil.ReadAll(resp.Body)
		if upstreamErr == nil {
			defer resp.Body.Close()
			errorDesc := fmt.Sprintf("UPSTREAM SHOUT ERROR: %s", body)
			upstreamErr = errors.New(errorDesc)
		}
		s.Error = upstreamErr
		return
	}

	decoder := json.NewDecoder(resp.Body)
	var shout ShoutRequest
	s.Error = decoder.Decode(&shout)
	if s.Error != nil {
		return
	}

	// gsub ? with ‽ and . with !
	proOutput := strings.Replace(shout.Output, "?", "‽", -1)
	proOutput = strings.Replace(proOutput, ".", "!", -1)

	s.Output = proOutput
}

func ProShout(w http.ResponseWriter, r *http.Request) {
	// get API key out of auth header
	// verify API key on gumroad

	// process shoutrequest
	log.Printf("POST /V1/SHOUT %v", r.RemoteAddr)

	if !(r.Header.Get("Content-Type") == "application/json" ||
		r.Header.Get("Content-Type") == "APPLICATION/JSON" ||
		r.Header.Get("CONTENT-TYPE") == "application/json" ||
		r.Header.Get("CONTENT-TYPE") == "APPLICATION/JSON") {

		http.Error(w, "BAD CONTENT-TYPE REQUEST", http.StatusBadRequest)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var shout ShoutRequest
	err := decoder.Decode(&shout)
	if err != nil {
		log.Printf("ERROR JSON DECODING: %v", r.Body)
		http.Error(w, "BAD JSON REQUEST", http.StatusBadRequest)
		return
	}

	shout.Process()
	if shout.Error != nil {
		http.Error(w, shout.Error.Error(), http.StatusInternalServerError)
		return
	}

	json, err := json.Marshal(shout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("CONTENT-TYPE", "APPLICATION/JSON")
	w.Write(json)
}