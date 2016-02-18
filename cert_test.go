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
	notBefore, notAfter, err := CertValidity(data)
	require.NoError(t, err)
	require.Equal(t, time.Date(2016, 2, 16, 12, 28, 28, 0, time.UTC), notBefore)
	require.Equal(t, time.Date(2017, 2, 15, 12, 28, 28, 0, time.UTC), notAfter)
}
