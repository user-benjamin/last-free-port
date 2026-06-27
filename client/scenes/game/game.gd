extends Node2D

## The in-game scene: the client-side *presentation* of the cove. The
## world's truth (positions, speed, bounds) lives in server/internal/game —
## this scene sends which way you're pushing and draws what the server
## says happened. If you're writing gameplay rules here, wrong folder.

const GAME_SERVER_URL := "ws://127.0.0.1:8081/ws"
const LOGIN_SCENE := "res://scenes/login/login.tscn"

## How aggressively rendered positions chase server positions. The server
## speaks at 20Hz; we render at 60+fps, so we glide between snapshots.
const LERP_RATE := 12.0

## Mirrors the server's TalkRange/GatherRange (both 48). Client-side this only
## decides when to show the prompt — the server re-checks distance on every
## talk_intent and gather_intent.
const INTERACT_RANGE := 48.0

const DRIFTWOOD_TEX := preload("res://assets/driftwood.png")

## Item presentation, the one place the client knows how to *show* an item
## type the server sends. Unknown types still appear (title-cased, iconless),
## so a new server-side item never silently vanishes from the hold.
const ITEM_ICONS := {"driftwood": DRIFTWOOD_TEX}
const ITEM_NAMES := {"driftwood": "Driftwood"}

## Resource-node tints, keyed by type. There's no scrap-iron art yet, so its
## nodes reuse the driftwood sprite with a rusty tint to read differently.
## Types absent here render untinted. Replace with real art down the line.
const NODE_TINTS := {"scrap_iron": Color(0.72, 0.55, 0.42)}

var _socket := WebSocketPeer.new()
var _join_sent := false
var _my_id := ""
var _last_sent_dir := Vector2.ZERO
var _nodes: Dictionary = {}       # player_id -> Node2D
var _targets: Dictionary = {}     # player_id -> Vector2 (latest server position)
var _npc_nodes: Dictionary = {}   # npc_id -> Node2D
var _npc_names: Dictionary = {}   # npc_id -> String
var _res_nodes: Dictionary = {}   # node_id -> {sprite, pos, type, available}
var _inventory: Dictionary = {}   # item_type -> quantity
var _recipes: Array = []          # recipe dicts from welcome (server-defined)
var _nearest := {}                # {} or {kind: "npc"|"node", id, label}

@onready var _players: Node2D = $Players
@onready var _resources_root: Node2D = $Resources
@onready var _camera: Camera2D = $Camera
@onready var _status: Label = $UI/Status
@onready var _leave: Button = $UI/LeaveButton
@onready var _waves: AudioStreamPlayer = $Waves
@onready var _prompt: Label = $UI/Prompt
@onready var _inventory_label: Label = $UI/Inventory
@onready var _inventory_panel: PanelContainer = $UI/InventoryPanel
@onready var _inventory_list: VBoxContainer = $UI/InventoryPanel/Margin/VBox/List
@onready var _inventory_empty: Label = $UI/InventoryPanel/Margin/VBox/Empty
@onready var _recipes_list: VBoxContainer = $UI/InventoryPanel/Margin/VBox/Recipes
@onready var _dialogue_panel: PanelContainer = $UI/DialoguePanel
@onready var _dialogue_name: Label = $UI/DialoguePanel/Margin/VBox/NpcName
@onready var _dialogue_line: Label = $UI/DialoguePanel/Margin/VBox/Line

var _pirate_frames: SpriteFrames
var _npc_frames: SpriteFrames

func _ready() -> void:
	_leave.pressed.connect(_return_to_port)

	# Running game.tscn directly (F6 in the editor) skips login — bounce back.
	if not Session.is_logged_in():
		get_tree().call_deferred("change_scene_to_file", LOGIN_SCENE)
		return

	_pirate_frames = _build_frames("res://assets/pirate_sheet.png")
	_npc_frames = _build_frames("res://assets/npc_sheet.png")
	_start_ambience()
	_camera.make_current()
	var err := _socket.connect_to_url(GAME_SERVER_URL)
	if err != OK:
		_status.text = "Failed to start connection: %s" % error_string(err)
		set_process(false)

