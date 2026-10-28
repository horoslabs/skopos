// Package importer convierte la salida de otras herramientas al modelo común.
package importer

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"

	"github.com/horoslabs/skopos/internal/model"
)

// Host es un host descubierto en un escaneo, antes de filtrarlo por scope.
type Host struct {
	Address   string
	Hostnames []string
	Services  []model.Service
}

type nmapRun struct {
	Hosts []nmapHost `xml:"host"`
}

type nmapHost struct {
	Status struct {
		State string `xml:"state,attr"`
	} `xml:"status"`
	Addresses []struct {
		Addr string `xml:"addr,attr"`
		Type string `xml:"addrtype,attr"`
	} `xml:"address"`
	Hostnames []struct {
		Name string `xml:"name,attr"`
	} `xml:"hostnames>hostname"`
	Ports []nmapPort `xml:"ports>port"`
}

type nmapPort struct {
	Proto string `xml:"protocol,attr"`
	ID    int    `xml:"portid,attr"`
	State struct {
		State string `xml:"state,attr"`
	} `xml:"state"`
	Service struct {
		Name      string `xml:"name,attr"`
		Product   string `xml:"product,attr"`
		Version   string `xml:"version,attr"`
		ExtraInfo string `xml:"extrainfo,attr"`
	} `xml:"service"`
}

// ParseNmap lee un XML de nmap (-oX) y devuelve los hosts activos con sus
// puertos abiertos. Ignora hosts caídos y puertos cerrados o filtrados.
func ParseNmap(r io.Reader) ([]Host, error) {
	var run nmapRun
	if err := xml.NewDecoder(r).Decode(&run); err != nil {
		return nil, fmt.Errorf("XML de nmap no válido: %w", err)
	}
	var hosts []Host
	for _, h := range run.Hosts {
		if h.Status.State != "up" {
			continue
		}
		addr := ""
		for _, a := range h.Addresses {
			if a.Type == "ipv4" || a.Type == "ipv6" {
				addr = a.Addr
				break
			}
		}
		if addr == "" {
			continue
		}
		out := Host{Address: addr}
		seen := map[string]bool{}
		for _, n := range h.Hostnames {
			if n.Name != "" && !seen[n.Name] {
				seen[n.Name] = true
				out.Hostnames = append(out.Hostnames, n.Name)
			}
		}
		for _, p := range h.Ports {
			if p.State.State != "open" {
				continue
			}
			out.Services = append(out.Services, model.Service{
				Port:      p.ID,
				Proto:     p.Proto,
				Name:      p.Service.Name,
				Product:   p.Service.Product,
				Version:   p.Service.Version,
				ExtraInfo: p.Service.ExtraInfo,
			})
		}
		sort.Slice(out.Services, func(i, j int) bool {
			a, b := out.Services[i], out.Services[j]
			if a.Proto != b.Proto {
				return a.Proto < b.Proto
			}
			return a.Port < b.Port
		})
		hosts = append(hosts, out)
	}
	return hosts, nil
}
