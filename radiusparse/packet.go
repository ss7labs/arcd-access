package radiusparse

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
)

// maximum RADIUS packet size
const maxPacketSize = 4095

// Code specifies the kind of RADIUS packet
type Code byte

// Codes which are defined in RFC 2865
const (
	CodeAccessRequest Code = 1
	CodeAccessAccept  Code = 2
	CodeAccessReject  Code = 3

	CodeAccountingRequest  Code = 4
	CodeAccountingResponse Code = 5

	CodeAccessChallenge Code = 11

	CodeStatusServer Code = 12
	CodeStatusClient Code = 13

	CodeDisconnectRequest Code = 40
	CodeDisconnectACK     Code = 41
	CodeDisconnectNAK     Code = 42

	CodeCoARequest Code = 43
	CodeCoAACK     Code = 44
	CodeCoANAK     Code = 45

	CodeReserved Code = 255
)

// Packet defines a RADIUS packet.
type Packet struct {
	Code          Code
	Identifier    byte
	Authenticator [16]byte
	Secret        []byte

	Raw        *[]byte
	Dictionary *Dictionary
	Attributes []*Attribute
}

// Parse parses a RADIUS packet from wire data, using the given shared secret
// and dictionary. nil and an error is returned if there is a problem parsing
// the packet.
//
// Note: this function does not validate the authenticity of a packet.
// Ensuring a packet's authenticity should be done using the IsAuthentic
// method.
func Parse(data, secret []byte, dictionary *Dictionary) (*Packet, error) {
	if len(data) < 20 {
		return nil, errors.New("radius: packet must be at least 20 bytes long")
	}

	packet := &Packet{
		Code:       Code(data[0]),
		Raw:        &data,
		Identifier: data[1],
		Secret:     secret,
		Dictionary: dictionary,
	}

	length := binary.BigEndian.Uint16(data[2:4])
	if length < 20 || length > maxPacketSize {
		return nil, errors.New("radius: invalid packet length")
	}

	copy(packet.Authenticator[:], data[4:20])

	// Attributes
	attributes := data[20:]
	for len(attributes) > 0 {
		if len(attributes) < 2 {
			return nil, errors.New("radius: attribute must be at least 2 bytes long")
		}

		attrLength := attributes[1]
		if attrLength < 1 || attrLength > 253 || len(attributes) < int(attrLength) {
			return nil, errors.New("radius: invalid attribute length")
		}

		attrType := attributes[0]
		attrValue := attributes[2:attrLength]

		codec := dictionary.Codec(attrType)
		decoded, err := codec.Decode(packet, attrValue)
		if err != nil {
			return nil, err
		}

		attr := &Attribute{
			Type:  attrType,
			Value: decoded,
		}

		packet.Attributes = append(packet.Attributes, attr)
		attributes = attributes[attrLength:]
	}

	// TODO: validate that the given packet (by code) has all the required attributes, etc.
	return packet, nil
}

// Values returns a slice of all attributes' values with given name
func (p *Packet) Values(name string) (values []interface{}) {
	for _, attr := range p.Attributes {
		if attrName, ok := p.Dictionary.Name(attr.Type); ok && attrName == name {
			values = append(values, attr.Value)
		}
	}
	return
}

// Value returns the value of the first attribute whose dictionary name matches
// the given name. nil is returned if no such attribute exists.
func (p *Packet) Value(name string) interface{} {
	if attr := p.Attr(name); attr != nil {
		return attr.Value
	}
	return nil
}

// Attr returns the first attribute whose dictionary name matches the given
// name. nil is returned if no such attribute exists.
func (p *Packet) Attr(name string) *Attribute {
	for _, attr := range p.Attributes {
		if attrName, ok := p.Dictionary.Name(attr.Type); ok && attrName == name {
			return attr
		}
	}
	return nil
}

// String returns the string representation of the value of the first attribute
// whose dictionary name matches the given name. The following rules are used
// for converting the attribute value to a string:
//
//  - If no such attribute exists with the given dictionary name, "" is
//    returned
//  - If the attribute's Codec implements AttributeStringer,
//    AttributeStringer.String(value) is returned
//  - If the value implements fmt.Stringer, value.String() is returned
//  - If the value is string, itself is returned
//  - If the value is []byte, string(value) is returned
//  - Otherwise, "" is returned
func (p *Packet) String(name string) string {
	attr := p.Attr(name)
	if attr == nil {
		return ""
	}
	value := attr.Value

	if codec := p.Dictionary.Codec(attr.Type); codec != nil {
		if stringer, ok := codec.(AttributeStringer); ok {
			return stringer.String(value)
		}
	}

	if stringer, ok := value.(interface {
		String() string
	}); ok {
		return stringer.String()
	}

	if str, ok := value.(string); ok {
		return str
	}

	if raw, ok := value.([]byte); ok {
		return string(raw)
	}
	return ""
}

// Encode encodes the packet to wire format. If there is an error encoding the
// packet, nil and an error is returned.
func (p *Packet) Encode() ([]byte, error) {
	var bufferAttrs bytes.Buffer

	for _, attr := range p.Attributes {
		codec := p.Dictionary.Codec(attr.Type)
		wire, err := codec.Encode(p, attr.Value)
		if err != nil {
			return nil, err
		}

		if len(wire) > 253 {
			return nil, errors.New("radius: encoded attribute is too long")
		}

		bufferAttrs.WriteByte(attr.Type)
		bufferAttrs.WriteByte(byte(len(wire) + 2))
		bufferAttrs.Write(wire)
	}

	length := 20 + bufferAttrs.Len()
	if length > maxPacketSize {
		return nil, errors.New("radius: encoded packet is too long")
	}

	var buffer bytes.Buffer
	buffer.Grow(length)
	buffer.WriteByte(byte(p.Code))
	buffer.WriteByte(p.Identifier)
	binary.Write(&buffer, binary.BigEndian, uint16(length))

	switch p.Code {
	case CodeAccessRequest, CodeStatusServer:
		buffer.Write(p.Authenticator[:])
		break

	case CodeCoARequest, CodeDisconnectRequest, CodeAccessAccept, CodeAccessReject, CodeAccountingRequest, CodeAccountingResponse, CodeAccessChallenge, CodeCoAACK, CodeCoANAK, CodeDisconnectACK, CodeDisconnectNAK:
		hash := md5.New()
		hash.Write(buffer.Bytes())

		switch p.Code {
		case CodeAccountingRequest, CodeCoARequest, CodeDisconnectRequest:
			var nul [16]byte
			hash.Write(nul[:])
			break

		default:
			hash.Write(p.Authenticator[:])
			break
		}

		hash.Write(bufferAttrs.Bytes())
		hash.Write(p.Secret)

		var sum [16]byte
		buffer.Write(hash.Sum(sum[0:0]))

		// We overwrite the original authenticator because it will be used in IsAuthentic() to authenticate a reply
		switch p.Code {
		case CodeCoARequest, CodeDisconnectRequest:
			copy(p.Authenticator[:], sum[:])
			break
		}

		break

	default:
		return nil, errors.New("radius: unknown Packet code")
	}

	buffer.ReadFrom(&bufferAttrs)

	return buffer.Bytes(), nil
}
