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

// Package ws is the GoSpring port of Spring Web Services: a contract-first SOAP
// endpoint model. Requests are routed by the local name of their payload root
// element (the @PayloadRoot analog) to a typed handler, with the SOAP 1.1
// envelope and Fault handling provided by the package. Marshalling is plain
// encoding/xml, so request/response types are ordinary structs with xml tags.
package ws

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// soapNS is the SOAP 1.1 envelope namespace.
const soapNS = "http://schemas.xmlsoap.org/soap/envelope/"

// envelope is the SOAP 1.1 envelope used to parse requests and render responses.
type envelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    body     `xml:"Body"`
}

type body struct {
	// Inner captures the raw payload XML so it can be inspected for its root
	// element and then unmarshalled into the handler's request type.
	Inner []byte `xml:",innerxml"`
}

// payloadRoot returns the local name of the first element inside the SOAP Body
// — the key Spring's PayloadRootAnnotationMethodEndpointMapping dispatches on.
func payloadRoot(inner []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(inner))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("ws: no payload root element: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}

// parseRequest extracts the raw payload XML from a SOAP request envelope.
func parseRequest(soap []byte) (inner []byte, root string, err error) {
	var e envelope
	if err := xml.Unmarshal(soap, &e); err != nil {
		return nil, "", fmt.Errorf("ws: malformed SOAP envelope: %w", err)
	}
	root, err = payloadRoot(e.Body.Inner)
	if err != nil {
		return nil, "", err
	}
	return e.Body.Inner, root, nil
}

// wrapResponse marshals payload and wraps it in a SOAP response envelope.
func wrapResponse(payload any) ([]byte, error) {
	inner, err := xml.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ws: marshal response payload: %w", err)
	}
	return buildEnvelope(inner), nil
}

// buildEnvelope assembles a SOAP 1.1 envelope around already-marshalled body XML.
func buildEnvelope(innerBody []byte) []byte {
	var b bytes.Buffer
	b.WriteString(xml.Header)
	fmt.Fprintf(&b, `<soap:Envelope xmlns:soap="%s"><soap:Body>`, soapNS)
	b.Write(innerBody)
	b.WriteString(`</soap:Body></soap:Envelope>`)
	return b.Bytes()
}

// Fault is a SOAP 1.1 fault, the analog of a SoapFault. It is both an error and
// the body rendered when a handler fails.
type Fault struct {
	Code   string // faultcode, e.g. "soap:Server"
	String string // faultstring, a human message
}

// Error implements error.
func (f Fault) Error() string { return f.Code + ": " + f.String }

// faultXML renders the fault element (inside a Body).
func (f Fault) faultXML() []byte {
	code := f.Code
	if code == "" {
		code = "soap:Server"
	}
	return fmt.Appendf(nil,
		`<soap:Fault><faultcode>%s</faultcode><faultstring>%s</faultstring></soap:Fault>`,
		xmlEscape(code), xmlEscape(f.String))
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
