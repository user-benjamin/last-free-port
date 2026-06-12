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

var _socket := WebSocketPeer.new()
var _hello_sent := false
var _my_id := ""
var _last_sent_dir := Vector2.ZERO
var _nodes: Dictionary = {}    # player_id -> Node2D
var _targets: Dictionary = {}  # player_id -> Vector2 (latest server position)

@onready var _players: Node2D = $Players
@onready var _camera: Camera2D = $Camera
@onready var _status: Label = $UI/Status
@onready var _leave: Button = $UI/LeaveButton

func _ready() -> void:
	_leave.pressed.connect(_return_to_port)

	# Running game.tscn directly (F6 in the editor) skips login — bounce back.
	if not Session.is_logged_in():
		get_tree().call_deferred("change_scene_to_file", LOGIN_SCENE)
		return

	_camera.make_current()
	var err := _socket.connect_to_url(GAME_SERVER_URL)
	if err != OK:
		_status.text = "Failed to start connection: %s" % error_string(err)
		set_process(false)

func _process(delta: float) -> void:
	_socket.poll()
	match _socket.get_ready_state():
		WebSocketPeer.STATE_OPEN:
			if not _hello_sent:
				_send({"type": "hello", "data": {"name": Session.username}})
				_hello_sent = true
			while _socket.get_available_packet_count() > 0:
				_handle_packet(_socket.get_packet())
			_send_movement()
		WebSocketPeer.STATE_CLOSED:
			_status.text = "Lost connection to the cove (%d): %s" % [
				_socket.get_close_code(), _socket.get_close_reason()
			]
			set_process(false)

	_animate_players(delta)

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
		node.position = node.position.lerp(_targets.get(id, node.position), blend)
	if _nodes.has(_my_id):
		_camera.position = _nodes[_my_id].position

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
	_status.text = "%s\nWASD or arrows to walk the cove." % data.get("motd")

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

	for id: String in _nodes.keys():
		if not seen.has(id):
			_nodes[id].queue_free()
			_nodes.erase(id)
			_targets.erase(id)

func _spawn_node(id: String, pirate_name: String, at: Vector2) -> void:
	var node := Node2D.new()
	node.position = at

	var body := ColorRect.new()
	body.size = Vector2(24, 24)
	body.position = Vector2(-12, -12)
	body.color = Color(0.95, 0.78, 0.25) if id == _my_id else Color(0.78, 0.28, 0.22)
	node.add_child(body)

	var label := Label.new()
	label.text = pirate_name
	label.position = Vector2(-60, -44)
	label.custom_minimum_size = Vector2(120, 0)
	label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	node.add_child(label)

	_players.add_child(node)
	_nodes[id] = node
	_targets[id] = at

func _return_to_port() -> void:
	_socket.close()
	Session.clear()
	get_tree().change_scene_to_file(LOGIN_SCENE)
