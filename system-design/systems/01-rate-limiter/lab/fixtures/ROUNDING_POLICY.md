# Runner rounding policy

All runner outputs are quantized to six decimal places from the actual finite IEEE-754 binary
value produced by the algorithm. Quantization multiplies the magnitude by `1_000_000`, rounds an
exact half away from zero, then divides by `1_000_000`. It does not add epsilon or another hidden
tolerance before rounding.

This makes the policy identical in Python (`float`), Go (`float64`), JavaScript (`Number`), and Java (`double`). A
decimal-looking input may already be slightly above or below a mathematical half after JSON is
converted to binary. The shared `token-bucket-ieee754-quantization-edge` fixture records that
behavior explicitly, so parity never depends on language-specific decimal or epsilon handling.
