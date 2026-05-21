<div align="center">
  <img src="assets/logo.png" alt="qrlogo" width="160" />

  # qrlogo

  **Plain QR codes with rounded finder patterns and an optional centred logo.**

  Versions 1 – 40, error-correction levels L/M/Q/H, byte mode.
</div>

---

`qrlogo` is a small, dependency-light Go library and CLI for generating QR codes that look at home next to a brand mark — rounded finder patterns by default, and an optional logo painted in the centre. The QR itself is standards-conformant (ISO/IEC 18004); the logo is overlaid on top of the rendered symbol and tolerated by the error-correction budget.

## Features

- **Versions 1 – 40, EC levels L / M / Q / H.** Auto-fits the smallest version that holds your payload at the chosen EC level.
- **Rounded finder patterns** by default. Opt out with `-rounded-finders=false`.
- **Centred logo overlay.** PNG / JPEG / GIF, configurable coverage and padding. No QR modules are cleared — the EC budget absorbs the obscured cells.
- **Penalty-based mask selection** per ISO/IEC 18004 §7.8.3. All eight masks are scored; the lowest-penalty one wins.
- **Zero runtime dependencies** outside the Go standard library and `golang.org/x/image` (used for high-quality logo scaling).

## How it works

```diagram
╭─────────╮   ╭──────────╮   ╭──────────╮   ╭──────────╮   ╭───────╮
│ payload │──▶│  encode  │──▶│ RS + int │──▶│ place +  │──▶│  PNG  │
│  + ec   │   │ (frame + │   │ erleave  │   │   mask   │   │       │
│ + ver?  │   │ padding) │   │          │   │  select  │   │       │
╰─────────╯   ╰──────────╯   ╰────┬─────╯   ╰────┬─────╯   ╰───┬───╯
                                  │              │             │
                                  ▼              ▼             ▼
                            ╭──────────╮   ╭──────────╮   ╭────────╮
                            │ qr/spec  │   │  qr/qr   │   │ engine │
                            │ (tables) │   │ (matrix) │   │ (paint)│
                            ╰──────────╯   ╰──────────╯   ╰────────╯
```

The encoder walks the standard pipeline: byte-mode framing → 0xEC/0x11 padding → Reed–Solomon per RS block → column-major interleave → zig-zag placement → 8-mask penalty scoring. The renderer paints modules as squares (or rounded shapes for the three finder patterns) and optionally composites a logo on top.

## Requirements

- Go 1.25 or later

## Install

```sh
go install github.com/rumo-lunar/qrlogo/cmd/qrlogo@latest
```

…or clone and build locally:

```sh
git clone https://github.com/rumo-lunar/qrlogo.git
cd qrlogo
go build ./cmd/qrlogo
```

## Quick start (CLI)

```sh
# Smallest QR that fits, EC level H, rounded finders.
qrlogo -url "https://lunar.app" -out qr.png

# Pin the version explicitly.
qrlogo -url "https://lunar.app" -version 10 -ec M -out qr.png

# Add a centred logo (painted on top of the QR; EC absorbs the loss).
qrlogo -url "https://lunar.app" -image assets/logo.png -logo-coverage 0.20 -out qr.png

# Square finders if you want the classic look.
qrlogo -url "https://lunar.app" -rounded-finders=false -out qr.png
```

### CLI flags

| Flag                | Default      | Description                                                    |
|---------------------|--------------|----------------------------------------------------------------|
| `-url`              | *(required)* | Byte-mode payload.                                             |
| `-ec`               | `H`          | Error-correction level: `L`, `M`, `Q` or `H`.                  |
| `-version`          | `0` (auto)   | QR version 1 – 40; `0` auto-fits the smallest version.         |
| `-image`            | `""`         | Optional logo image (PNG / JPEG / GIF).                        |
| `-logo-coverage`    | `0.18`       | Logo box width as a fraction of the QR width, in `(0, 1]`.     |
| `-logo-padding`     | `0.10`       | Background padding around the logo, as fraction of box width.  |
| `-rounded-finders`  | `true`       | Render the three finder patterns with rounded corners.         |
| `-scale`            | `8`          | Pixels per QR module.                                          |
| `-quiet`            | `4`          | Quiet-zone modules around the symbol.                          |
| `-out`              | `qrlogo.png` | Output PNG path (`-` for stdout).                              |

