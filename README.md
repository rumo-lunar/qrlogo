# qrlogo

> Aesthetic QR codes that embed a logo or text by **construction**, not overlay — built on the linearity of Reed–Solomon over GF(2).

`qrlogo` generates QR codes whose dark/light modules *already* form your image, while still decoding to your URL on any standard scanner. This is the **QArt** technique (Russ Cox): instead of stamping a logo on top of a finished QR and hoping error-correction recovers the URL, the QR is *solved* so the desired modules come out the way you want.

```diagram
╭───────────╮   ╭────────────╮   ╭───────────────╮   ╭───────╮
│  /render  │──▶│ Target map │──▶│   /engine     │──▶│  PNG  │
│ image →   │   │  (B/W/?)   │   │ build system  │   ╰───────╯
│  pixels   │   ╰────────────╯   │   + solve     │
╰───────────╯                    ╰───────┬───────┘
                                         ▲
╭───────────╮   ╭────────────╮           │
│   /qr     │──▶│ Ghost grid │───────────╯
│  URL +    │   │  (linear   │
│  vars     │   │   forms)   │           ▲
╰───────────╯   ╰────────────╯           │
                              ╭──────────┴──────────╮
                              │      /bitset        │
                              │ GF(2) Gauss–Jordan  │
                              ╰─────────────────────╯
```

## Status

All four phases are implemented and tested.

- ✅ `/bitset`  — GF(2) Gauss–Jordan solver over `[]uint64` rows
- ✅ `/qr`      — symbolic QR encoder + concrete function-pattern bits (V11-M mask 2)
- ✅ `/render`  — text / image → 61×61 target map with halo
- ✅ `/engine`  — full pipeline + PNG output

> [!NOTE]
> A round-trip scannability test through a real QR decoder is still TODO. Until then, every commit is verified by:
> ```sh
> go vet -copylocks=false ./... && go test ./... -count=1
> ```

## Quick start

```go
package main

import (
    "image/png"
    "os"

    "github.com/rumo-lunar/qrlogo/engine"
    "github.com/rumo-lunar/qrlogo/render"
    _ "image/png"
)

func main() {
    // 1. Build a 61×61 target map from any image, glyph or hand-drawn matrix.
    f, _ := os.Open("logo.png")
    defer f.Close()
    src, _ := png.Decode(f)
    target := render.FromImage(src, 61, 61, render.ImageOptions{
        IgnoreTransparent: true,
    })
    render.ApplyHalo(target)

    // 2. Synthesize a V11-M QR symbol whose modules approximate the target.
    res, err := engine.Synthesize(engine.Options{
        URL:    "https://lunar.app",
        Target: target,
    })
    if err != nil { panic(err) }

    // 3. Write the PNG (default scale 8 px/module, quiet zone 4 modules).
    out, _ := os.Create("qrlogo.png")
    defer out.Close()
    _ = res.EncodePNG(out, engine.PNGOptions{})
}
```

`engine.Synthesize` is also happy with `Target: nil`, in which case it produces a plain V11-M QR symbol carrying the URL.

## Contract (v1 scope)

| Parameter        | Value                       |
|------------------|-----------------------------|
| QR version       | **11**                      |
| Module grid      | **61 × 61** (3 721 modules) |
| Error correction | **M** (Medium, ~15 %)       |
| Encoding mode    | **Byte**                    |
| Max URL length   | **100 bytes**               |
| Mask             | **2** (fixed)               |
| Use case         | **Screen display**          |
| Output           | PNG, 1 bit per module       |

Anything outside this contract (longer URLs, other QR versions, alphanumeric / Kanji, print or sticker robustness) is **out of scope for v1** and a v2 conversation.

## Capacity budget

At V11-M the spec gives:

- **254** data codewords — `1 × 50 + 4 × 51`
- **150** EC codewords — `5 × 30`
- **404** total codewords, **0** remainder bits

A 100-byte byte-mode payload uses:

```
   4 bits   mode indicator       (0100 = byte mode)
  16 bits   character count      (V ≥ 10 uses 16-bit length)
 800 bits   payload              (100 × 8)
   4 bits   terminator
────────
 824 bits   = 103 codewords
```

So the **free padding budget** is:

```
free padding codewords = 254 − 103 = 151
free padding bits      = 151 × 8   = 1208
```

These **1208 bits are the only true degrees of freedom**. Every other bit in the matrix — including all 150 × 8 = 1200 EC bits — is a fixed linear function of the 1208 free bits plus the URL bits. A 61 × 61 grid has 3 721 modules, so we cannot control every pixel; legibility comes from being deliberate about *which* modules we constrain.

