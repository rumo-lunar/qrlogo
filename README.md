# qrlogo

An aesthetic QR code generator that embeds a binary image (logo or text) into a scannable QR code by exploiting the linearity of Reed‑Solomon error correction over GF(2).

This is the **QArt** technique, originally described by Russ Cox: instead of overlaying a logo on a finished QR code (and praying that error correction recovers the URL), the QR code is *constructed* so that selected modules already form the desired image — and the resulting code is still a perfectly valid QR symbol.

---

## What it does

Given:

- a URL (up to 100 characters), and
- a binary image to embed (logo or rasterised text),

`qrlogo` produces a PNG of a valid QR code whose dark/light module pattern approximates the image, while still decoding to the URL on any standard scanner.

---

## Contract (v1 scope)

| Parameter          | Value                          |
|--------------------|--------------------------------|
| QR version         | **11**                         |
| Module grid        | **61 × 61** (3 721 modules)    |
| Error correction   | **M** (Medium, ~15 %)          |
| Encoding mode      | **Byte**                       |
| Max URL length     | **100 characters**             |
| Use case           | **Screen display**             |
| Output             | PNG, 1 bit per module          |
| Mask               | Fixed (mask 2 — see *Choices*) |

Anything outside this contract (longer URLs, other QR versions, alphanumeric mode, Kanji, print/sticker robustness) is **out of scope for v1**. Widening the contract is a v2 conversation.

---

## Capacity budget

At V11‑M the QR spec gives:

- **254** data codewords, split as **1 block of 50 + 4 blocks of 51** data codewords
- **150** EC codewords (5 blocks × 30 EC codewords per block)
- **404** total codewords; 0 remainder bits

A 100‑character byte‑mode payload encodes as:

```
   4 bits   mode indicator       (0100 = byte mode)
  16 bits   character count      (V≥10 uses 16-bit length)
 800 bits   payload              (100 × 8)
   4 bits   terminator
────────
 824 bits   = 103 codewords used by the URL message
```

So the free padding region is:

```
free padding codewords = 254 − 103 = 151
free padding bits      = 151 × 8   = 1208
```

These **1208 bits are the only true degrees of freedom** in the system. Every other bit in the matrix — including all 150 × 8 = 1200 EC bits — is a fixed linear function of these 1208 free bits plus the URL bits.

**Implication:** the maximum number of image pixels we can pin is bounded by the rank of the equation system, which cannot exceed 1208. A 61 × 61 grid has 3 721 modules, so we cannot control every pixel. Legibility of the embedded image comes from being deliberate about *which* modules we constrain (image core first, edge halo only if budget allows).

---

## How it works

Four‑stage pipeline:

```diagram
╭───────────╮   ╭────────────╮   ╭───────────────╮   ╭─────────╮
│  /render  │──▶│ Target map │──▶│   /engine     │──▶│   PNG   │
│  image →  │   │  (B/W/?)   │   │  build system │   ╰─────────╯
│  pixels   │   ╰────────────╯   │  + solve      │
╰───────────╯                    ╰───────┬───────┘
                                         ▲
╭───────────╮   ╭────────────╮           │
│   /qr     │──▶│ Ghost grid │───────────╯
│  URL +    │   │  (linear   │
│  vars     │   │   forms)   │
╰───────────╯   ╰────────────╯           ▲
                                         │
                              ╭──────────┴──────────╮
                              │      /bitset        │
                              │ GF(2) Gauss–Jordan  │
                              ╰─────────────────────╯
```

1. **`/render`** rasterises the input image into a 61 × 61 grid of `{Black, White, DontCare}` cells.
2. **`/qr`** builds a *symbolic* QR matrix. Function patterns (finders, separators, timing, alignment, format info, version info, dark module) are placed as constants. Every other module carries a **linear form** over the 1208 free padding bits — that is, the set of free‑bit indices that XOR into it, plus a constant offset contributed by the URL bits. Mask 2 is applied symbolically (it only toggles the constant offset).
3. **`/engine`** walks the target map and the ghost grid together. For each `Black` or `White` cell at `(x, y)`, it emits the linear equation `ghost[x][y] = target_bit` and appends it to the system. `DontCare` cells emit nothing.
4. **`/bitset`** solves the system using Gauss–Jordan elimination over GF(2), implemented with `[]uint64` bitsets and bitwise XOR. The solved padding bits are then plugged back into a standard QR encoder to produce the final PNG via `image/png`.

If the system is **inconsistent** (image asks for more than 1208 bits can satisfy), the engine drops the lowest‑priority constraints and retries — see *Fallback* below.

---

## The math

### Why Reed–Solomon is linear over GF(2)

QR codes use Reed–Solomon over `GF(256) = GF(2)[x]/(x⁸ + x⁴ + x³ + x² + 1)`. The encoder treats the data codewords as the coefficients of a polynomial, multiplies by `xᵏ`, and divides by a fixed generator polynomial; the EC codewords are the remainder.

All of these operations are **linear** in the input bytes. Each byte is an 8‑dimensional vector over GF(2); multiplication by any fixed element of GF(256) is a fixed 8 × 8 matrix over GF(2). Polynomial division is built from such multiplications and additions.

Therefore, every output bit `b` of the data + EC stream can be written as

