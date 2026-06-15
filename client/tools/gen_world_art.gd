extends SceneTree

## Generates the cove's placeholder-but-presentable art: an island ground
## texture with a noisy coastline, a 4-direction animated pirate sprite
## sheet, and a dock. Deterministic (seeded). All of it is meant to be
## replaceable by real art later without touching scene code — keep the
## filenames.
##
## Run from the repo root:
##   godot --headless --path client -s res://tools/gen_world_art.gd

const CX := 640.0
const CY := 360.0
const RX := 560.0
const RY := 310.0

var _rng := RandomNumberGenerator.new()
var _ph1: float
var _ph2: float
var _ph3: float

func _init() -> void:
	_rng.seed = 1715
	_ph1 = _rng.randf_range(0.0, TAU)
	_ph2 = _rng.randf_range(0.0, TAU)
	_ph3 = _rng.randf_range(0.0, TAU)

	_gen_island()
	_gen_pirate_sheet()
	_gen_dock()
	_gen_driftwood()
	quit()

# --- island ----------------------------------------------------------------

## Coastline radius multiplier around the island — low-frequency sine
## stack gives a wobbly, natural-looking shore instead of an ellipse.
func _coast(theta: float) -> float:
	return 1.0 + 0.14 * sin(3.0 * theta + _ph1) \
		+ 0.09 * sin(7.0 * theta + _ph2) \
		+ 0.05 * sin(11.0 * theta + _ph3)

## Normalized distance from island center: <1 is land, <0.82 is grassy
## interior, 0.82..1 is beach.
func _island_d(x: float, y: float) -> float:
	var dx := (x - CX) / RX
	var dy := (y - CY) / RY
	var d := sqrt(dx * dx + dy * dy)
	return d / _coast(atan2(dy, dx))

func _gen_island() -> void:
	var w := 1280
	var h := 720
	var img := Image.create(w, h, false, Image.FORMAT_RGBA8)
	img.fill(Color(0, 0, 0, 0))

	var sand := Color(0.85, 0.75, 0.54)
	var wet_sand := Color(0.72, 0.6, 0.42)
	var grass := Color(0.37, 0.55, 0.29)
	var grass_dark := Color(0.30, 0.46, 0.24)

	for y in h:
		for x in w:
			var nd := _island_d(x, y)
			if nd >= 1.015:
				continue
			var c: Color
			if nd >= 1.0:
				# Foam rim where land meets animated water.
				c = Color(0.95, 0.97, 0.94, (1.015 - nd) / 0.015 * 0.55)
			elif nd >= 0.96:
				c = wet_sand
			elif nd >= 0.82:
				c = wet_sand.lerp(sand, clampf((0.96 - nd) / 0.10, 0.0, 1.0))
			else:
				c = grass.lerp(grass_dark, _rng.randf() * 0.5) if _rng.randf() < 0.06 else grass
				# Soft transition from beach into grass.
				if nd > 0.78:
					c = c.lerp(sand, (nd - 0.78) / 0.04 * 0.7)
			# Per-pixel speckle keeps large fills from looking flat.
			var sp := (_rng.randf() - 0.5) * 0.05
			img.set_pixel(x, y, Color(c.r + sp, c.g + sp, c.b + sp, c.a))

	_draw_path(img)
	_draw_rocks(img)
	_draw_palms(img)

	var err := img.save_png("res://assets/island.png")
	print("island.png: ", error_string(err))

## Dirt path from the island center toward the dock on the east shore.
func _draw_path(img: Image) -> void:
	for i in 600:
		var t := i / 600.0
		var px := lerpf(CX, 1195.0, t) + sin(t * 9.0) * 26.0
		var py := lerpf(CY, 330.0, t) + cos(t * 7.0) * 18.0
		for oy in range(-4, 5):
			for ox in range(-4, 5):
				if ox * ox + oy * oy > 18:
					continue
				var x := int(px) + ox
				var y := int(py) + oy
				if x < 0 or x >= img.get_width() or y < 0 or y >= img.get_height():
					continue
				if _island_d(x, y) < 0.95 and _rng.randf() < 0.8:
					var shade := (_rng.randf() - 0.5) * 0.05
					img.set_pixel(x, y, Color(0.62 + shade, 0.5 + shade, 0.34 + shade))

