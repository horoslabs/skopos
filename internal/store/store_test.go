package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/horoslabs/skopos/internal/importer"
	"github.com/horoslabs/skopos/internal/model"
)

func host(addr string, svcs ...model.Service) importer.Host {
	return importer.Host{Address: addr, Services: svcs}
}

func svc(port int, product, version string) model.Service {
	return model.Service{Port: port, Proto: "tcp", Product: product, Version: version}
}

func kinds(cs []model.Change) map[string]int {
	m := map[string]int{}
	for _, c := range cs {
		m[c.Kind]++
	}
	return m
}

func TestMergeDiff(t *testing.T) {
	st, err := Load(filepath.Join(t.TempDir(), "db.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()

	first := st.Merge(host("192.0.2.10", svc(22, "OpenSSH", "8.9"), svc(443, "nginx", "1.24")), now, true)
	if k := kinds(first); k["new_host"] != 1 || len(first) != 1 {
		t.Fatalf("primera importación: %+v", first)
	}

	second := st.Merge(host("192.0.2.10", svc(22, "OpenSSH", "9.6"), svc(8080, "Jetty", "9.4")), now, true)
	k := kinds(second)
	if k["changed_service"] != 1 || k["new_service"] != 1 || k["closed_service"] != 1 || len(second) != 3 {
		t.Fatalf("diff inesperado: %+v", second)
	}
	if got := len(st.Assets["192.0.2.10"].Services); got != 2 {
		t.Errorf("servicios guardados = %d, quería 2", got)
	}
}

func TestMergeKeepsClosedWithoutFlag(t *testing.T) {
	st, _ := Load(filepath.Join(t.TempDir(), "db.json"))
	now := time.Now()
	st.Merge(host("192.0.2.10", svc(22, "OpenSSH", "8.9"), svc(443, "nginx", "1.24")), now, false)
	cs := st.Merge(host("192.0.2.10", svc(22, "OpenSSH", "8.9")), now, false)
	if len(cs) != 0 || len(st.Assets["192.0.2.10"].Services) != 2 {
		t.Errorf("sin -detect-closed no debe cerrar nada: %+v", cs)
	}
}

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.json")
	st, _ := Load(path)
	st.Merge(host("192.0.2.10", svc(22, "OpenSSH", "8.9")), time.Now(), false)
	st.Record("scan.xml", nil, []string{"203.0.113.50"}, time.Now())
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Assets) != 1 || len(again.Imports) != 1 || again.Imports[0].Rejected[0] != "203.0.113.50" {
		t.Errorf("recarga incorrecta: %+v", again)
	}
}
