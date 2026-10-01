/*
 * Copyright 2025 The Go-Spring Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ws

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type GetGreetingRequest struct {
	XMLName xml.Name `xml:"getGreetingRequest"`
	Name    string   `xml:"name"`
}

type GetGreetingResponse struct {
	XMLName xml.Name `xml:"getGreetingResponse"`
	Message string   `xml:"message"`
}

func dispatcher() *Dispatcher {
	d := NewDispatcher()
	Register(d, "getGreetingRequest", func(_ context.Context, req GetGreetingRequest) (GetGreetingResponse, error) {
		if req.Name == "" {
			return GetGreetingResponse{}, Fault{Code: "soap:Client", String: "name is required"}
		}
		return GetGreetingResponse{Message: "Hello, " + req.Name}, nil
	})
	return d
}

const greetingReq = `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getGreetingRequest><name>Ada</name></getGreetingRequest>
  </soap:Body>
</soap:Envelope>`

func TestDispatchSuccess(t *testing.T) {
	resp, err := dispatcher().Handle(context.Background(), []byte(greetingReq))
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	s := string(resp)
	if !strings.Contains(s, "<message>Hello, Ada</message>") {
		t.Fatalf("response payload missing:\n%s", s)
	}
	if !strings.Contains(s, "soap:Envelope") || !strings.Contains(s, "soap:Body") {
		t.Fatalf("response not wrapped in envelope:\n%s", s)
	}
}

func TestDispatchFaultFromHandler(t *testing.T) {
	req := strings.Replace(greetingReq, "<name>Ada</name>", "<name></name>", 1)
	resp, err := dispatcher().Handle(context.Background(), []byte(req))
	if err == nil {
		t.Fatal("expected fault error")
	}
	if !strings.Contains(string(resp), "<faultstring>name is required</faultstring>") {
		t.Fatalf("fault not rendered:\n%s", resp)
	}
}

func TestUnknownPayloadRoot(t *testing.T) {
	req := strings.Replace(greetingReq, "getGreetingRequest", "unknownOp", 2)
	_, err := dispatcher().Handle(context.Background(), []byte(req))
	var f Fault
	if !errors.As(err, &f) || !strings.Contains(f.String, "no endpoint") {
		t.Fatalf("expected no-endpoint fault, got %v", err)
	}
}

func TestServerHTTP(t *testing.T) {
	srv := httptest.NewServer(dispatcher().Server())
	defer srv.Close()

	// Success.
	resp, err := http.Post(srv.URL, "text/xml", strings.NewReader(greetingReq))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/xml") {
		t.Fatalf("content-type = %q", ct)
	}

	// Fault → HTTP 500.
	bad := strings.Replace(greetingReq, "<name>Ada</name>", "<name></name>", 1)
	resp, _ = http.Post(srv.URL, "text/xml", strings.NewReader(bad))
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("fault status = %d, want 500", resp.StatusCode)
	}

	// GET not allowed.
	resp, _ = http.Get(srv.URL)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", resp.StatusCode)
	}
}
