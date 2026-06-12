extends Node

## Autoloaded singleton (see [autoload] in project.godot). Godot creates one
## instance of this at startup and keeps it alive across scene changes —
## scenes are destroyed when you switch screens, so anything that must
## survive the login -> game transition lives here.

var username := ""
var access_token := ""
var ticket := ""

func is_logged_in() -> bool:
	return ticket != ""

func clear() -> void:
	username = ""
	access_token = ""
	ticket = ""
