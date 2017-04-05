package wifimanager

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

// Regex testing was done on: https://regex101.com/r/RZzdwY/1
var networkRegex = regexp.MustCompile("(?s)network={(?P<network>.*?)}")
var ssidRegex = regexp.MustCompile(`^ssid="(?P<ssid>.*)"`)
var passwordRegex = regexp.MustCompile(`^#psk=(?P<password>.*)`)
var pskRegex = regexp.MustCompile(`^psk=(?P<psk>.*)`)

type WPANetwork struct {
	SSID     string
	PSK      string
	Password string
}

func (wn *WPANetwork) String() string {
	return fmt.Sprintf("(ssid=%v psk=%v)", wn.SSID, wn.PSK)
}

func (wn *WPANetwork) AsConf() string {
	return fmt.Sprintf(`
network={
	ssid="%v"
	#psk=%v
	psk=%v
}`, wn.SSID, wn.Password, wn.PSK)
}

func ParseWPANetwork(s string) *WPANetwork {
	lines := strings.Split(s, "\n")
	var (
		ssid     string
		password string
		psk      string
	)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "ssid=") {
			match := ssidRegex.FindStringSubmatch(line)
			if len(match) > 0 {
				m := mapSubexpNames(match, ssidRegex.SubexpNames())
				ssid = m["ssid"]
			}
		} else if strings.Contains(line, "#psk=") {
			match := passwordRegex.FindStringSubmatch(line)
			if len(match) > 0 {
				m := mapSubexpNames(match, pskRegex.SubexpNames())
				password = m["password"]
			}
		} else if strings.Contains(line, "psk=") {
			match := pskRegex.FindStringSubmatch(line)
			if len(match) > 0 {
				m := mapSubexpNames(match, pskRegex.SubexpNames())
				psk = m["psk"]
			}
		}
	}

	if len(ssid) > 0 {
		return &WPANetwork{
			SSID:     ssid,
			Password: password,
			PSK:      psk,
		}
	} else {
		return nil
	}
}

func mapSubexpNames(m, n []string) map[string]string {
	m, n = m[1:], n[1:]
	r := make(map[string]string, len(m))
	for i, _ := range n {
		r[n[i]] = m[i]
	}
	return r
}

func ParseWPASupplicantConf(path string) ([]*WPANetwork, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseConf(string(data))
}

func parseConf(data string) ([]*WPANetwork, error) {
	result := make([]*WPANetwork, 0)

	matches := networkRegex.FindAllStringSubmatch(data, -1)
	for _, str := range matches {
		m := mapSubexpNames(str, networkRegex.SubexpNames())
		wn := ParseWPANetwork(strings.TrimSpace(m["network"]))
		if wn != nil {
			result = append(result, wn)
		}
	}
	return result, nil
}
