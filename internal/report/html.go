package report

// htmlTpl es un informe autocontenido: un solo fichero, sin dependencias
// externas, con modo claro/oscuro y un filtro de texto.
const htmlTpl = `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
:root{--bg:#fafaf9;--fg:#1c1917;--muted:#78716c;--card:#ffffff;--line:#e7e5e4;--accent:#b45309;--good:#15803d;--warn:#b45309;--bad:#b91c1c}
@media (prefers-color-scheme:dark){:root{--bg:#141210;--fg:#f5f5f4;--muted:#a8a29e;--card:#1c1917;--line:#2f2a27;--accent:#f59e0b;--good:#4ade80;--warn:#fbbf24;--bad:#f87171}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:15px/1.5 system-ui,-apple-system,"Segoe UI",sans-serif}
main{max-width:1100px;margin:0 auto;padding:24px 16px 64px}
h1{font-size:1.5rem;margin:0 0 4px}
h2{font-size:1.05rem;margin:32px 0 10px}
.meta{color:var(--muted);font-size:.85rem}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:12px;margin-top:20px}
.card{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:14px 16px}
.card b{display:block;font-size:1.6rem;line-height:1.2}
.card span{color:var(--muted);font-size:.8rem}
.wrap{overflow-x:auto;background:var(--card);border:1px solid var(--line);border-radius:10px}
table{width:100%;border-collapse:collapse}
th,td{text-align:left;padding:9px 12px;border-bottom:1px solid var(--line);vertical-align:top}
th{font-size:.75rem;text-transform:uppercase;letter-spacing:.04em;color:var(--muted);font-weight:600}
tr:last-child td{border-bottom:0}
code{font:13px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace}
.svc{display:block;white-space:nowrap}
.k-new_host,.k-new_service{color:var(--good)}
.k-changed_service{color:var(--warn)}
.k-closed_service{color:var(--bad)}
.scope code{display:inline-block;background:var(--card);border:1px solid var(--line);border-radius:6px;padding:1px 8px;margin:0 6px 6px 0}
input{width:100%;max-width:360px;padding:9px 12px;border:1px solid var(--line);border-radius:8px;background:var(--card);color:var(--fg);font:inherit;margin-bottom:10px}
.rej{color:var(--bad)}
@media (max-width:640px){
  .svc{white-space:normal;word-break:break-word}
  #inv th:nth-child(4),#inv td:nth-child(4){display:none}
  th,td{padding:8px 10px}
}
</style>
</head>
<body>
<main>
<h1>{{.Title}}</h1>
<div class="meta">Generado {{date .Generated}}</div>

<div class="cards">
  <div class="card"><b>{{.TotalHosts}}</b><span>hosts</span></div>
  <div class="card"><b>{{.TotalServices}}</b><span>puertos abiertos</span></div>
  <div class="card"><b>{{len .ScopeEntries}}</b><span>entradas de scope</span></div>
</div>

<h2>Scope</h2>
<div class="scope">{{range .ScopeEntries}}<code>{{.}}</code>{{end}}</div>

{{with .LastImport}}
  {{if .Changes}}
  <h2>Cambios de la última importación <span class="meta">({{.Source}}, {{date .Time}})</span></h2>
  <div class="wrap"><table>
    <thead><tr><th>Tipo</th><th>Host</th><th>Detalle</th></tr></thead>
    <tbody>
    {{range .Changes}}<tr class="k-{{.Kind}}"><td>{{kindLabel .Kind}}</td><td><code>{{.Address}}</code></td><td>{{.Detail}}</td></tr>
    {{end}}</tbody>
  </table></div>
  {{end}}
  {{if .Rejected}}
  <h2 class="rej">Fuera de scope (no importado)</h2>
  <div class="wrap"><table><tbody>
    {{range .Rejected}}<tr><td><code>{{.}}</code></td></tr>
    {{end}}</tbody></table></div>
  {{end}}
{{end}}

{{if .TopServices}}
<h2>Servicios más frecuentes</h2>
<div class="wrap"><table>
  <thead><tr><th>Servicio</th><th>Hosts</th></tr></thead>
  <tbody>{{range .TopServices}}<tr><td><code>{{.Label}}</code></td><td>{{.N}}</td></tr>
  {{end}}</tbody>
</table></div>
{{end}}

<h2>Inventario</h2>
<input id="q" type="search" placeholder="Filtrar por host, nombre o servicio" aria-label="Filtrar inventario">
<div class="wrap"><table id="inv">
  <thead><tr><th>Host</th><th>Nombres</th><th>Servicios abiertos</th><th>Visto</th></tr></thead>
  <tbody>
  {{range .Assets}}<tr>
    <td><code>{{.Address}}</code></td>
    <td>{{join .Hostnames ", "}}</td>
    <td>{{range .Services}}<code class="svc">{{label .}}</code>{{end}}</td>
    <td class="meta">{{date .FirstSeen}}<br>{{date .LastSeen}}</td>
  </tr>
  {{end}}</tbody>
</table></div>
</main>
<script>
document.getElementById("q").addEventListener("input", function (e) {
  var needle = e.target.value.toLowerCase();
  var rows = document.querySelectorAll("#inv tbody tr");
  for (var i = 0; i < rows.length; i++) {
    rows[i].style.display = rows[i].textContent.toLowerCase().indexOf(needle) === -1 ? "none" : "";
  }
});
</script>
</body>
</html>
`
