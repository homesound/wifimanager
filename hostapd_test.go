package wifimanager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHotspot(t *testing.T) {
	require := require.New(t)

	wm, err := New("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	err = wm.StartHotspot("wlan0")
	require.Nil(err)

	err = wm.StopHotspot("wlan0")
	require.Nil(err)
}