func _draw_rocks(img: Image) -> void:
	for cluster in 7:
		var cx := 0.0
		var cy := 0.0
		while true:
			cx = _rng.randf_range(100, 1180)
			cy = _rng.randf_range(80, 640)
			var nd := _island_d(cx, cy)
			if nd > 0.55 and nd < 0.93:
				break
		for blob in _rng.randi_range(2, 3):
			var bx := cx + _rng.randf_range(-10, 10)
			var by := cy + _rng.randf_range(-6, 6)
			var r := _rng.randf_range(3.0, 6.0)
			for oy in range(-int(r) - 1, int(r) + 2):
				for ox in range(-int(r) - 1, int(r) + 2):
					var d := Vector2(ox, oy).length()
					if d > r:
						continue
					var base := 0.52 + (0.12 if (ox < 0 and oy < 0) else 0.0) - d / r * 0.1
					_blend(img, int(bx) + ox, int(by) + oy, Color(base, base, base * 0.97), 1.0)

func _draw_palms(img: Image) -> void:
	var trunk := Color(0.45, 0.33, 0.2)
	var frond_a := Color(0.25, 0.46, 0.22)
	var frond_b := Color(0.32, 0.56, 0.27)
	for tree in 16:
		var tx := 0.0
		var ty := 0.0
		while true:
			tx = _rng.randf_range(120, 1160)
			ty = _rng.randf_range(100, 620)
			if _island_d(tx, ty) < 0.74:
				break
		# Ground shadow first.
		for oy in range(-4, 5):
			for ox in range(-12, 13):
				if (ox * ox) / 144.0 + (oy * oy) / 16.0 <= 1.0:
					_blend(img, int(tx) + ox, int(ty) + oy, Color(0, 0, 0), 0.22)
		# Curved trunk.
		var lean := _rng.randf_range(-0.35, 0.35)
		var top := Vector2(tx, ty)
		for i in 16:
			top = Vector2(tx + lean * i, ty - i)
			for wpx in 2:
				_blend(img, int(top.x) + wpx, int(top.y), trunk, 1.0)
		# Fronds radiating from the crown, drooping at the tips.
		for f in 7:
			var ang := TAU * f / 7.0 + _rng.randf_range(-0.2, 0.2)
			var dir := Vector2(cos(ang), sin(ang) * 0.6)
			for step in 9:
				var p := top + dir * step + Vector2(0, step * step * 0.06)
				var col := frond_a if f % 2 == 0 else frond_b
				_blend(img, int(p.x), int(p.y), col, 1.0)
				if step < 6:
					_blend(img, int(p.x), int(p.y) + 1, col, 0.7)

func _blend(img: Image, x: int, y: int, c: Color, a: float) -> void:
	if x < 0 or x >= img.get_width() or y < 0 or y >= img.get_height():
		return
	var base := img.get_pixel(x, y)
	if base.a == 0.0:
		return # don't paint on open water
	img.set_pixel(x, y, base.lerp(Color(c.r, c.g, c.b, base.a), a))

# --- pirate sprite sheet ----------------------------------------------------

## 16x16 frames as ASCII art: rows of 16 chars, one char per pixel.
## Sheet layout: rows = down/up/left/right, cols = walk frame 0/1.
## "right" is generated by mirroring "left".

const PALETTE := {
	"R": Color(0.78, 0.18, 0.16), # bandana
	"r": Color(0.55, 0.12, 0.11), # bandana band
	"S": Color(0.87, 0.66, 0.48), # skin
	"E": Color(0.12, 0.10, 0.09), # eye
	"W": Color(0.92, 0.90, 0.82), # shirt
	"w": Color(0.72, 0.70, 0.62), # shirt shade
	"B": Color(0.20, 0.14, 0.10), # belt / boots
	"N": Color(0.23, 0.28, 0.38), # pants
}

