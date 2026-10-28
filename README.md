# skopos

Inventario de superficie externa para engagements de pentest. Importa escaneos de nmap, **rechaza todo lo que esté fuera del scope**, detecta qué cambia entre escaneos y genera informes en HTML, Markdown, CSV y JSON.

Parte de [Horos Labs](https://horoslabs.com). Solo librería estándar de Go, un único binario, sin dependencias.

## Flujo

```bash
go build -o skopos ./cmd/skopos

# 1. Define el scope (una entrada por línea)
cat > scope.txt <<'EOF'
192.0.2.0/28
example.com
*.example.com
EOF

# 2. Escanea lo que esté autorizado e importa el XML
nmap -sV -oX scan1.xml 192.0.2.0/28
./skopos import nmap scan1.xml

# 3. Más tarde, otro escaneo: ves qué ha cambiado
./skopos import nmap -detect-closed scan2.xml

# 4. Informe
./skopos report -o informe.html
./skopos report -o activos.csv
```

## Reglas del scope

| Entrada | Significado |
|---|---|
| `192.0.2.10` | esa IP |
| `192.0.2.0/28` | rango CIDR (IPv4 o IPv6) |
| `example.com` | ese dominio exacto |
| `*.example.com` | cualquier subdominio (no incluye `example.com`) |

Un host se importa si su IP **o** alguno de sus nombres está en el scope. Lo demás no entra en el inventario y queda anotado como "fuera de scope" en el informe, como evidencia.

Comprobación rápida: `./skopos scope check 192.0.2.10 otro.com`.

## Cambios detectados

`new_host`, `new_service`, `changed_service` (producto o versión) y `closed_service` (solo con `-detect-closed`, úsalo únicamente si el nuevo escaneo cubre el mismo rango de puertos).

## Datos de engagement

`skopos.json` y `scope.txt` contienen datos del cliente: están en `.gitignore` y no deben subirse. El fichero se escribe con permisos `0600`.

## Tests

```bash
go test ./...
```

## Hoja de ruta

- Importadores: masscan/naabu (JSON), httpx, testssl.
- Check de DNS y correo (SPF/DKIM/DMARC) y subdominios huérfanos.
- Notas por activo y etiquetas (en scope, crítico, validado).
- Exportar al formato de hallazgos de Hacksolo.
