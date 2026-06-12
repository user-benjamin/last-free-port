extends SceneTree

## Generates the placeholder title-screen art (night sea, moon, stars) so the
## login screen isn't blank before real art exists. Deterministic (seeded).
##
## Run from the repo root:
##   godot --headless --path client -s res://tools/gen_title_art.gd

func _init() -> void:
	var w := 960
	var h := 540
	var horizon := int(h * 0.62)
	var img := Image.create(w, h, false, Image.FORMAT_RGB8)
	var rng := RandomNumberGenerator.new()
	rng.seed = 1715  # the golden age, roughly

	# Night sky: dark zenith fading toward the horizon.
	for y in horizon:
		var t := float(y) / horizon
		var c := Color(0.02, 0.03, 0.07).lerp(Color(0.10, 0.13, 0.20), t)
		for x in w:
			img.set_pixel(x, y, c)

	# Stars.
	for i in 140:
		var sx := rng.randi_range(0, w - 1)
		var sy := rng.randi_range(0, horizon - 40)
		var b := rng.randf_range(0.4, 0.95)
		img.set_pixel(sx, sy, Color(b, b, b * 0.9))

	# Moon.
	var mx := int(w * 0.72)
	var my := int(h * 0.26)
	var mr := 34.0
	for y in range(my - 40, my + 40):
		for x in range(mx - 40, mx + 40):
			var d := Vector2(x - mx, y - my).length()
			if d < mr:
				var glow := clampf(1.0 - (d / mr) * 0.25, 0.0, 1.0)
				img.set_pixel(x, y, Color(0.91 * glow, 0.89 * glow, 0.80 * glow))

	# Sea: darker than the sky, fading to near-black at the bottom.
	for y in range(horizon, h):
		var t := float(y - horizon) / float(h - horizon)
		var c := Color(0.05, 0.10, 0.14).lerp(Color(0.01, 0.03, 0.05), t)
		for x in w:
			img.set_pixel(x, y, c)

	# Moonlight glitter on the water, widening with distance.
	for y in range(horizon, h, 2):
		var half := 8 + int((y - horizon) * 0.10)
		for x in range(mx - half, mx + half):
			if x >= 0 and x < w and rng.randf() < 0.18:
				img.set_pixel(x, y, img.get_pixel(x, y) + Color(0.10, 0.10, 0.08))

	var err := img.save_png("res://assets/title_art.png")
	print("save_png result: ", error_string(err))
	quit()