const DOWN_0 := [
	"................",
	"......RRRR......",
	".....RRRRRR.....",
	".....rrrrrr.....",
	".....SSSSSS.....",
	".....SESSES.....",
	".....SSSSSS.....",
	"......WWWW......",
	"....SWWWWWWS....",
	"....SWwWWwWS....",
	".....BBBBBB.....",
	".....NN..NN.....",
	".....NN..NN.....",
	".....BB..BB.....",
	"................",
	"................",
]

const DOWN_1 := [
	"......RRRR......",
	".....RRRRRR.....",
	".....rrrrrr.....",
	".....SSSSSS.....",
	".....SESSES.....",
	".....SSSSSS.....",
	"......WWWW......",
	"....SWWWWWWS....",
	"....SWwWWwWS....",
	".....BBBBBB.....",
	".....NNNNNN.....",
	"......NNNN......",
	"......B..B......",
	"................",
	"................",
	"................",
]

const UP_0 := [
	"................",
	"......RRRR......",
	".....RRRRRR.....",
	".....rrrrrr.....",
	".....RRRRRR.....",
	".....RRRRrr.....",
	".....SSSSSS.....",
	"......WWWW......",
	"....SWWWWWWS....",
	"....SWwWWwWS....",
	".....BBBBBB.....",
	".....NN..NN.....",
	".....NN..NN.....",
	".....BB..BB.....",
	"................",
	"................",
]

const UP_1 := [
	"......RRRR......",
	".....RRRRRR.....",
	".....rrrrrr.....",
	".....RRRRRR.....",
	".....RRRRrr.....",
	".....SSSSSS.....",
	"......WWWW......",
	"....SWWWWWWS....",
	"....SWwWWwWS....",
	".....BBBBBB.....",
	".....NNNNNN.....",
	"......NNNN......",
	"......B..B......",
	"................",
	"................",
	"................",
]

const LEFT_0 := [
	"................",
	"......RRRR......",
	".....RRRRRR.....",
	".....rrrrrr.....",
	".....SSSSS......",
	".....ESSSS......",
	".....SSSSS......",
	"......WWWW......",
	".....SWWWW......",
	".....SWwWW......",
	"......BBBB......",
	"......NNNN......",
	"......NN........",
	"......BB........",
	"................",
	"................",
]

const LEFT_1 := [
	"......RRRR......",
	".....RRRRRR.....",
	".....rrrrrr.....",
	".....SSSSS......",
	".....ESSSS......",
	".....SSSSS......",
	"......WWWW......",
	".....SWWWW......",
	".....SWwWW......",
	"......BBBB......",
	"......NNNN......",
	".....NN..N......",
	".....B....B.....",
	"................",
	"................",
	"................",
]

func _frame_image(rows: Array, palette: Dictionary) -> Image:
	var img := Image.create(16, 16, false, Image.FORMAT_RGBA8)
	img.fill(Color(0, 0, 0, 0))
	for y in 16:
		var row: String = rows[y]
		for x in 16:
			var ch := row[x]
			if palette.has(ch):
				img.set_pixel(x, y, palette[ch])
	return img

func _gen_sheet(palette: Dictionary, path: String) -> void:
	var sheet := Image.create(32, 64, false, Image.FORMAT_RGBA8)
	sheet.fill(Color(0, 0, 0, 0))

	var left0 := _frame_image(LEFT_0, palette)
	var left1 := _frame_image(LEFT_1, palette)
	var right0 := left0.duplicate()
	var right1 := left1.duplicate()
	right0.flip_x()
	right1.flip_x()

	var frames := [
		[_frame_image(DOWN_0, palette), _frame_image(DOWN_1, palette)],
		[_frame_image(UP_0, palette), _frame_image(UP_1, palette)],
		[left0, left1],
		[right0, right1],
	]
	for row in 4:
		for col in 2:
			sheet.blit_rect(frames[row][col], Rect2i(0, 0, 16, 16), Vector2i(col * 16, row * 16))

	var err := sheet.save_png(path)
	print(path.get_file(), ": ", error_string(err))

