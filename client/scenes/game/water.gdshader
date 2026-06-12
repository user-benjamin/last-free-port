// Animated open-water shader for the cove. Two octaves of scrolling value
// noise blended between deep and shallow tones, with brightened crests.
// Applied to the full-screen-and-beyond Water ColorRect; the island sprite
// (with its baked foam rim) sits on top.
shader_type canvas_item;

uniform vec3 deep_color: source_color = vec3(0.02, 0.12, 0.16);
uniform vec3 shallow_color: source_color = vec3(0.05, 0.24, 0.27);
uniform vec2 wave_scale = vec2(160.0, 140.0);

float hash(vec2 p) {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}

float vnoise(vec2 p) {
	vec2 i = floor(p);
	vec2 f = fract(p);
	f = f * f * (3.0 - 2.0 * f);
	return mix(
		mix(hash(i), hash(i + vec2(1.0, 0.0)), f.x),
		mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), f.x),
		f.y
	);
}

void fragment() {
	vec2 uv = UV * wave_scale;
	float n = vnoise(uv + vec2(TIME * 0.5, TIME * 0.25)) * 0.6
		+ vnoise(uv * 2.3 - vec2(TIME * 0.3, TIME * 0.4)) * 0.4;
	vec3 col = mix(deep_color, shallow_color, n);
	float crest = smoothstep(0.74, 0.95, n);
	col += crest * vec3(0.08, 0.11, 0.12);
	COLOR = vec4(col, 1.0);
}
