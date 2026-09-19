package bus

import (
	"strings"
	"sync"
)

// abbreviationIndex resolves shortened dot-suffixes of a fully-qualified
// name (e.g. "SPORE.node.spawn") back to the fqname that owns them
// ("dev.sporeos.SPORE.node.spawn"), and tracks when a suffix is claimed by
// more than one fqname so callers can reject the ambiguous shorthand
// instead of guessing which one was meant.
type abbreviationIndex struct {
	mu sync.RWMutex
	abbreviations map[string]string  // dotted suffix -> fqname, only while unambiguous
	collisions map[string][]string   // dotted suffix -> contending fqnames, once ambiguous
}

func newAbbreviationIndex() *abbreviationIndex {
	return &abbreviationIndex{
		abbreviations: make(map[string]string),
		collisions: make(map[string][]string),
	}
}

// suffixesOf returns every dot-suffix of a fully-qualified name, from the
// bare trailing segment up to the fqname itself, e.g. for
// "dev.sporeos.SPORE.spawn": [spawn, SPORE.spawn, sporeos.SPORE.spawn, dev.sporeos.SPORE.spawn].
func suffixesOf(fqname string) []string {
	segments := strings.Split(fqname, ".")
	keys := make([]string, 0, len(segments))
	for i := len(segments) - 1; i >= 0; i-- {
		keys = append(keys, strings.Join(segments[i:], "."))
	}
	return keys
}

// add binds every suffix of fqname into the index.
func (idx *abbreviationIndex) add(fqname string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, key := range suffixesOf(fqname) {
		idx.bind(key, fqname)
	}
}

// remove reverses add.
func (idx *abbreviationIndex) remove(fqname string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, key := range suffixesOf(fqname) {
		idx.unbind(key, fqname)
	}
}

// resolve looks up key, returning the unambiguous fqname it maps to. If key
// is currently ambiguous, ok is false and candidates lists the contenders.
func (idx *abbreviationIndex) resolve(key string) (fqname string, candidates []string, ok bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if contenders, found := idx.collisions[key]; found {
		return "", contenders, false
	}
	fqname, found := idx.abbreviations[key]
	return fqname, nil, found
}

// bind maps key to fqname, demoting it to a tracked collision the moment a
// second, different fqname claims the same key.
func (idx *abbreviationIndex) bind(key string, fqname string) {
	if candidates, ok := idx.collisions[key]; ok {
		idx.collisions[key] = append(candidates, fqname)
		return
	}

	if existing, ok := idx.abbreviations[key]; ok {
		if existing == fqname {
			return
		}
		delete(idx.abbreviations, key)
		idx.collisions[key] = []string{existing, fqname}
		return
	}

	idx.abbreviations[key] = fqname
}

// unbind reverses bind, promoting a collision back to a plain abbreviation
// once it is down to a single remaining candidate.
func (idx *abbreviationIndex) unbind(key string, fqname string) {
	if candidates, ok := idx.collisions[key]; ok {
		remaining := candidates[:0]
		for _, candidate := range candidates {
			if candidate != fqname {
				remaining = append(remaining, candidate)
			}
		}
		switch len(remaining) {
		case 0:
			delete(idx.collisions, key)
		case 1:
			delete(idx.collisions, key)
			idx.abbreviations[key] = remaining[0]
		default:
			idx.collisions[key] = remaining
		}
		return
	}

	if existing, ok := idx.abbreviations[key]; ok && existing == fqname {
		delete(idx.abbreviations, key)
	}
}
