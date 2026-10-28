package importer

import (
	"os"
	"testing"
)

func TestParseNmap(t *testing.T) {
	f, err := os.Open("../../testdata/scan1.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	hosts, err := ParseNmap(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 3 { // el host caído se ignora
		t.Fatalf("hosts = %d, quería 3", len(hosts))
	}
	h := hosts[0]
	if h.Address != "192.0.2.10" || len(h.Hostnames) != 1 || h.Hostnames[0] != "vpn.example.com" {
		t.Errorf("host inesperado: %+v", h)
	}
	if len(h.Services) != 2 { // el puerto cerrado se ignora
		t.Fatalf("servicios = %d, quería 2", len(h.Services))
	}
	if s := h.Services[1]; s.Port != 443 || s.Product != "nginx" || s.Version != "1.24.0" {
		t.Errorf("servicio inesperado: %+v", s)
	}
}

func TestParseNmapInvalid(t *testing.T) {
	f, _ := os.Open("../../testdata/scope.txt")
	defer f.Close()
	if _, err := ParseNmap(f); err == nil {
		t.Error("un fichero que no es XML debería fallar")
	}
}