## Slice a generated sprite sheet (rows: down/up/left/right, two walk
## frames each) into named animations. One SpriteFrames per sheet is shared
## by every node using it.
func _build_frames(sheet_path: String) -> SpriteFrames:
	var sheet: Texture2D = load(sheet_path)
	var frames := SpriteFrames.new()
	var dirs := ["down", "up", "left", "right"]
	for row in dirs.size():
		var anim: String = "walk_" + dirs[row]
		frames.add_animation(anim)
		frames.set_animation_speed(anim, 6.0)
		for col in 2:
			var tile := AtlasTexture.new()
			tile.atlas = sheet
			tile.region = Rect2(col * 16, row * 16, 16, 16)
			frames.add_frame(anim, tile)
	return frames

func _start_ambience() -> void:
	var stream: AudioStreamWAV = load("res://assets/audio/waves.wav")
	if stream == null:
		push_warning("waves.wav missing — run: make art")
		return
	stream.loop_mode = AudioStreamWAV.LOOP_FORWARD
	stream.loop_end = stream.data.size() / 4 # 16-bit stereo: 4 bytes/frame
	_waves.stream = stream
	_waves.play()

func _process(delta: float) -> void:
	_socket.poll()
	match _socket.get_ready_state():
		WebSocketPeer.STATE_OPEN:
			if not _join_sent:
				# Identity comes from the ticket, not from anything we claim.
				_send({"type": "join", "data": {"ticket": Session.ticket}})
				_join_sent = true
			while _socket.get_available_packet_count() > 0:
				_handle_packet(_socket.get_packet())
			_send_movement()
			if Input.is_action_just_pressed("interact"):
				_on_interact()
			if Input.is_action_just_pressed("inventory"):
				_toggle_inventory()
		WebSocketPeer.STATE_CLOSED:
			_status.text = "Lost connection to the cove (%d): %s" % [
				_socket.get_close_code(), _socket.get_close_reason()
			]
			set_process(false)

	_animate_players(delta)
	_update_prompt()

## Send the held direction only when it changes (including release -> zero).
## The server holds the last intent and applies it every tick.
func _send_movement() -> void:
	var dir := Input.get_vector("move_left", "move_right", "move_up", "move_down")
	if dir != _last_sent_dir:
		_send({"type": "move_intent", "data": {"dx": dir.x, "dy": dir.y}})
		_last_sent_dir = dir

func _animate_players(delta: float) -> void:
	var blend := 1.0 - exp(-LERP_RATE * delta)
	for id: String in _nodes:
		var node: Node2D = _nodes[id]
		var target: Vector2 = _targets.get(id, node.position)
		var motion := target - node.position
		node.position = node.position.lerp(target, blend)
		_face_sprite(node.get_node("Sprite") as AnimatedSprite2D, motion)
	if _nodes.has(_my_id):
		_camera.position = _nodes[_my_id].position

## Choose the walk animation from how the sprite is actually moving; stand
## still on frame 0 when the server stops reporting movement.
func _face_sprite(sprite: AnimatedSprite2D, motion: Vector2) -> void:
	if motion.length() < 1.5:
		sprite.stop()
		sprite.frame = 0
		return
	var anim := "walk_right" if motion.x > 0 else "walk_left"
	if absf(motion.y) > absf(motion.x):
		anim = "walk_down" if motion.y > 0 else "walk_up"
	if sprite.animation != anim or not sprite.is_playing():
		sprite.play(anim)

func _send(msg: Dictionary) -> void:
	_socket.send_text(JSON.stringify(msg))

