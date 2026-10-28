// Package store guarda el inventario en un fichero JSON y detecta cambios.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/horoslabs/skopos/internal/importer"
	"github.com/horoslabs/skopos/internal/model"
)

const maxImports = 50

// ImportRecord resume una importación: qué cambió y qué se rechazó.
type ImportRecord struct {
	Time     time.Time      `json:"time"`
	Source   string         `json:"source"`
	Changes  []model.Change `json:"changes"`
	Rejected []string       `json:"rejected,omitempty"`
}

// Store es el inventario completo.
type Store struct {
	Assets  map[string]*model.Asset `json:"assets"`
	Imports []ImportRecord          `json:"imports"`
	path    string
}

// Load abre el inventario; si el fichero no existe devuelve uno vacío.
func Load(path string) (*Store, error) {
	s := &Store{Assets: map[string]*model.Asset{}, path: path}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if s.Assets == nil {
		s.Assets = map[string]*model.Asset{}
	}
	return s, nil
}

// Save escribe el inventario de forma atómica.
func (s *Store) Save() error {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Merge incorpora un host y devuelve los cambios respecto a lo guardado.
// Con detectClosed, los puertos que ya no aparecen se marcan como cerrados
// (úsalo solo si el nuevo escaneo cubre el mismo rango de puertos).
func (s *Store) Merge(h importer.Host, now time.Time, detectClosed bool) []model.Change {
	var changes []model.Change
	a, existed := s.Assets[h.Address]
	if !existed {
		a = &model.Asset{Address: h.Address, FirstSeen: now}
		s.Assets[h.Address] = a
		changes = append(changes, model.Change{
			Kind: "new_host", Address: h.Address,
			Detail: fmt.Sprintf("%d puertos abiertos", len(h.Services)),
		})
	}
	a.LastSeen = now
	a.Hostnames = union(a.Hostnames, h.Hostnames)

	current := map[string]model.Service{}
	for _, svc := range a.Services {
		current[svc.Key()] = svc
	}
	seen := map[string]bool{}
	for _, svc := range h.Services {
		k := svc.Key()
		seen[k] = true
		old, had := current[k]
		switch {
		case !had && existed:
			changes = append(changes, model.Change{Kind: "new_service", Address: h.Address, Detail: svc.Label()})
		case had && (old.Product != svc.Product || old.Version != svc.Version):
			changes = append(changes, model.Change{
				Kind: "changed_service", Address: h.Address,
				Detail: fmt.Sprintf("%s: %q → %q", k, old.Product+" "+old.Version, svc.Product+" "+svc.Version),
			})
		}
		current[k] = svc
	}
	if detectClosed {
		for k, old := range current {
			if !seen[k] {
				changes = append(changes, model.Change{Kind: "closed_service", Address: h.Address, Detail: old.Label()})
				delete(current, k)
			}
		}
	}

	a.Services = a.Services[:0]
	for _, svc := range current {
		a.Services = append(a.Services, svc)
	}
	sort.Slice(a.Services, func(i, j int) bool {
		x, y := a.Services[i], a.Services[j]
		if x.Proto != y.Proto {
			return x.Proto < y.Proto
		}
		return x.Port < y.Port
	})
	return changes
}

// Record añade una importación al historial (se conservan las últimas 50).
func (s *Store) Record(source string, changes []model.Change, rejected []string, now time.Time) {
	sort.Slice(changes, func(i, j int) bool {
		a, b := changes[i], changes[j]
		if a.Address != b.Address {
			return a.Address < b.Address
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Detail < b.Detail
	})
	s.Imports = append(s.Imports, ImportRecord{Time: now, Source: source, Changes: changes, Rejected: rejected})
	if len(s.Imports) > maxImports {
		s.Imports = s.Imports[len(s.Imports)-maxImports:]
	}
}

func union(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range append(append([]string(nil), a...), b...) {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
