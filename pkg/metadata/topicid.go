package metadata

import (
	"crypto/sha1"
)

// TopicIDForName returns a stable 16-byte ID derived from a topic name.
func TopicIDForName(name string) [16]byte {
	sum := sha1.Sum([]byte(name))
	var id [16]byte
	copy(id[:], sum[:16])
	return id
}