Exit codes:

| Code | Meaning                                                          |
|------|------------------------------------------------------------------|
| `0`  | Success.                                                         |
| `1`  | Invalid arguments (missing `-url`, bad EC level, …).             |
| `2`  | Invalid input (unreadable / undecodable image).                  |
| `3`  | Encoding failed (payload too large for the chosen version / EC). |
| `4`  | Output write failed.                                             |

> [!TIP]
> Logo coverage past about `0.25` starts to defeat even EC level H. `qrlogo` prints a warning to stderr but still produces the PNG — verify with a real scanner before shipping.

## Quick start (library)

```go
package main

import (
    "image/png"
    "os"

    "github.com/rumo-lunar/qrlogo/engine"
    "github.com/rumo-lunar/qrlogo/qr/spec"
)

func main() {
    // Open an optional logo.
    f, _ := os.Open("logo.png")
    defer f.Close()
    logo, _ := png.Decode(f)

    // Encode the QR (auto-fit version, EC level H by default).
    res, err := engine.Encode(engine.Options{
        URL: "https://lunar.app",
        EC:  spec.ECHigh,
    })
    if err != nil {
        panic(err)
    }

    // Render to PNG with a centred logo overlay.
    out, _ := os.Create("qr.png")
    defer out.Close()
    _ = res.EncodePNG(out, engine.PNGOptions{
        Scale:        8,
        QuietZone:    4,
        Logo:         logo,
        LogoCoverage: 0.20,
        LogoPadding:  0.10,
    })
}
```

> [!NOTE]
> `engine.Encode` accepts an empty `Logo` (no overlay) and `Version == 0` (auto-fit to the smallest version that holds the URL at the chosen EC level).

## Contract

| Parameter          | Value                                          |
|--------------------|------------------------------------------------|
| QR versions        | **1 – 40**                                     |
| Error correction   | **L / M / Q / H**                              |
| Encoding mode      | **Byte**                                       |
| Mask               | Penalty-selected (ISO/IEC 18004 §7.8.3)        |
| Module rendering   | Square modules; rounded finder patterns        |
| Logo overlay       | Painted on top; no modules cleared             |
| Output             | PNG (8-bit RGB)                                |

Alphanumeric / numeric / Kanji modes, structured-append symbols, micro-QR and print-noise robustness are out of scope.

## Project layout

```
qrlogo/
├── qr/
│   ├── gf256/      GF(256) field arithmetic
│   ├── spec/       per-(version, EC) constants & tables (V1 – V40, L/M/Q/H)
│   ├── encode.go   byte-mode framing, 0xEC/0x11 padding
│   ├── rs.go       Reed–Solomon encoder
│   ├── module.go   function-pattern Kind map
│   ├── function.go finder / timing / alignment / dark / version-info placement
│   ├── place.go    zig-zag data placement
│   ├── mask.go     8 data masks + penalty-based selection
│   ├── penalty.go  ISO/IEC 18004 §7.8.3 penalty scoring
│   └── build.go    public top-level Build()
├── engine/
│   ├── engine.go   public Encode(): autofit + assemble Spec + run qr.Build
│   ├── render.go   internal: rounded-rect rasteriser + finder + logo composite
│   └── png.go      public EncodePNG() + PNGOptions
└── cmd/qrlogo/     CLI entry point
```

## Development

Every commit is verified by:

```sh
go vet -copylocks=false ./... && go test ./... -count=1
```

> [!NOTE]
> A round-trip scannability test through a real QR decoder (e.g. [`github.com/makiuchi-d/gozxing`](https://github.com/makiuchi-d/gozxing)) is recommended before relying on `qrlogo` output in production. Structural tests cover dimensions and function patterns, but they will not catch subtle bit-ordering bugs in mask, format-info or version-info placement.

## References

- ISO/IEC 18004:2015 — QR code symbology specification
- Thonky — [*QR Code Tutorial*](https://www.thonky.com/qr-code-tutorial/)
- [`rsc.io/qr`](https://pkg.go.dev/rsc.io/qr) — Russ Cox's reference Go QR implementation
- [`github.com/makiuchi-d/gozxing`](https://github.com/makiuchi-d/gozxing) — Go port of ZXing, candidate for a round-trip decode test
