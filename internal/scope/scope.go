// Package scope parsea y aplica el alcance (scope) de un engagement.
//
// Formato del fichero, una entrada por línea (# para comentarios):
//
//	192.0.2.10          IP concreta
//	192.0.2.0/28        rango CIDR (IPv4 o IPv6)
//	example.com         ese dominio exacto
//	*.example.com       cualquier subdominio (no incluye example.com)
package scope

import (
	"bufio"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strings"
)

// Scope define qué objetivos se pueden tocar. Todo lo demás queda fuera.
type Scope struct {
	prefixes  []netip.Prefix
	domains   map[string]struct{}
	wildcards []string // sufijos ".dominio" de las entradas "*.dominio"
	entries   []string
}

// Load lee un fichero de scope.
func Load(path string) (*Scope, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse lee un scope desde un reader.
func Parse(r io.Reader) (*Scope, error) {
	s := &Scope{domains: map[string]struct{}{}}
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" {
			continue
		}
		if err := s.add(line); err != nil {
			return nil, fmt.Errorf("línea %d: %w", n, err)
		}
		s.entries = append(s.entries, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(s.entries) == 0 {
		return nil, fmt.Errorf("el scope está vacío")
	}
	return s, nil
}

func (s *Scope) add(entry string) error {
	if strings.Contains(entry, "/") {
		p, err := netip.ParsePrefix(entry)
		if err != nil {
			return fmt.Errorf("CIDR inválido %q", entry)
		}
		s.prefixes = append(s.prefixes, p.Masked())
		return nil
	}
	if a, err := netip.ParseAddr(entry); err == nil {
		a = a.Unmap()
		s.prefixes = append(s.prefixes, netip.PrefixFrom(a, a.BitLen()))
		return nil
	}
	if rest, ok := strings.CutPrefix(entry, "*."); ok {
		if !validDomain(rest) {
			return fmt.Errorf("dominio inválido %q", entry)
		}
		s.wildcards = append(s.wildcards, "."+rest)
		return nil
	}
	if !validDomain(entry) {
		return fmt.Errorf("entrada inválida %q (esperaba IP, CIDR o dominio)", entry)
	}
	s.domains[entry] = struct{}{}
	return nil
}

func validDomain(d string) bool {
	labels := strings.Split(d, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if l == "" || len(l) > 63 || strings.HasPrefix(l, "-") || strings.HasSuffix(l, "-") {
			return false
		}
		for _, c := range l {
			ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
			if !ok {
				return false
			}
		}
	}
	return true
}

// Contains indica si una IP o un nombre de host está dentro del scope.
func (s *Scope) Contains(target string) bool {
	t := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(target)), ".")
	if t == "" {
		return false
	}
	if a, err := netip.ParseAddr(t); err == nil {
		a = a.Unmap()
		for _, p := range s.prefixes {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}
	if _, ok := s.domains[t]; ok {
		return true
	}
	for _, w := range s.wildcards {
		if strings.HasSuffix(t, w) {
			return true
		}
	}
	return false
}

// ContainsAny devuelve true si alguno de los objetivos está dentro del scope.
func (s *Scope) ContainsAny(targets ...string) bool {
	for _, t := range targets {
		if s.Contains(t) {
			return true
		}
	}
	return false
}

// Entries devuelve las entradas normalizadas, en el orden del fichero.
func (s *Scope) Entries() []string {
	return append([]string(nil), s.entries...)
}
