#!/usr/bin/env python3
"""Generates the looping ocean-ambience WAV (client/assets/audio/waves.wav).

Layered low-passed noise with slow amplitude swells whose periods divide the
loop length, so the loop point lands in a trough and the seam is inaudible.
Pure stdlib — run from the repo root:

    python3 client/tools/gen_waves.py
"""

import math
import random
import struct
import wave

RATE = 44100
SECONDS = 24
FRAMES = RATE * SECONDS

random.seed(1715)


def render_channel(phase_offset: float) -> list[float]:
    samples = []
    brown = 0.0
    lp = 0.0
    for i in range(FRAMES):
        t = i / RATE
        # Brown-ish noise: integrate white noise, leak so it can't wander off.
        brown += random.uniform(-1.0, 1.0) * 0.08
        brown *= 0.997
        # Gentle low-pass: surf rumble, not hiss.
        lp += (brown - lp) * 0.12
        # Swell envelope: 8s and 12s periods both divide 24s, and the -pi/2
        # phase puts a trough exactly at the loop boundary.
        swell = 0.6 * math.sin(math.tau * t / 8.0 - math.pi / 2 + phase_offset) \
            + 0.4 * math.sin(math.tau * t / 12.0 - math.pi / 2)
        env = 0.5 + 0.35 * swell
        samples.append(lp * env)
    return samples


def main() -> None:
    left = render_channel(0.0)
    right = render_channel(0.7)  # decorrelated swells -> wide stereo sea

    # Normalize to a comfortable ambience level.
    peak = max(max(abs(s) for s in left), max(abs(s) for s in right))
    gain = 0.55 / peak

    out = wave.open("client/assets/audio/waves.wav", "wb")
    out.setnchannels(2)
    out.setsampwidth(2)
    out.setframerate(RATE)
    frames = bytearray()
    for l, r in zip(left, right):
        frames += struct.pack(
            "<hh",
            int(max(-1.0, min(1.0, l * gain)) * 32767),
            int(max(-1.0, min(1.0, r * gain)) * 32767),
        )
    out.writeframes(bytes(frames))
    out.close()
    print(f"waves.wav: {SECONDS}s loop, {len(frames)} bytes")


if __name__ == "__main__":
    main()
