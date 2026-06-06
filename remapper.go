package evdev

import (
	"errors"
	"io"
	"sync"
)

// MapFunc transforms a source event into the events to emit in its place.
// Returning nil or an empty slice drops the event (suppressing it); returning
// several events expands one input into many (a macro or key combo). Include
// EV_SYN/SYN_REPORT events in the result to split a macro into separate input
// frames.
//
// EV_SYN frame markers from the source are forwarded automatically and are not
// passed to MapFunc, so a plain passthrough mapping is simply:
//
//	func(ev InputEvent) []InputEvent { return []InputEvent{ev} }
type MapFunc func(InputEvent) []InputEvent

// Remapper grabs a source device exclusively and re-emits its events — as
// transformed by a MapFunc — through a uinput virtual device. It packages the
// grab -> read -> transform -> inject loop with correct setup and teardown, so a
// client only has to express the mapping.
type Remapper struct {
	src *Device
	out *VirtualDevice
	fn  MapFunc

	closeOnce sync.Once
	closeErr  error
}

// RemapOption configures a Remapper at construction.
type RemapOption func(*remapOptions)

type remapOptions struct {
	name  string
	extra Capabilities
}

// WithName sets the virtual device's name (default "go-evdev remapper").
func WithName(name string) RemapOption {
	return func(o *remapOptions) { o.name = name }
}

// WithExtraKeys registers EV_KEY codes on the virtual device beyond those the
// source supports — needed when a mapping emits keys the source lacks, e.g. a
// mouse button that types a letter or a Ctrl+C combo.
func WithExtraKeys(keys ...EvCode) RemapOption {
	return func(o *remapOptions) { o.extra.Keys = append(o.extra.Keys, keys...) }
}

// WithExtraCapabilities registers additional capabilities (relative axes, misc
// codes, properties) on the virtual device, merged with the source's.
func WithExtraCapabilities(c Capabilities) RemapOption {
	return func(o *remapOptions) { o.extra = mergeCaps(o.extra, c) }
}

// NewRemapper grabs src exclusively and builds a virtual device mirroring its
// capabilities (plus any added via options), ready to re-emit events through fn.
// Call Run to process events and Close to release the grab and destroy the
// virtual device.
//
// The caller retains ownership of src and must Close it separately; closing src
// is also how a blocked Run is unblocked (see Run).
func NewRemapper(src *Device, fn MapFunc, opts ...RemapOption) (*Remapper, error) {
	o := remapOptions{name: "go-evdev remapper"}
	for _, opt := range opts {
		opt(&o)
	}

	caps, err := CapabilitiesOf(src)
	if err != nil {
		return nil, err
	}
	caps = mergeCaps(caps, o.extra)

	id, err := src.ID()
	if err != nil {
		return nil, err
	}
	out, err := CreateVirtualDevice(o.name, id, caps)
	if err != nil {
		return nil, err
	}
	if err := src.Grab(); err != nil {
		out.Close()
		return nil, err
	}
	return &Remapper{src: src, out: out, fn: fn}, nil
}

// Output returns the virtual device that events are emitted through, for callers
// that want to inject additional events directly (e.g. a timed macro driven from
// another goroutine).
func (r *Remapper) Output() *VirtualDevice { return r.out }

// Run reads, transforms, and re-emits events until the source returns io.EOF
// (returning nil) or another error. It blocks, so run it in its own goroutine if
// the caller needs to do other work; a slow MapFunc backpressures the source. To
// stop a running Run, Close the source device so its ReadOne unblocks.
func (r *Remapper) Run() error {
	for {
		ev, err := r.src.ReadOne()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		// Forward frame markers verbatim; map only real events.
		if ev.Type == EV_SYN {
			if err := r.out.Write(ev); err != nil {
				return err
			}
			continue
		}
		for _, e := range r.fn(ev) {
			if err := r.out.Write(e); err != nil {
				return err
			}
		}
	}
}

// Close releases the source grab and destroys the virtual device. It does not
// close the source device, which the caller owns. It is safe to call more than
// once.
func (r *Remapper) Close() error {
	r.closeOnce.Do(func() {
		r.closeErr = firstErr(r.src.Ungrab(), r.out.Close())
	})
	return r.closeErr
}

// mergeCaps returns the union of two capability sets.
func mergeCaps(a, b Capabilities) Capabilities {
	return Capabilities{
		Keys:  append(a.Keys, b.Keys...),
		Rels:  append(a.Rels, b.Rels...),
		Mscs:  append(a.Mscs, b.Mscs...),
		Props: append(a.Props, b.Props...),
	}
}
