extends Control

## Login screen — the first stop in the player journey:
##
##   login screen --POST /v1/login--> API --> token + ticket --> Session
##   --> switch to the game scene, which connects to the game server
##
## This mirrors proposal §16: passwords/credentials only ever touch the API
## over HTTP(S); the game server never sees them. Registration and login use
## the same form and reply handling — they differ only in the endpoint and
## the messages, so they share one HTTPRequest.

const API_URL := "http://127.0.0.1:8080"
const GAME_SCENE := "res://scenes/game/game.tscn"
const MIN_PASSWORD := 8

@onready var _username: LineEdit = $Center/Form/UsernameField
@onready var _password: LineEdit = $Center/Form/PasswordField
@onready var _button: Button = $Center/Form/SetSailButton
@onready var _register: Button = $Center/Form/RegisterButton
@onready var _status: Label = $Center/Form/Status
@onready var _request: HTTPRequest = $LoginRequest

# Which endpoint the in-flight request targets, so _on_reply can phrase
# errors correctly ("name already taken" only makes sense for register).
var _pending_path := ""

func _ready() -> void:
	_button.pressed.connect(func() -> void: _submit("/v1/login"))
	_register.pressed.connect(func() -> void: _submit("/v1/register"))
	# Enter from either field signs in (the common case); register is a
	# deliberate button press.
	_username.text_submitted.connect(func(_t: String) -> void: _submit("/v1/login"))
	_password.text_submitted.connect(func(_t: String) -> void: _submit("/v1/login"))
	_request.request_completed.connect(_on_reply)
	_username.grab_focus()

	# Dev convenience: LFP_AUTOLOGIN=ben LFP_AUTOPASSWORD=... make play skips
	# the form. Also lets headless smoke tests drive the full journey. It
	# logs in; register a new name once by hand the first time.
	var auto := OS.get_environment("LFP_AUTOLOGIN")
	if auto != "":
		_username.text = auto
		_password.text = OS.get_environment("LFP_AUTOPASSWORD")
		_submit("/v1/login")

func _submit(path: String) -> void:
	var pirate_name := _username.text.strip_edges()
	if pirate_name.is_empty():
		_status.text = "Every pirate needs a name."
		return
	# Password is not trimmed: leading/trailing spaces are legitimate. The
	# server enforces the real rule; this is just a friendlier early no.
	if _password.text.length() < MIN_PASSWORD:
		_status.text = "Password must be at least %d characters." % MIN_PASSWORD
		return

	_pending_path = path
	_set_busy(true)
	_status.text = "Registering your pirate..." if path == "/v1/register" else "Signing the articles..."

	var body := JSON.stringify({"username": pirate_name, "password": _password.text})
	var err := _request.request(
		API_URL + path,
		["Content-Type: application/json"],
		HTTPClient.METHOD_POST,
		body,
	)
	if err != OK:
		_fail("Could not start request: %s" % error_string(err))

func _on_reply(
	result: int, code: int, _headers: PackedStringArray, body: PackedByteArray
) -> void:
	if result != HTTPRequest.RESULT_SUCCESS:
		_fail("No reply from the harbormaster. Is the backend up? (make up)")
		return

	if code != 200:
		_fail(_error_for(code))
		return

	var reply: Variant = JSON.parse_string(body.get_string_from_utf8())
	if typeof(reply) != TYPE_DICTIONARY:
		_fail("Garbled reply from the API")
		return

	Session.username = _username.text.strip_edges()
	Session.access_token = reply.get("access_token", "")
	Session.ticket = reply.get("ticket", "")
	print("[client] %s ok: username=%s ticket=%s" % [_pending_path, Session.username, Session.ticket])

	get_tree().change_scene_to_file(GAME_SCENE)

# _error_for turns a status code into a player-facing line. The server keeps
# 401 deliberately vague (no username enumeration); the client honors that.
func _error_for(code: int) -> String:
	match code:
		400:
			return "Name must be 2-24 characters and password 8+."
		401:
			return "Invalid pirate name or password."
		409:
			return "That name is already spoken for. Try another, or Set Sail."
		_:
			return "Request refused (HTTP %d)" % code

func _set_busy(busy: bool) -> void:
	_button.disabled = busy
	_register.disabled = busy

func _fail(message: String) -> void:
	_status.text = message
	_set_busy(false)
