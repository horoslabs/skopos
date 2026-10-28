// Package model define los datos del inventario.
package model

import (
	"fmt"
	"time"
)

// Service es un puerto abierto con lo que se sabe de él.
type Service struct {
	Port      int    `json:"port"`
	Proto     string `json:"proto"`
	Name      string `json:"name,omitempty"`
	Product   string `json:"product,omitempty"`
	Version   string `json:"version,omitempty"`
	ExtraInfo string `json:"extra_info,omitempty"`
}

// Key identifica un servicio dentro de un host.
func (s Service) Key() string { return fmt.Sprintf("%d/%s", s.Port, s.Proto) }

// Label es una descripción corta para informes.
func (s Service) Label() string {
	out := s.Key()
	if s.Name != "" {
		out += " " + s.Name
	}
	if s.Product != "" {
		out += " · " + s.Product
		if s.Version != "" {
			out += " " + s.Version
		}
	}
	return out
}

// Asset es un host del inventario, identificado por su IP.
type Asset struct {
	Address   string    `json:"address"`
	Hostnames []string  `json:"hostnames,omitempty"`
	Services  []Service `json:"services"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// Change describe una diferencia detectada al importar un escaneo.
type Change struct {
	Kind    string `json:"kind"` // new_host, new_service, changed_service, closed_service
	Address string `json:"address"`
	Detail  string `json:"detail"`
}
