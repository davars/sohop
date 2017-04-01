package state

import (
	"encoding/base64"
	"time"

	"github.com/davars/sohop/globals"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	nonceLen = 24
)

// boxer provides a limited API for encrypting protobuf messages that expire after some time
type boxer struct {
	noncer func() [nonceLen]byte
	secret [32]byte
}

// seal encrypts the given message in a box that expires after maxAge seconds
func (b *boxer) seal(message proto.Message, maxAge int) (string, error) {
	marshaled, err := proto.Marshal(message)
	if err != nil {
		return "", err
	}

	expires, err := ptypes.TimestampProto(globals.Clock.Now().Add(time.Duration(maxAge) * time.Second))
	if err != nil {
		return "", err
	}

	boxed, err := proto.Marshal(&TimeBox{Payload: marshaled, NotAfter: expires})
	if err != nil {
		return "", err
	}

	n := b.noncer()
	encrypted := secretbox.Seal(n[:], boxed, &n, &b.secret)
	return base64.RawURLEncoding.EncodeToString(encrypted), nil
}

// open takes an encrpyted TimeBox and attempts to decrypt and unmarshal it into output. If all operations succeed
// (decryption, verifying expiration, unmarshalling), open returns true.  Otherwise, open returns false.
func (b *boxer) open(sealed string, output proto.Message) bool {
	encrypted, err := base64.RawURLEncoding.DecodeString(sealed)
	if err != nil || len(encrypted) < nonceLen+1 {
		return false
	}

	var decryptNonce [nonceLen]byte
	copy(decryptNonce[:], encrypted[:nonceLen])

	decrypted, ok := secretbox.Open([]byte{}, encrypted[nonceLen:], &decryptNonce, &b.secret)
	if !ok {
		return false
	}

	box := &TimeBox{}
	if err := proto.Unmarshal(decrypted, box); err != nil {
		return false
	}

	notAfter, err := ptypes.Timestamp(box.NotAfter)
	if err != nil {
		return false
	}

	if globals.Clock.Now().After(notAfter) {
		return false
	}

	if err := proto.Unmarshal(box.Payload, output); err != nil {
		return false
	}

	return true
}
