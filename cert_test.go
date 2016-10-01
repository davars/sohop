package sohop

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCertValidity(t *testing.T) {
	data, err := ioutil.ReadFile("fixtures/cert.pem")
	if err != nil {
		t.Fatal(err)
	}
	notBefore, notAfter, err := certValidity(data)
	require.NoError(t, err)
	require.Equal(t, time.Date(2016, 4, 23, 02, 44, 44, 0, time.UTC), notBefore)
	require.Equal(t, time.Date(2017, 4, 23, 02, 44, 44, 0, time.UTC), notAfter)
}
