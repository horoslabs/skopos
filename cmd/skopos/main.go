// skopos mantiene un inventario de superficie externa a partir de escaneos,
// respeta el scope del engagement y genera informes.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/horoslabs/skopos/internal/importer"
	"github.com/horoslabs/skopos/internal/model"
	"github.com/horoslabs/skopos/internal/report"
	"github.com/horoslabs/skopos/internal/scope"
	"github.com/horoslabs/skopos/internal/store"
)

var version = "dev"

const usage = `skopos — inventario de superficie externa

Uso:
  skopos scope check [-scope scope.txt] <host|ip>...
  skopos import nmap [-scope scope.txt] [-db skopos.json] [-detect-closed] <scan.xml>
  skopos report      [-scope scope.txt] [-db skopos.json] [-format html|md|csv|json] [-o fichero] [-title texto]
  skopos version

Notas:
  - Todo lo que no esté en el scope se rechaza al importar y queda anotado.
  - -detect-closed marca como cerrados los puertos que ya no aparecen; úsalo
    solo si el nuevo escaneo cubre el mismo rango de puertos.
`

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("falta el subcomando")
	}
	switch args[0] {
	case "scope":
		return cmdScope(args[1:], out)
	case "import":
		return cmdImport(args[1:], out)
	case "report":
		return cmdReport(args[1:], out)
	case "version", "-v", "--version":
		fmt.Fprintln(out, version)
		return nil
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
		return nil
	}
	fmt.Fprint(os.Stderr, usage)
	return fmt.Errorf("subcomando desconocido %q", args[0])
}

func cmdScope(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "check" {
		return fmt.Errorf("uso: skopos scope check [-scope scope.txt] <host|ip>...")
	}
	fs := flag.NewFlagSet("scope check", flag.ContinueOnError)
	scopePath := fs.String("scope", "scope.txt", "fichero de scope")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	sc, err := scope.Load(*scopePath)
	if err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("indica al menos un host o IP")
	}
	for _, t := range fs.Args() {
		state := "FUERA"
		if sc.Contains(t) {
			state = "DENTRO"
		}
		fmt.Fprintf(out, "%-7s %s\n", state, t)
	}
	return nil
}

func cmdImport(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "nmap" {
		return fmt.Errorf("uso: skopos import nmap [flags] <scan.xml>")
	}
	fs := flag.NewFlagSet("import nmap", flag.ContinueOnError)
	scopePath := fs.String("scope", "scope.txt", "fichero de scope")
	dbPath := fs.String("db", "skopos.json", "fichero del inventario")
	detectClosed := fs.Bool("detect-closed", false, "marcar como cerrados los puertos que ya no aparecen")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("indica el XML de nmap (-oX)")
	}
	file := fs.Arg(0)

	sc, err := scope.Load(*scopePath)
	if err != nil {
		return fmt.Errorf("scope: %w", err)
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	hosts, err := importer.ParseNmap(f)
	if err != nil {
		return err
	}
	st, err := store.Load(*dbPath)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	var changes []model.Change
	var rejected []string
	imported := 0
	for _, h := range hosts {
		targets := append([]string{h.Address}, h.Hostnames...)
		if !sc.ContainsAny(targets...) {
			label := h.Address
			if len(h.Hostnames) > 0 {
				label += " (" + strings.Join(h.Hostnames, ", ") + ")"
			}
			rejected = append(rejected, label)
			continue
		}
		changes = append(changes, st.Merge(h, now, *detectClosed)...)
		imported++
	}
	st.Record(filepath.Base(file), changes, rejected, now)
	if err := st.Save(); err != nil {
		return err
	}

	fmt.Fprintf(out, "Importados: %d hosts · Cambios: %d · Fuera de scope: %d\n", imported, len(changes), len(rejected))
	for _, r := range rejected {
		fmt.Fprintf(out, "  FUERA DE SCOPE (no importado): %s\n", r)
	}
	for _, c := range changes {
		fmt.Fprintf(out, "  %-16s %-15s %s\n", c.Kind, c.Address, c.Detail)
	}
	return nil
}

// sameFile indica si dos rutas apuntan al mismo fichero (existente) o, si
// alguno no existe todavía, a la misma ruta absoluta.
func sameFile(a, b string) bool {
	if ia, err := os.Stat(a); err == nil {
		if ib, err := os.Stat(b); err == nil {
			return os.SameFile(ia, ib)
		}
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && absA == absB
}

func cmdReport(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	scopePath := fs.String("scope", "scope.txt", "fichero de scope")
	dbPath := fs.String("db", "skopos.json", "fichero del inventario")
	format := fs.String("format", "", "html, md, csv o json (por defecto, según la extensión de -o; si no, html)")
	output := fs.String("o", "", "fichero de salida (por defecto, stdout)")
	title := fs.String("title", "Inventario de superficie externa", "título del informe")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := store.Load(*dbPath)
	if err != nil {
		return err
	}
	var sc *scope.Scope
	if s, err := scope.Load(*scopePath); err == nil {
		sc = s
	}

	f := *format
	if f == "" {
		switch strings.ToLower(filepath.Ext(*output)) {
		case ".md":
			f = "md"
		case ".csv":
			f = "csv"
		case ".json":
			f = "json"
		default:
			f = "html"
		}
	}

	w := out
	if *output != "" {
		if sameFile(*output, *dbPath) {
			return fmt.Errorf("-o apunta al fichero del inventario (%s); elige otro nombre para no pisarlo", *dbPath)
		}
		file, err := os.Create(*output)
		if err != nil {
			return err
		}
		defer file.Close()
		w = file
	}
	return report.Write(f, w, report.Build(*title, sc, st, time.Now().UTC()))
}