func _handle_packet(packet: PackedByteArray) -> void:
	var msg: Variant = JSON.parse_string(packet.get_string_from_utf8())
	if typeof(msg) != TYPE_DICTIONARY:
		push_warning("Unparseable message from server")
		return
	match msg.get("type"):
		"welcome":
			_on_welcome(msg.get("data", {}))
		"state":
			_on_state(msg.get("data", {}))
		"dialogue":
			_on_dialogue(msg.get("data", {}))
		"inventory":
			_on_inventory(msg.get("data", {}))
		"error":
			_status.text = "Server error: %s" % JSON.stringify(msg.get("data"))
		_:
			# Unknown types are ignored so the client survives protocol growth.
			push_warning("Unknown message type: %s" % msg.get("type"))

func _on_welcome(data: Dictionary) -> void:
	_my_id = data.get("player_id", "")
	print("[client] welcome: player_id=%s motd=%s" % [_my_id, data.get("motd")])
	var spawn := Vector2(data.get("spawn_x", 0.0), data.get("spawn_y", 0.0))
	_camera.position = spawn
	_status.text = "%s\nWASD to walk, E to interact, I for your hold." % data.get("motd")

	# Recipes are server-defined content; the client only renders them.
	_recipes = data.get("recipes", [])

	# Returning players get their stuff back here — this is the payoff.
	_inventory.clear()
	for item: String in data.get("inventory", {}):
		_inventory[item] = int(data["inventory"][item])
	_refresh_inventory()

func _on_inventory(data: Dictionary) -> void:
	var item: String = data.get("item_type", "")
	if item != "":
		_inventory[item] = int(data.get("quantity", 0))
		_refresh_inventory()
		print("[client] inventory: %s=%d" % [item, _inventory[item]])

## Keep both views of the hold in sync from the one authoritative dict: the
## always-on corner readout and the toggle panel. Called whenever the server
## changes a quantity, so an open panel updates live as you gather.
func _refresh_inventory() -> void:
	_inventory_label.text = "Driftwood: %d" % int(_inventory.get("driftwood", 0))
	_rebuild_inventory_panel()
	# Recipe rows show their cost and disable when unaffordable, so they must
	# rebuild whenever the inventory changes, not just on join.
	_rebuild_recipes()

func _toggle_inventory() -> void:
	_inventory_panel.visible = not _inventory_panel.visible

## Rebuild the panel's rows from scratch — at hold-sized item counts this is
## cheaper than diffing, and it can't drift out of sync. Items are listed
## alphabetically so a row never jumps around between rebuilds; zero-quantity
## items are hidden, and an empty hold shows the placeholder line instead.
func _rebuild_inventory_panel() -> void:
	for row in _inventory_list.get_children():
		row.queue_free()

	var items := _inventory.keys()
	items.sort()
	var shown := 0
	for item: String in items:
		var qty := int(_inventory[item])
		if qty <= 0:
			continue
		_inventory_list.add_child(_make_inventory_row(item, qty))
		shown += 1
	_inventory_empty.visible = shown == 0

## One row: icon (if the client has art for the type), name, right-aligned
## count. Built in code because the row count is data-driven.
func _make_inventory_row(item: String, qty: int) -> HBoxContainer:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)

	var icon := TextureRect.new()
	icon.custom_minimum_size = Vector2(20, 20)
	icon.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	icon.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	icon.texture = ITEM_ICONS.get(item, null)
	row.add_child(icon)

	var name_label := Label.new()
	name_label.text = _display_name(item)
	name_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.add_child(name_label)

	var qty_label := Label.new()
	qty_label.text = "x%d" % qty
	qty_label.add_theme_color_override("font_color", Color(0.95, 0.88, 0.6, 1))
	row.add_child(qty_label)

	return row

## Friendly label for an item type: a known name, else the type title-cased
## (Godot's capitalize turns "scrap_iron" into "Scrap Iron"). The one place
## item display names are resolved, shared by the hold and the crafting list.
func _display_name(item: String) -> String:
	return ITEM_NAMES.get(item, item.capitalize())

