package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

const (
	QueryParamRegex           = "regex"
	QueryParamSearch          = "search"
	QueryParamCaseInsensitive = "ci"
)

var (
	httpPort    int
	httpAddress string
)

//go:embed index.html
var webUiIndex []byte

func init() {
	flag.IntVar(&httpPort, "port", 8080, "http server port")
	flag.StringVar(&httpAddress, "address", "", "http server address")
}

// trimSliceString removes the first and last elements of the supplied slice if they are empty strings
func trimSliceString(s []string) []string {
	if s == nil {
		return nil
	}
	if len(s) == 0 {
		return s
	}
	newSlice := slices.Clone(s)
	// start of slice
	if newSlice[0] == "" {
		if len(newSlice) == 1 {
			return make([]string, 0)
		}
		newSlice = newSlice[1:]
	}
	// end of slice
	if newSlice[len(newSlice)-1] == "" {
		if len(newSlice) == 1 {
			return make([]string, 0)
		}
		newSlice = newSlice[:len(newSlice)-1]
	}
	return newSlice
}

// runLocate executes the `locate` command
func runLocate(search *string, regex *regexp.Regexp, caseInsensitive bool) []string {
	locateCmd := exec.Command("locate")
	locateCmd.Args = []string{"--null"}
	if caseInsensitive {
		locateCmd.Args = append(locateCmd.Args, "--ignore-case")
	}
	if search != nil {
		locateCmd.Args = append(locateCmd.Args, "--", *search)
	} else if regex != nil {
		locateCmd.Args = append(locateCmd.Args, "--regex", regex.String())
	} else {
		return nil
	}
	log.Printf("command: locate %v\n", locateCmd.Args)
	locateStdout, err := locateCmd.Output()
	if err != nil {
		exitErr := new(exec.ExitError)
		if errors.As(err, &exitErr) {
			log.Printf("Failed to execute locate command: %v\nStderr: %s", err, exitErr.Stderr)
			return nil
		}
		log.Printf("Unexpected error when executing locate command: %v\n", err)
		return nil
	}
	results := strings.Split(string(locateStdout), "\n")
	return trimSliceString(results)
}

// handler is the method that handles HTTP requests to the /locate URL
func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s\n", r.RemoteAddr, r.URL)
	query := r.URL.Query()
	// parse query parameters
	caseInsensitive := false
	if query.Has(QueryParamCaseInsensitive) && query.Get(QueryParamCaseInsensitive) == "true" {
		caseInsensitive = true
	}
	searchPattern := ""
	regexPattern := ""
	if query.Has(QueryParamSearch) {
		searchPattern = strings.TrimSpace(query.Get(QueryParamSearch))
	} else if query.Has(QueryParamRegex) {
		regexPattern = strings.TrimSpace(query.Get(QueryParamRegex))
	}
	// perform search
	var results []string
	if searchPattern != "" {
		log.Printf("[%s] searchPattern=%s, caseInsensitive=%t\n", r.RemoteAddr, searchPattern, caseInsensitive)
		results = runLocate(&searchPattern, nil, caseInsensitive)
	} else if regexPattern != "" {
		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		log.Printf("[%s] regex=%s, caseInsensitive=%t\n", r.RemoteAddr, r.URL, caseInsensitive)
		results = runLocate(nil, regex, caseInsensitive)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if results == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// marshal and return results
	log.Printf("[%s] %d results\n", r.RemoteAddr, len(results))
	resultsJson, err := json.Marshal(results)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resultsJson)
}

// webUiHandler serves the web UI content
func webUiHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write(webUiIndex)
}

func main() {
	flag.Parse()
	http.HandleFunc("/", webUiHandler)
	http.HandleFunc("/locate", handler)
	httpAddr := fmt.Sprintf("%s:%d", httpAddress, httpPort)
	log.Println("Listening on", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}
