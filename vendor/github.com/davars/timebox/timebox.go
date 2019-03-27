package timebox

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/davars/timebox/internal/timebox"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/crypto/nacl/secretbox"
)

const nonceLength = 24

// Boxer provides a limited API for encrypting protobuf messages that expire after some time
type Boxer struct {
	noncer func() [nonceLength]byte
	secret [32]byte
}

// New returns a new Boxer
func New(secret string) (*Boxer, error) {
	secretKeyBytes, err := hex.DecodeString(secret)
	if err != nil || len(secretKeyBytes) != 32 {
		var freshKey [32]byte
		if _, err := io.ReadFull(rand.Reader, freshKey[:]); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf(
			"The cookie secret should be a 64-character hex-encoded string.  "+
				"Here's a freshly generated one: %q",
			hex.EncodeToString(freshKey[:]))
	}

	var secretKey [32]byte
	copy(secretKey[:], secretKeyBytes)

	return &Boxer{
		secret: secretKey,
		noncer: func() [nonceLength]byte {
			// You must use a different nonce for each message you encrypt with the
			// same key. Since the nonce here is 192 bits long, a random value
			// provides a sufficiently small probability of repeats.
			var nonce [nonceLength]byte
			if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
				panic(err) // don't want to continue encrypting anything
			}
			return nonce
		},
	}, nil
}

// Seal encrypts the given message in a box that expires after maxAge has elapsed
func (b *Boxer) Seal(message proto.Message, maxAge time.Duration) (string, error) {
	marshaled, err := proto.Marshal(message)
	if err != nil {
		return "", err
	}

	expires, err := ptypes.TimestampProto(Clock.Now().Add(maxAge))
	if err != nil {
		return "", err
	}

	boxed, err := proto.Marshal(&timebox.TimeBox{Payload: marshaled, NotAfter: expires})
	if err != nil {
		return "", err
	}

	n := b.noncer()
	encrypted := secretbox.Seal(n[:], boxed, &n, &b.secret)
	return base64.RawURLEncoding.EncodeToString(encrypted), nil
}

// Open takes an encrpyted timebox and attempts to decrypt and unmarshal it into output. If all operations succeed
// (decryption, verifying expiration, unmarshalling), Open returns true.  Otherwise, Open returns false.
func (b *Boxer) Open(sealed string, output proto.Message) bool {
	encrypted, err := base64.RawURLEncoding.DecodeString(sealed)
	if err != nil || len(encrypted) < nonceLength+1 {
		return false
	}

	var decryptNonce [nonceLength]byte
	copy(decryptNonce[:], encrypted[:nonceLength])

	decrypted, ok := secretbox.Open([]byte{}, encrypted[nonceLength:], &decryptNonce, &b.secret)
	if !ok {
		return false
	}

	box := &timebox.TimeBox{}
	if err := proto.Unmarshal(decrypted, box); err != nil {
		return false
	}

	notAfter, err := ptypes.Timestamp(box.NotAfter)
	if err != nil {
		return false
	}

	if Clock.Now().After(notAfter) {
		return false
	}

	if err := proto.Unmarshal(box.Payload, output); err != nil {
		return false
	}

	return true
}