## Rebuild the crafting list from the server's recipes. Each row is a Craft
## button plus its cost; the button disables when the player can't afford it,
## but the server re-checks regardless — this is only a courtesy, not a gate.
func _rebuild_recipes() -> void:
	for row in _recipes_list.get_children():
		row.queue_free()
	for recipe: Dictionary in _recipes:
		_recipes_list.add_child(_make_recipe_row(recipe))

func _make_recipe_row(recipe: Dictionary) -> HBoxContainer:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)

	var inputs: Dictionary = recipe.get("inputs", {})
	var affordable := _can_afford(inputs)

	var button := Button.new()
	button.text = "Craft %s" % _display_name(recipe.get("output", "?"))
	button.disabled = not affordable
	var recipe_id: String = recipe.get("id", "")
	button.pressed.connect(func() -> void: _craft(recipe_id))
	row.add_child(button)

	var cost := Label.new()
	cost.text = _format_inputs(inputs)
	cost.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	cost.add_theme_font_size_override("font_size", 11)
	cost.add_theme_color_override(
		"font_color", Color(1, 1, 1, 0.55) if affordable else Color(1, 0.6, 0.5, 0.7)
	)
	row.add_child(cost)

	return row

## True only if the player holds every input in the required quantity.
func _can_afford(inputs: Dictionary) -> bool:
	for item: String in inputs:
		if int(_inventory.get(item, 0)) < int(inputs[item]):
			return false
	return true

## "1 driftwood, 1 scrap iron" — the recipe's inputs in a stable order.
func _format_inputs(inputs: Dictionary) -> String:
	var items := inputs.keys()
	items.sort()
	var parts: Array[String] = []
	for item: String in items:
		parts.append("%d %s" % [int(inputs[item]), _display_name(item)])
	return ", ".join(parts)

func _craft(recipe_id: String) -> void:
	_send({"type": "craft_intent", "data": {"recipe_id": recipe_id}})

## The server snapshot is the truth: create nodes for new players, update
## targets for known ones, remove anyone no longer present.
func _on_state(data: Dictionary) -> void:
	var seen := {}
	for p: Dictionary in data.get("players", []):
		var id: String = p.get("id", "")
		seen[id] = true
		if not _nodes.has(id):
			_spawn_node(id, p.get("name", "?"), Vector2(p.get("x", 0.0), p.get("y", 0.0)))
			print("[client] sighted: %s (%s players in cove)" % [p.get("name"), data.get("players", []).size()])
		_targets[id] = Vector2(p.get("x", 0.0), p.get("y", 0.0))

	for npc: Dictionary in data.get("npcs", []):
		var nid: String = npc.get("id", "")
		if not _npc_nodes.has(nid):
			var npc_name: String = npc.get("name", "?")
			var at := Vector2(npc.get("x", 0.0), npc.get("y", 0.0))
			_npc_nodes[nid] = _make_actor(npc_name, at, _npc_frames, Color(0.93, 0.82, 0.5))
			_npc_names[nid] = npc_name
			print("[client] local sighted: %s" % npc_name)

	for res: Dictionary in data.get("resources", []):
		_sync_resource(res)

	for id: String in _nodes.keys():
		if not seen.has(id):
			_nodes[id].queue_free()
			_nodes.erase(id)
			_targets.erase(id)

## Resource nodes are fixed spawn points (never removed), so we create the
## sprite once and then just toggle visibility as the server reports the node
## depleting and respawning.
func _sync_resource(res: Dictionary) -> void:
	var id: String = res.get("id", "")
	if id == "":
		return
	var available: bool = res.get("available", true)
	if not _res_nodes.has(id):
		var pos := Vector2(res.get("x", 0.0), res.get("y", 0.0))
		var node_type: String = res.get("type", "driftwood")
		var sprite := Sprite2D.new()
		sprite.texture = DRIFTWOOD_TEX
		# No per-type art yet; a tint distinguishes scrap iron from driftwood.
		sprite.modulate = NODE_TINTS.get(node_type, Color.WHITE)
		sprite.position = pos
		_resources_root.add_child(sprite)
		_res_nodes[id] = {
			"sprite": sprite, "pos": pos, "type": res.get("type", "driftwood"),
			"available": available,
		}
	var entry: Dictionary = _res_nodes[id]
	entry["available"] = available
	(entry["sprite"] as Sprite2D).visible = available

