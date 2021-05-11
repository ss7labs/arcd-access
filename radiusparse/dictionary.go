package radiusparse

import (
	"errors"
	"sync"
)

var builtinOnce sync.Once

// Builtin is the built-in dictionary. It is initially loaded with the
// attributes defined in RFC 2865 and RFC 2866.
var Builtin *Dictionary

func initDictionary() {
	Builtin = &Dictionary{}
}

type Dictionary struct {
	mu               sync.RWMutex
	attributesByType [256]*dictEntry
	attributesByName map[string]*dictEntry
}
type dictEntry struct {
	Type  byte
	Name  string
	Codec AttributeCodec
}

// Register registers the AttributeCodec for the given attribute name and type.
func (d *Dictionary) Register(name string, t byte, codec AttributeCodec) error {
	d.mu.Lock()
	if d.attributesByType[t] != nil {
		d.mu.Unlock()
		return errors.New("radius: attribute already registered")
	}
	entry := &dictEntry{
		Type:  t,
		Name:  name,
		Codec: codec,
	}
	d.attributesByType[t] = entry
	if d.attributesByName == nil {
		d.attributesByName = make(map[string]*dictEntry)
	}
	d.attributesByName[name] = entry
	d.mu.Unlock()
	return nil
}

// MustRegister is a helper for Register that panics if it returns an error.
func (d *Dictionary) MustRegister(name string, t byte, codec AttributeCodec) {
	if err := d.Register(name, t, codec); err != nil {
		panic(err)
	}
}

// Codec returns the AttributeCodec for the given registered type. nil is
// returned if the given type is not registered.
func (d *Dictionary) Codec(t byte) AttributeCodec {
	d.mu.RLock()
	entry := d.attributesByType[t]
	d.mu.RUnlock()
	if entry == nil {
		return AttributeUnknown
	}
	return entry.Codec
}

// Name returns the registered name for the given attribute type. ok is false
// if the given type is not registered.
func (d *Dictionary) Name(t byte) (name string, ok bool) {
	d.mu.RLock()
	entry := d.attributesByType[t]
	d.mu.RUnlock()
	if entry == nil {
		return
	}
	name = entry.Name
	ok = true
	return
}
