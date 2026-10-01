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
	"fmt"
)

// Dispatcher routes a SOAP request to the endpoint registered for its payload
// root element — the analog of Spring WS's endpoint mapping + marshalling
// endpoint adapter. Register typed endpoints with [Register], then feed raw
// SOAP via [Dispatcher.Handle] (or mount [Dispatcher.Server] on net/http).
type Dispatcher struct {
	routes map[string]route
}

type route struct {
	invoke func(ctx context.Context, inner []byte) (any, error)
}

// NewDispatcher returns an empty dispatcher.
func NewDispatcher() *Dispatcher { return &Dispatcher{routes: map[string]route{}} }

// Register maps a payload root local name to a typed handler, the analog of an
// @Endpoint method annotated @PayloadRoot(localPart=rootElement). The request
// payload is unmarshalled into Req; the returned Resp is marshalled into the
// SOAP response body.
func Register[Req any, Resp any](d *Dispatcher, rootElement string, h func(ctx context.Context, req Req) (Resp, error)) {
	d.routes[rootElement] = route{
		invoke: func(ctx context.Context, inner []byte) (any, error) {
			var req Req
			if err := xml.Unmarshal(inner, &req); err != nil {
				return nil, Fault{Code: "soap:Client", String: fmt.Sprintf("cannot parse %s: %v", rootElement, err)}
			}
			return h(ctx, req)
		},
	}
}

// Handle processes one raw SOAP request and returns the raw SOAP response. A
// handler error is rendered as a SOAP Fault (returned as the response bytes,
// with a non-nil error so the transport can set HTTP 500).
func (d *Dispatcher) Handle(ctx context.Context, soapRequest []byte) (soapResponse []byte, err error) {
	inner, root, err := parseRequest(soapRequest)
	if err != nil {
		return d.fault(Fault{Code: "soap:Client", String: err.Error()}), err
	}
	r, ok := d.routes[root]
	if !ok {
		f := Fault{Code: "soap:Client", String: "no endpoint for payload root " + root}
		return d.fault(f), f
	}
	resp, err := r.invoke(ctx, inner)
	if err != nil {
		f, ok := err.(Fault)
		if !ok {
			f = Fault{Code: "soap:Server", String: err.Error()}
		}
		return d.fault(f), f
	}
	out, err := wrapResponse(resp)
	if err != nil {
		f := Fault{Code: "soap:Server", String: err.Error()}
		return d.fault(f), f
	}
	return out, nil
}

// fault wraps a Fault in a full SOAP envelope.
func (d *Dispatcher) fault(f Fault) []byte { return buildEnvelope(f.faultXML()) }
