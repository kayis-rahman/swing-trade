// Package parity compares read-only API responses from the Java and Go services.
package parity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// Request describes one deterministic read request to send to both services.
type Request struct {
	Method string
	Path   string
}

// Difference identifies a mismatch between two API responses.
type Difference struct {
	Path   string
	Reason string
}

// Compare sends the same requests to both services and returns every mismatch.
// Each service receives requests sequentially in manifest order.
func Compare(client *http.Client, javaBaseURL, goBaseURL string, requests []Request) ([]Difference, error) {
	if client == nil {
		client = http.DefaultClient
	}
	var differences []Difference
	for _, request := range requests {
		method := request.Method
		if method == "" {
			method = http.MethodGet
		}
		javaResponse, err := fetch(client, javaBaseURL, method, request.Path)
		if err != nil {
			return differences, fmt.Errorf("request Java %s %s: %w", method, request.Path, err)
		}
		goResponse, err := fetch(client, goBaseURL, method, request.Path)
		if err != nil {
			return differences, fmt.Errorf("request Go %s %s: %w", method, request.Path, err)
		}
		if difference := compareResponses(request.Path, javaResponse, goResponse); difference != nil {
			differences = append(differences, *difference)
		}
	}
	return differences, nil
}

type response struct {
	status      int
	contentType string
	body        []byte
}

func fetch(client *http.Client, baseURL, method, path string) (response, error) {
	// #nosec G704 -- bases and paths are supplied by the local parity-test operator.
	request, err := http.NewRequest(method, strings.TrimRight(baseURL, "/")+path, nil)
	if err != nil {
		return response{}, err
	}
	// #nosec G704 -- this intentionally sends requests to the operator-selected parity services.
	result, err := client.Do(request)
	if err != nil {
		return response{}, err
	}
	body, readErr := io.ReadAll(result.Body)
	closeErr := result.Body.Close()
	if readErr != nil {
		return response{}, readErr
	}
	if closeErr != nil {
		return response{}, closeErr
	}
	return response{status: result.StatusCode, contentType: result.Header.Get("Content-Type"), body: body}, nil
}

func compareResponses(path string, java, goResponse response) *Difference {
	if java.status != goResponse.status {
		return &Difference{Path: path, Reason: fmt.Sprintf("status Java=%d Go=%d", java.status, goResponse.status)}
	}
	if java.contentType != goResponse.contentType {
		return &Difference{Path: path, Reason: fmt.Sprintf("content-type Java=%q Go=%q", java.contentType, goResponse.contentType)}
	}
	var equal bool
	if json.Valid(java.body) && json.Valid(goResponse.body) {
		equal = equalJSON(path, java.body, goResponse.body)
	} else {
		equal = bytes.Equal(java.body, goResponse.body)
	}
	if !equal {
		return &Difference{Path: path, Reason: "response bodies differ"}
	}
	return nil
}

func equalJSON(path string, java, goBody []byte) bool {
	var javaValue, goValue any
	javaDecoder := json.NewDecoder(bytes.NewReader(java))
	javaDecoder.UseNumber()
	goDecoder := json.NewDecoder(bytes.NewReader(goBody))
	goDecoder.UseNumber()
	if javaDecoder.Decode(&javaValue) != nil || goDecoder.Decode(&goValue) != nil {
		return false
	}
	if path == "/api/watchlist/all" {
		javaRecords, javaOK := recordsByID(javaValue)
		goRecords, goOK := recordsByID(goValue)
		if !javaOK || !goOK {
			return false
		}
		return equalJSONValue(javaRecords, goRecords)
	}
	return equalJSONValue(javaValue, goValue)
}

func recordsByID(value any) ([]any, bool) {
	envelope, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	records, ok := envelope["data"].([]any)
	if !ok {
		return nil, false
	}
	ids := make(map[int64]struct{}, len(records))
	for _, value := range records {
		record, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		id, ok := record["id"].(json.Number)
		if !ok {
			return nil, false
		}
		parsedID, err := strconv.ParseInt(id.String(), 10, 64)
		if err != nil {
			return nil, false
		}
		if _, duplicate := ids[parsedID]; duplicate {
			return nil, false
		}
		ids[parsedID] = struct{}{}
	}
	sort.Slice(records, func(i, j int) bool {
		left := records[i].(map[string]any)["id"].(json.Number).String()
		right := records[j].(map[string]any)["id"].(json.Number).String()
		leftID, _ := strconv.ParseInt(left, 10, 64)
		rightID, _ := strconv.ParseInt(right, 10, 64)
		return leftID < rightID
	})
	return records, true
}

func equalJSONValue(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}