func _gen_pirate_sheet() -> void:
	_gen_sheet(PALETTE, "res://assets/pirate_sheet.png")

	# NPCs get a palette swap — green headscarf, brown coat — so they read
	# as locals, not players, at a glance.
	var npc_palette := PALETTE.duplicate()
	npc_palette["R"] = Color(0.24, 0.45, 0.28)
	npc_palette["r"] = Color(0.16, 0.32, 0.19)
	npc_palette["W"] = Color(0.46, 0.34, 0.24)
	npc_palette["w"] = Color(0.36, 0.26, 0.18)
	npc_palette["N"] = Color(0.25, 0.23, 0.2)
	_gen_sheet(npc_palette, "res://assets/npc_sheet.png")

# --- dock -------------------------------------------------------------------

func _gen_dock() -> void:
	var w := 200
	var h := 56
	var img := Image.create(w, h, false, Image.FORMAT_RGBA8)
	img.fill(Color(0, 0, 0, 0))

	for y in range(8, 48):
		for x in w:
			var shade := (_rng.randf() - 0.5) * 0.04
			var c := Color(0.45 + shade, 0.33 + shade, 0.19 + shade)
			if x % 20 < 2:
				c = c.darkened(0.35) # plank gaps
			if y == 8 or y == 47:
				c = c.darkened(0.25) # edge bevel
			img.set_pixel(x, y, c)

	# Mooring posts.
	for post_x in [10, 186]:
		for y in range(2, 54):
			for x in range(post_x, post_x + 5):
				var c := Color(0.32, 0.22, 0.12)
				if y < 6:
					c = c.lightened(0.2)
				img.set_pixel(x, y, c)

	var err := img.save_png("res://assets/dock.png")
	print("dock.png: ", error_string(err))

# --- driftwood --------------------------------------------------------------

## A small pile of two crossed, weathered logs with darker end-grain. 24x16,
## drawn centered so the client can place it directly on a node's position.
func _gen_driftwood() -> void:
	var w := 24
	var h := 16
	var img := Image.create(w, h, false, Image.FORMAT_RGBA8)
	img.fill(Color(0, 0, 0, 0))

	var bark := Color(0.42, 0.33, 0.24)
	var bark_dark := Color(0.30, 0.23, 0.16)
	var grain := Color(0.55, 0.46, 0.34)

	# Two logs as thick line segments, slightly crossed.
	_log(img, Vector2(3, 11), Vector2(20, 8), 2.4, bark, bark_dark, grain)
	_log(img, Vector2(4, 6), Vector2(19, 12), 2.0, bark.lightened(0.08), bark_dark, grain)

	var err := img.save_png("res://assets/driftwood.png")
	print("driftwood.png: ", error_string(err))

## Draws a fat line from a to b: a rounded log of the given half-thickness,
## shaded along the bottom, with bright end-grain caps at both ends. Writes
## opaque pixels directly (unlike _blend, which refuses transparent targets).
func _log(img: Image, a: Vector2, b: Vector2, r: float, top: Color, bottom: Color, cap: Color) -> void:
	var steps := int(a.distance_to(b)) + 1
	for i in steps + 1:
		var p := a.lerp(b, float(i) / steps)
		var is_end := i <= 1 or i >= steps - 1
		for oy in range(-int(r) - 1, int(r) + 2):
			var d := absf(oy)
			if d > r:
				continue
			var x := int(round(p.x))
			var y := int(round(p.y)) + oy
			if x < 0 or x >= img.get_width() or y < 0 or y >= img.get_height():
				continue
			var c := top.lerp(bottom, d / r) # round shading top->bottom
			if is_end:
				c = cap
			img.set_pixel(x, y, Color(c.r, c.g, c.b, 1.0))
