// Package ids mints short collision-resistant identifiers.
//
// Task/decision IDs carry a random 4-char component instead of a sequence
// number so that two branches minting IDs concurrently cannot collide on
// the same "next" number. The alphabet omits ambiguous glyphs (0/o, 1/i/l, u).
package ids

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const alphabet = "23456789abcdefghjkmnpqrstvwxyz"

func randChars(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		r, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err) // crypto/rand failure is unrecoverable
		}
		b[i] = alphabet[r.Int64()]
	}
	return string(b)
}

// NewTask mints a task ID, retrying while exists reports a collision
// with an ID already present in local state.
func NewTask(exists func(string) bool) string { return mint("T-", exists) }

// NewDecision mints a decision ID.
func NewDecision(exists func(string) bool) string { return mint("D-", exists) }

func mint(prefix string, exists func(string) bool) string {
	for {
		id := prefix + randChars(4)
		if exists == nil || !exists(id) {
			return id
		}
	}
}

// NewEvent returns a time-prefixed event ID so lexicographic order on IDs
// approximates creation order and breaks TS ties deterministically.
func NewEvent(t time.Time) string {
	return fmt.Sprintf("%011x-%s", t.UnixMilli(), randChars(4))
}
