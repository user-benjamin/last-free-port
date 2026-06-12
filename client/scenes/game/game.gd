extends Control

## The in-game scene — second stop in the player journey. This is where the
## client-side *presentation* of the world lives: rendering, input, UI.
## The world's *truth* (positions, inventory, combat results) lives in
## server/ — this scene only sends intent and draws what the server says.
##
## Today it: connects to the game server, does the hello/welcome handshake
## using the logged-in name, and shows the result. The cove grows from here.

const GAME_SERVER_URL := "ws://127.0.0.1:8081/ws"
const LOGIN_SCENE := "res://scenes/login/login.tscn"

var _socket := WebSocketPeer.new()
var _hello_sent := false

@onready var _status: Label = $Center/Status
@onready var _leave: Button = $LeaveButton

func _ready() -> void:
	_leave.pressed.connect(_return_to_port)

	# Running game.tscn directly (F6 in the editor) skips login — bounce back.
	if not Session.is_logged_in():
		get_tree().call_deferred("change_scene_to_file", LOGIN_SCENE)
		return

	var err := _socket.connect_to_url(GAME_SERVER_URL)
	if err != OK:
		_status.text = "Failed to start connection: %s" % error_string(err)
		set_process(false)

func _process(_delta: float) -> void:
	_socket.poll()
	match _socket.get_ready_state():
		WebSocketPeer.STATE_OPEN:
			if not _hello_sent:
				_send({"type": "hello", "data": {"name": Session.username}})
				_hello_sent = true
			while _socket.get_available_packet_count() > 0:
				_handle_packet(_socket.get_packet())
		WebSocketPeer.STATE_CLOSED:
			_status.text = "Lost connection to the cove (%d): %s" % [
				_socket.get_close_code(), _socket.get_close_reason()
			]
			set_process(false)

func _send(msg: Dictionary) -> void:
	_socket.send_text(JSON.stringify(msg))

func _handle_packet(packet: PackedByteArray) -> void:
	var msg: Variant = JSON.parse_string(packet.get_string_from_utf8())
	if typeof(msg) != TYPE_DICTIONARY:
		push_warning("Unparseable message from server")
		return
	match msg.get("type"):
		"welcome":
			var data: Dictionary = msg.get("data", {})
			print("[client] welcome: player_id=%s motd=%s" % [
				data.get("player_id"), data.get("motd")
			])
			_status.text = "Welcome aboard, %s.\nPlayer %s\n\n%s" % [
				Session.username, data.get("player_id"), data.get("motd")
			]
		"error":
			_status.text = "Server error: %s" % JSON.stringify(msg.get("data"))
		_:
			# Unknown types are ignored so the client survives protocol growth.
			push_warning("Unknown message type: %s" % msg.get("type"))

func _return_to_port() -> void:
	_socket.close()
	Session.clear()
	get_tree().change_scene_to_file(LOGIN_SCENE)
