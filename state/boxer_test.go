package state

import (
	"testing"

	"reflect"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/davars/sohop/globals"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
)

func TestSealAndReopen(t *testing.T) {
	globals.Clock = fakeclock.NewFakeClock(time.Time{})
	defer func() {
		globals.Clock = clock.NewClock()
	}()

	tests := map[string]struct {
		message proto.Message
		sealed  string
	}{
		"empty": {
			message: &empty.Empty{},
			sealed:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA_lhDK9z2adhnXlIn15w9asw1s39sPQ20FSEII0M",
		},
		"OAuthState empty": {
			message: &OAuthState{},
			sealed:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA_lhDK9z2adhnXlIn15w9asw1s39sPQ20FSEII0M",
		},
		"OAuthState gibberish": {
			message: &OAuthState{RedirectUrl: "asdf"},
			sealed:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAi3cP9pd4IJd3NLSVZu4Hr8w1s39sPQ20FSEII0Pmkl1pZM6z3w",
		},
		"OAuthState long": {
			message: &OAuthState{RedirectUrl: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAi3cP9pd4IJd3NLSVZu4Hr8w1s39sPQ20FSEII0Pmkl1pZM6z3w"},
			sealed:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAFqCunyMTLmKbhvKLYgy5M8w1s39sPQ20FSEII0PmwF0_RPyW-Giq8QSz5tK2AWY8RALWQ5b-qypFWPb2kWu2AfUbu7HaqaT5hno8SfcrrvovRI6Ger1082-DByY9ax4BfkGCFdMmMLcTUJ7FhxgBVQ9zZQ",
		},
		"Session empty": {
			message: &Session{Authorized: false},
			sealed:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA_lhDK9z2adhnXlIn15w9asw1s39sPQ20FSEII0M",
		},
		"Session full": {
			message: &Session{Authorized: true, User: "somebody"},
			sealed:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAANo6lli6oXCN6eIfWuFbtY8w1s39sPQ20FSEII0PmmF1ldtK63EuE1Dzqpg",
		},
	}

	b := boxer{
		noncer: func() [24]byte {
			return [24]byte{}
		},
		secret: [32]byte{},
	}

	for name, test := range tests {
		t.Log(name)
		sealed, err := b.seal(test.message, 0)
		assert.NoError(t, err)
		assert.Equal(t, test.sealed, sealed)

		v := reflect.New(reflect.TypeOf(test.message).Elem()).Interface().(proto.Message)
		assert.True(t, b.open(sealed, v))

		if !reflect.DeepEqual(v, test.message) {
			t.Errorf("got %+v, want %+v", v, test.message)
		}
	}
}

func TestSealFailures(t *testing.T) {
	tests := map[string]struct {
		message proto.Message
		maxAge  int
		now     time.Time
	}{
		"nil": {
			message: nil,
		},
		"future": {
			message: &empty.Empty{},
			now:     time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	b := boxer{
		noncer: func() [24]byte {
			return [24]byte{}
		},
		secret: [32]byte{},
	}

	defer func() {
		globals.Clock = clock.NewClock()
	}()

	for name, test := range tests {
		t.Log(name)
		globals.Clock = fakeclock.NewFakeClock(test.now)
		sealed, err := b.seal(test.message, test.maxAge)
		assert.Error(t, err)
		t.Log(err)
		assert.Equal(t, "", sealed)
	}
}

func TestOpenFailures(t *testing.T) {
	tests := map[string]struct {
		message proto.Message
		sealed  string
		now     time.Time
	}{
		"empty": {
			sealed: "",
		},
		"short": {
			sealed: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		},
		"not decryptable": {
			sealed: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAasfsfsaf",
		},
		"not a proto": {
			sealed: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAS610yvaV-aHUi0pmr6i7hqhRz9-fpb5ehKqY",
		},
		"invalid NotAfter": {
			sealed: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAyLhzIoaRpaXteTb-9-GGh8w4q39qbhIv",
		},
		"expired": {
			sealed: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA_lhDK9z2adhnXlIn15w9asw1s39sPQ20FSEII0M",
			now:    time.Time{}.Add(1),
		},
	}

	b := boxer{
		noncer: func() [24]byte {
			return [24]byte{}
		},
		secret: [32]byte{},
	}

	defer func() {
		globals.Clock = clock.NewClock()
	}()

	for name, test := range tests {
		t.Log(name)
		globals.Clock = fakeclock.NewFakeClock(test.now)

		v := &empty.Empty{}
		assert.False(t, b.open(test.sealed, v))
	}
}
