// Package report genera informes del inventario en varios formatos.
package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/horoslabs/skopos/internal/model"
	"github.com/horoslabs/skopos/internal/scope"
	"github.com/horoslabs/skopos/internal/store"
)

// Count es un servicio y las veces que aparece en el inventario.
type Count struct {
	Label string
	N     int
}

// View es todo lo que necesita un informe.
type View struct {
	Title         string
	Generated     time.Time
	ScopeEntries  []string
	Assets        []*model.Asset
	TotalHosts    int
	TotalServices int
	TopServices   []Count
	LastImport    *store.ImportRecord
}

// Build prepara la vista a partir del inventario y el scope.
func Build(title string, sc *scope.Scope, st *store.Store, now time.Time) View {
	v := View{Title: title, Generated: now}
	if sc != nil {
		v.ScopeEntries = sc.Entries()
	}
	counts := map[string]int{}
	for _, a := range st.Assets {
		v.Assets = append(v.Assets, a)
		v.TotalServices += len(a.Services)
		for _, svc := range a.Services {
			label := svc.Key()
			if svc.Name != "" {
				label += " " + svc.Name
			}
			counts[label]++
		}
	}
	sort.Slice(v.Assets, func(i, j int) bool { return v.Assets[i].Address < v.Assets[j].Address })
	v.TotalHosts = len(v.Assets)
	for l, n := range counts {
		v.TopServices = append(v.TopServices, Count{l, n})
	}
	sort.Slice(v.TopServices, func(i, j int) bool {
		a, b := v.TopServices[i], v.TopServices[j]
		if a.N != b.N {
			return a.N > b.N
		}
		return a.Label < b.Label
	})
	if len(v.TopServices) > 10 {
		v.TopServices = v.TopServices[:10]
	}
	if n := len(st.Imports); n > 0 {
		v.LastImport = &st.Imports[n-1]
	}
	return v
}

// Write escribe el informe en el formato indicado: html, md, csv o json.
func Write(format string, w io.Writer, v View) error {
	switch format {
	case "html":
		return writeHTML(w, v)
	case "md", "markdown":
		return writeMarkdown(w, v)
	case "csv":
		return writeCSV(w, v)
	case "json":
		return writeJSON(w, v)
	}
	return fmt.Errorf("formato desconocido %q (usa html, md, csv o json)", format)
}

var kindLabels = map[string]string{
	"new_host":        "Nuevo host",
	"new_service":     "Nuevo puerto",
	"changed_service": "Servicio cambiado",
	"closed_service":  "Puerto cerrado",
}

func kindLabel(k string) string {
	if l, ok := kindLabels[k]; ok {
		return l
	}
	return k
}

func writeHTML(w io.Writer, v View) error {
	t, err := template.New("report").Funcs(template.FuncMap{
		"join":      strings.Join,
		"date":      func(t time.Time) string { return t.Format("2006-01-02 15:04") },
		"kindLabel": kindLabel,
		"label":     func(s model.Service) string { return s.Label() },
	}).Parse(htmlTpl)
	if err != nil {
		return err
	}
	return t.Execute(w, v)
}

func esc(s string) string { return strings.ReplaceAll(s, "|", "\\|") }

func writeMarkdown(w io.Writer, v View) error {
	fmt.Fprintf(w, "# %s\n\n", v.Title)
	fmt.Fprintf(w, "Generado: %s · %d hosts · %d puertos abiertos\n\n", v.Generated.Format("2006-01-02 15:04"), v.TotalHosts, v.TotalServices)
	fmt.Fprintf(w, "## Scope\n\n")
	for _, e := range v.ScopeEntries {
		fmt.Fprintf(w, "- `%s`\n", e)
	}
	if v.LastImport != nil && len(v.LastImport.Changes) > 0 {
		fmt.Fprintf(w, "\n## Cambios de la última importación (%s)\n\n| Tipo | Host | Detalle |\n|---|---|---|\n", v.LastImport.Source)
		for _, c := range v.LastImport.Changes {
			fmt.Fprintf(w, "| %s | %s | %s |\n", kindLabel(c.Kind), c.Address, esc(c.Detail))
		}
	}
	fmt.Fprintf(w, "\n## Inventario\n\n| Host | Nombres | Servicios |\n|---|---|---|\n")
	for _, a := range v.Assets {
		var labels []string
		for _, s := range a.Services {
			labels = append(labels, esc(s.Label()))
		}
		fmt.Fprintf(w, "| %s | %s | %s |\n", a.Address, esc(strings.Join(a.Hostnames, ", ")), strings.Join(labels, "<br>"))
	}
	return nil
}

func writeCSV(w io.Writer, v View) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"address", "hostnames", "port", "proto", "service", "product", "version", "first_seen", "last_seen"})
	for _, a := range v.Assets {
		host := strings.Join(a.Hostnames, " ")
		if len(a.Services) == 0 {
			_ = cw.Write([]string{a.Address, host, "", "", "", "", "", a.FirstSeen.Format(time.RFC3339), a.LastSeen.Format(time.RFC3339)})
		}
		for _, s := range a.Services {
			_ = cw.Write([]string{a.Address, host, strconv.Itoa(s.Port), s.Proto, s.Name, s.Product, s.Version,
				a.FirstSeen.Format(time.RFC3339), a.LastSeen.Format(time.RFC3339)})
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeJSON(w io.Writer, v View) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		Title        string              `json:"title"`
		Generated    time.Time           `json:"generated"`
		Scope        []string            `json:"scope"`
		Hosts        int                 `json:"hosts"`
		OpenPorts    int                 `json:"open_ports"`
		Assets       []*model.Asset      `json:"assets"`
		LastImport   *store.ImportRecord `json:"last_import,omitempty"`
		TopServicesN []Count             `json:"top_services"`
	}{v.Title, v.Generated, v.ScopeEntries, v.TotalHosts, v.TotalServices, v.Assets, v.LastImport, v.TopServices})
}
