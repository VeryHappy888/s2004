package register

import "strings"

type ISPConfig struct {
	Country string
	MCC     string
	MNC     string
	ISO     string
	Code    string
}

func ISP(ISO string) ISPConfig {
	switch strings.ToUpper(ISO) {
	case "KH":
		return ISPConfig{"柬埔寨", "456", "003", "KH", "855"}
	case "CA":
		return ISPConfig{"加拿大", "302", "012", "CA", "1"}
	case "UK":
		return ISPConfig{"英国", "234", "012", "UK", "44"}
	case "MY":
		return ISPConfig{"马来西亚", "502", "001", "MY", "60"}
	case "HK":
		return ISPConfig{"中国香港", "454", "012", "HK", "852"}
	default:
		return ISPConfig{"中国大陆", "460", "001", "CN", "86"}
	}
}