func _spawn_node(id: String, pirate_name: String, at: Vector2) -> void:
	var node := _make_actor(pirate_name, at, _pirate_frames, Color.WHITE)
	if id != _my_id:
		# Subtle cool tint to spot strangers.
		(node.get_node("Sprite") as AnimatedSprite2D).modulate = Color(0.85, 0.95, 1.0)
	_nodes[id] = node
	_targets[id] = at

func _make_actor(display_name: String, at: Vector2, frames: SpriteFrames, label_color: Color) -> Node2D:
	var node := Node2D.new()
	node.position = at

	var sprite := AnimatedSprite2D.new()
	sprite.name = "Sprite"
	sprite.sprite_frames = frames
	sprite.animation = "walk_down"
	sprite.position = Vector2(0, -8) # feet at the node's world position
	node.add_child(sprite)

	var label := Label.new()
	label.text = display_name
	label.position = Vector2(-60, -32)
	label.custom_minimum_size = Vector2(120, 0)
	label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	label.add_theme_font_size_override("font_size", 8)
	label.add_theme_color_override("font_color", label_color)
	label.add_theme_color_override("font_outline_color", Color(0, 0, 0, 0.8))
	label.add_theme_constant_override("outline_size", 3)
	node.add_child(label)

	_players.add_child(node)
	return node

## Interact (E): close an open conversation, otherwise act on whatever's
## nearest — talk to an NPC or gather a node. The server validates either way.
func _on_interact() -> void:
	if _dialogue_panel.visible:
		_dialogue_panel.visible = false
		return
	if _nearest.is_empty():
		return
	match _nearest.kind:
		"npc":
			_send({"type": "talk_intent", "data": {"npc_id": _nearest.id}})
		"node":
			_send({"type": "gather_intent", "data": {"node_id": _nearest.id}})

## Each frame, find the single nearest interactable (NPC or available node)
## within range and describe it for the prompt.
func _update_prompt() -> void:
	_nearest = {}
	if _nodes.has(_my_id):
		var me: Vector2 = _nodes[_my_id].position
		var best := INTERACT_RANGE
		for id: String in _npc_nodes:
			var d := me.distance_to(_npc_nodes[id].position)
			if d <= best:
				best = d
				_nearest = {"kind": "npc", "id": id, "label": "talk to %s" % _npc_names[id]}
		for id: String in _res_nodes:
			var entry: Dictionary = _res_nodes[id]
			if not entry.available:
				continue
			var d := me.distance_to(entry.pos)
			if d <= best:
				best = d
				_nearest = {"kind": "node", "id": id, "label": "gather %s" % _display_name(entry.type).to_lower()}

	_prompt.visible = not _nearest.is_empty() and not _dialogue_panel.visible
	if _prompt.visible:
		_prompt.text = "E — %s" % _nearest.label
	# Walking away from an NPC ends an open conversation.
	if _dialogue_panel.visible and (_nearest.is_empty() or _nearest.kind != "npc"):
		_dialogue_panel.visible = false

func _on_dialogue(data: Dictionary) -> void:
	_dialogue_name.text = data.get("npc_name", "?")
	_dialogue_line.text = data.get("line", "...")
	_dialogue_panel.visible = true
	print("[client] %s says: %s" % [data.get("npc_name"), data.get("line")])

func _return_to_port() -> void:
	_socket.close()
	Session.clear()
	get_tree().change_scene_to_file(LOGIN_SCENE)
