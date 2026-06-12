extends Control

## Login screen — the first stop in the player journey:
##
##   login screen --POST /v1/login--> API --> token + ticket --> Session
##   --> switch to the game scene, which connects to the game server
##
## This mirrors proposal §16: passwords/credentials only ever touch the API
## over HTTP(S); the game server never sees them.

const API_URL := "http://127.0.0.1:8080"
const GAME_SCENE := "res://scenes/game/game.tscn"

@onready var _username: LineEdit = $Center/Form/UsernameField
@onready var _button: Button = $Center/Form/SetSailButton
@onready var _status: Label = $Center/Form/Status
@onready var _request: HTTPRequest = $LoginRequest

func _ready() -> void:
	_button.pressed.connect(_attempt_login)
	_username.text_submitted.connect(func(_text: String) -> void: _attempt_login())
	_request.request_completed.connect(_on_login_reply)
	_username.grab_focus()

	# Dev convenience: LFP_AUTOLOGIN=ben make play skips the form. Also lets
	# headless smoke tests drive the full login -> game journey.
	var auto := OS.get_environment("LFP_AUTOLOGIN")
	if auto != "":
		_username.text = auto
		_attempt_login()

func _attempt_login() -> void:
	var pirate_name := _username.text.strip_edges()
	if pirate_name.is_empty():
		_status.text = "Every pirate needs a name."
		return

	_button.disabled = true
	_status.text = "Signing the articles..."

	var body := JSON.stringify({"username": pirate_name})
	var err := _request.request(
		API_URL + "/v1/login",
		["Content-Type: application/json"],
		HTTPClient.METHOD_POST,
		body,
	)
	if err != OK:
		_fail("Could not start login request: %s" % error_string(err))

func _on_login_reply(
	result: int, code: int, _headers: PackedStringArray, body: PackedByteArray
) -> void:
	if result != HTTPRequest.RESULT_SUCCESS:
		_fail("No reply from the harbormaster. Is the backend up? (make up)")
		return
	if code != 200:
		_fail("Login refused (HTTP %d)" % code)
		return

	var reply: Variant = JSON.parse_string(body.get_string_from_utf8())
	if typeof(reply) != TYPE_DICTIONARY:
		_fail("Garbled reply from the API")
		return

	Session.username = _username.text.strip_edges()
	Session.access_token = reply.get("access_token", "")
	Session.ticket = reply.get("ticket", "")
	print("[client] login ok: username=%s ticket=%s" % [Session.username, Session.ticket])

	get_tree().change_scene_to_file(GAME_SCENE)

func _fail(message: String) -> void:
	_status.text = message
	_button.disabled = false
