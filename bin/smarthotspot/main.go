package main

import (
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/homesound/wifimanager"
	"github.com/homesound/wifimanager/smarthotspot"
	log "github.com/sirupsen/logrus"
)

var (
	app         = kingpin.New("smarthotspot", "Auto-host hotspot if no network found")
	iface       = app.Arg("iface", "Interface to use").String()
	wpaConfPath = app.Flag("wpa-conf", "Path to wpa_supplicant configuration file").Short('w').Default("/etc/wpa_supplicant.conf").String()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	wifiManager, err := wifimanager.New(*wpaConfPath)
	if err != nil {
		log.Errorf("Failed with error: %v", err)
		os.Exit(-1)
	}
	if err := smarthotspot.StartSmartHotspot(wifiManager, *iface); err != nil {
		log.Errorf("Error in smart-hotspot: %v", err)
		os.Exit(-1)
	}
}