## The math

### Why Reed–Solomon is linear over GF(2)

QR codes use Reed–Solomon over `GF(256) = GF(2)[x] / (x⁸ + x⁴ + x³ + x² + 1)`. The encoder treats data codewords as polynomial coefficients, multiplies by `xᵏ`, and divides by a fixed generator polynomial — the EC codewords are the remainder.

All of those operations are **linear** in the input bytes. Each byte is an 8-dim vector over GF(2); multiplication by any fixed GF(256) element is a fixed 8 × 8 GF(2) matrix. Polynomial division is built from such multiplications and XORs.

So every output bit `b` of the data + EC stream is

```
b = c ⊕ x_{i₁} ⊕ x_{i₂} ⊕ … ⊕ x_{iₘ}
```

where `c ∈ {0, 1}` is the contribution of the URL bits and the `x_{iⱼ}` are free padding bits. In code (`/qr/sym`), every ghost module is a `Bit{Vars []uint64, Const byte}`.

### Masking is constant

Data masks XOR a fixed boolean pattern over the data region. In our system this only flips the `Const` term — no new variables, no nonlinearity. Mask 2 is `col mod 3 == 0`.

### From image to equations

For every `Black`/`White` cell `(x, y)` we emit one GF(2) row:

```
x_{i₁} ⊕ … ⊕ x_{iₘ} = wantBit ⊕ Const
```

`DontCare` cells emit nothing. Function-pattern cells (finders, timing, alignment, format info, version info, dark module) are spec-fixed and silently skipped — `Stats.FunctionConflicts` reports how many target cells landed on a wrong-polarity function bit.

### The solver

Gauss–Jordan over GF(2), `[]uint64`-row XORs:

1. **Forward elimination.** Per pivot column, find a row with a 1 there and XOR it into every other row with a 1 in that column. Row XOR is `O(n / 64)` `uint64` ops.
2. **Consistency check.** A row of the form `0 = 1` means the constraints over-ran the budget → inconsistent.
3. **Back-substitution.** Each pivot row yields one variable; unpivoted variables default to 0.

The 1208-bit solution feeds back into the symbolic forms (`sym.ResolveBit`) to give the concrete 61 × 61 module grid.

## Halo

Without a halo, the QR solver is free to paint a dark module immediately next to an intended dark logo pixel — the logo blurs into the random data. `render.ApplyHalo` flips every `DontCare` cell that is 8-adjacent to a `Black` into `White`, giving the logo a guaranteed 1-cell light outline. Halos are strictly 1 cell wide (computed from a snapshot so they don't chain).

## Design choices

| Decision                | Choice          | Rationale                                                                                              |
|-------------------------|-----------------|--------------------------------------------------------------------------------------------------------|
| QR version              | V11 (61 × 61)   | Big enough for a recognisable image; small enough to stay under 4 000 modules.                          |
| Error-correction level  | M               | 254 data codewords → 1208 free bits at 100-byte URL. Q/H crush the budget; L is needlessly fragile.    |
| Encoding mode           | Byte            | URLs contain `:/?#=&` — alphanumeric mode rejects them.                                                |
| Max URL length          | 100 bytes       | Locks the free-bit budget at design time so we never run out mid-solve.                                |
| Mask                    | Fixed mask 2    | Re-solving for every mask multiplies work ×8 without changing capacity. Fixed mask = deterministic.    |
| Halo                    | 8-neighbour     | Logo legibility against random data modules. 1-cell wide so cost stays linear in image perimeter.      |
| Output                  | PNG via stdlib  | `image`, `image/color`, `image/png`. Text rendering uses `golang.org/x/image/{font,math/fixed}`.       |
| Variable layout         | Free padding only | URL bits are constants. Only the 1208 padding bits are variables. EC bits are linear combinations.   |

## Project layout

```
qrlogo/
├── bitset/         GF(2) Gauss–Jordan solver
├── qr/             V11-M symbolic encoder + function-pattern bits
│   ├── gf256/       GF(256) field arithmetic
│   └── sym/         linear-form Bit and Byte over GF(2)
├── render/         text/image → 61×61 target map + halo
├── engine/         pipeline + PNG output
├── go.mod
└── README.md
```

## References

- Russ Cox, *QArt Codes* — <https://research.swtch.com/qart>
- ISO/IEC 18004:2015 — QR code symbology specification
- Thonky, *QR Code Tutorial* — <https://www.thonky.com/qr-code-tutorial/>
- `rsc.io/qr` — Russ Cox's reference Go QR implementation
- `github.com/makiuchi-d/gozxing` — Go port of ZXing, candidate for the future round-trip decode test