```
b = c ⊕ x_{i₁} ⊕ x_{i₂} ⊕ … ⊕ x_{iₘ}
```

where:
- `c ∈ {0, 1}` is the contribution of the forced URL bits,
- `x_{iⱼ}` are free padding bits.

In code, each ghost module is represented as `(bitset of variable indices, constant target)`.

### Masking

QR masks XOR a fixed boolean pattern over the data region. In our equation system this is a constant — it just flips the `c` term:

```
b' = b ⊕ mask(x, y)
```

No new variables, no nonlinearity.

### From image to equations

For each pinned cell `(x, y)` with desired colour `t ∈ {0, 1}`, we get one linear equation over GF(2):

```
x_{i₁} ⊕ … ⊕ x_{iₘ} = t ⊕ c
```

i.e. a row of a sparse `m × 1208` matrix over GF(2) with right‑hand side `t ⊕ c`.

### The solver

Standard Gauss–Jordan elimination over GF(2):

1. **Forward elimination.** For each pivot column, find a row with a 1 there; XOR it into every other row that also has a 1 in that column. (Row XOR = a single bitset XOR, which is `O(n / 64)` `uint64` ops.)
2. **Consistency check.** Any row of the form `0 = 1` means the image overran the budget → inconsistent.
3. **Back‑substitution.** Each pivot row trivially yields one variable; unpivoted variables are free (set to 0).

The result is a vector of 1208 bits, which feeds into the standard QR encoder as hardcoded padding.

---

## Design choices (and their trade‑offs)

| Decision                          | Choice                | Rationale                                                                                                                                                       |
|-----------------------------------|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **QR version**                    | V11 (61 × 61)         | Large enough for a recognisable image; small enough to stay under 5 000 modules. V10 was too tight on free bits, V12+ adds modules we don't need.               |
| **Error‑correction level**        | M                     | Gives 254 data codewords → 1208 free bits at 100‑char URL. Higher EC (Q/H) crushes the budget; lower EC (L) is more fragile on screen and unnecessary here.     |
| **Encoding mode**                 | Byte                  | URLs contain `:/?#=&` and lowercase letters — alphanumeric mode would refuse them. Byte mode is universal at the cost of 1 bit/char overhead.                   |
| **Max URL length**                | 100 chars             | Locks the free‑bit budget at design time so we never run out mid‑solve. Anything shorter just gives us more budget.                                             |
| **Mask**                          | Fixed mask 2          | The QR spec accepts any of 8 masks. Re‑solving the system for every mask multiplies work by 8 without changing capacity. Fixed mask keeps the solver deterministic. |
| **Halo around image edges**       | Cardinal neighbours only | Forcing all 8 neighbours of every black pixel to white roughly doubles the constraint count. 4‑neighbour halo gives sufficient contrast without doubling cost. |
| **Output format**                 | PNG (stdlib only)     | `image`, `image/color`, `image/png` — no rendering deps. Text rendering uses `golang.org/x/image/font` + `golang.org/x/image/math/fixed`.                       |
| **Fallback when over‑constrained**| Priority‑ranked drop  | Each constraint is tagged with a priority (image core = high, halo = low). On `inconsistent`, drop the lowest‑priority rows and re‑solve. Deterministic, debuggable. |
| **Variable layout**               | All free padding bits | Forced URL bits are constants. Only the 1208 padding bits are variables. EC bits are linear combinations, never variables themselves.                            |

---

## Project layout

```
qrlogo/
├── bitset/         Phase 1: GF(2) linear algebra (Equation, System, Solve)
├── qr/             Phase 2: symbolic QR encoder ("ghost" matrix)
├── render/         Phase 3: image / text → target grid + halo
├── engine/         Phase 4: pipeline, priority fallback, final PNG render
├── cmd/qrlogo/     CLI entry point
├── go.mod
└── README.md
```

---

## Status

🚧 **Pre‑implementation.** This README is the design contract; code follows in four phases.

- [ ] **Phase 1 — `/bitset`** — GF(2) Gauss–Jordan solver with `[]uint64` rows. Unit tests on small known systems, contradictory systems, and underdetermined systems.
- [ ] **Phase 2 — `/qr`** — Symbolic QR encoder. Function‑pattern map, codeword zig‑zag traversal, RS over GF(256) with parallel variable tracking via the 8 × 8 GF(2) bridge, mask 2.
- [ ] **Phase 3 — `/render`** — Image / text → 61 × 61 grid, 4‑neighbour halo, priority tags.
- [ ] **Phase 4 — `/engine`** — Integration, priority fallback, final PNG.
- [ ] **CLI + round‑trip test** through a real decoder (e.g. `github.com/makiuchi-d/gozxing`) to prove every generated PNG actually decodes to the input URL.

---

## References

- Russ Cox, *QArt Codes* — <https://research.swtch.com/qart>
- ISO/IEC 18004:2015 — QR code symbology specification
- Thonky, *QR Code Tutorial* — <https://www.thonky.com/qr-code-tutorial/>
- `rsc.io/qr` — Russ Cox's reference Go QR implementation
- `github.com/makiuchi-d/gozxing` — Go port of ZXing, useful for round‑trip decode tests

---

## License

TBD.
