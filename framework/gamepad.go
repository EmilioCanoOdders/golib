package golib

import (
	"math"
	"slices"
	"strings"
	"sync/atomic"

	"golib/internal/device"
)

// GamepadButton is a button on a gamepad. The face buttons are named after an
// Xbox controller: A is the bottom one (Cross on PlayStation), B the right one
// (Circle), X the left one (Square) and Y the top one (Triangle).
type GamepadButton int32

// Gamepad buttons.
const (
	GamepadUp    GamepadButton = device.GamepadUp // the d-pad
	GamepadRight GamepadButton = device.GamepadRight
	GamepadDown  GamepadButton = device.GamepadDown
	GamepadLeft  GamepadButton = device.GamepadLeft

	GamepadY GamepadButton = device.GamepadY
	GamepadB GamepadButton = device.GamepadB
	GamepadA GamepadButton = device.GamepadA
	GamepadX GamepadButton = device.GamepadX

	GamepadLeftBumper   GamepadButton = device.GamepadLeftBumper
	GamepadLeftTrigger  GamepadButton = device.GamepadLeftTrigger // read as a button: pressed or not
	GamepadRightBumper  GamepadButton = device.GamepadRightBumper
	GamepadRightTrigger GamepadButton = device.GamepadRightTrigger

	GamepadBack  GamepadButton = device.GamepadBack  // View, Select or Share
	GamepadStart GamepadButton = device.GamepadStart // Menu, Start or Options

	GamepadLeftStickButton  GamepadButton = device.GamepadLeftStickButton // pressing the left stick in
	GamepadRightStickButton GamepadButton = device.GamepadRightStickButton
)

// GamepadType is the kind of a gamepad, which says what is printed on its
// buttons. GoLib reads the buttons by where they are, not by what they say:
// GamepadA is always the bottom face button, which reads A on an Xbox
// controller, Cross on a PlayStation one and B on a Nintendo one. A game that
// shows the player which button to press asks [Input.GamepadType] which
// labels or pictures to show.
type GamepadType int

// Gamepad types.
const (
	GamepadTypeXbox        GamepadType = iota // A, B, X, Y, LB, RB, LT, RT: GoLib's names for the buttons; also any gamepad GoLib doesn't recognize
	GamepadTypePlayStation                    // Cross, Circle, Square, Triangle, L1, R1, L2, R2
	GamepadTypeNintendo                       // B at the bottom, A on the right, Y on the left, X on top, L, R, ZL, ZR
)

// The USB vendor numbers of the makers whose gamepads have labels of their
// own, as a browser writes them in a gamepad's name.
const (
	sonyVendor     = "054c"
	nintendoVendor = "057e"
)

// nintendoNames are gamepads with Nintendo's labels whose names don't say
// so, as they call themselves, in lower case.
var nintendoNames = []string{
	"pro controller", // Nintendo's Switch Pro Controller, on macOS
	"horipad s",      // HORI's HORIPAD for Nintendo Switch
}

// maxGamepads is how many gamepads Input follows, numbered from 0.
const maxGamepads = 4

// gamepadButtonCount is one more than the highest GamepadButton, so buttons
// can index arrays.
const gamepadButtonCount = GamepadRightStickButton + 1

// stickDeadZone is how far a stick must move from its center before it counts,
// as a fraction of full tilt. Sticks rarely rest exactly at the center.
const stickDeadZone = 0.2

// gamepadButtonNames names every GamepadButton constant the way golib shot
// --input spells it.
var gamepadButtonNames = map[GamepadButton]string{
	GamepadUp: "GamepadUp", GamepadRight: "GamepadRight", GamepadDown: "GamepadDown", GamepadLeft: "GamepadLeft",
	GamepadY: "GamepadY", GamepadB: "GamepadB", GamepadA: "GamepadA", GamepadX: "GamepadX",
	GamepadLeftBumper: "GamepadLeftBumper", GamepadLeftTrigger: "GamepadLeftTrigger",
	GamepadRightBumper: "GamepadRightBumper", GamepadRightTrigger: "GamepadRightTrigger",
	GamepadBack: "GamepadBack", GamepadStart: "GamepadStart",
	GamepadLeftStickButton: "GamepadLeftStickButton", GamepadRightStickButton: "GamepadRightStickButton",
}

func validGamepadButton(button GamepadButton) bool {
	return button > 0 && button < gamepadButtonCount
}

// gamepadState is one gamepad as an update sees it, or, in inputQueue, as the
// latest frame left it, with presses not yet delivered.
type gamepadState struct {
	connected      bool
	name           string
	down           [gamepadButtonCount]bool
	pressed        [gamepadButtonCount]bool
	leftX, leftY   float32 // dead zone already removed
	rightX, rightY float32
}

// gamepadFrame is what the machine reports about one gamepad in one frame.
type gamepadFrame struct {
	connected      bool
	name           string
	down           [gamepadButtonCount]bool
	pressed        [gamepadButtonCount]bool
	leftX, leftY   float32 // raw, dead zone included
	rightX, rightY float32
}

// deviceGamepadFrame reads gamepad number pad from the machine.
func deviceGamepadFrame(pad int) gamepadFrame {
	var frame gamepadFrame
	if !device.GamepadConnected(pad) {
		return frame
	}
	frame.connected = true
	frame.name = device.GamepadName(pad)
	for button := range gamepadButtonNames {
		frame.down[button] = device.IsGamepadDown(pad, int32(button))
		frame.pressed[button] = device.IsGamepadPressed(pad, int32(button))
	}
	frame.leftX, frame.leftY, frame.rightX, frame.rightY = device.GamepadSticks(pad)
	return frame
}

// applyDeadZone removes the dead zone from a stick position: positions within
// stickDeadZone of the center become 0, 0, and the rest is rescaled so that
// the edge of the dead zone is 0 and full tilt is 1.
func applyDeadZone(x, y float32) (float32, float32) {
	length := float32(math.Hypot(float64(x), float64(y)))
	if length <= stickDeadZone {
		return 0, 0
	}
	scale := min((length-stickDeadZone)/(1-stickDeadZone), 1) / length
	return x * scale, y * scale
}

// gamepadTypeOf guesses the kind of a gamepad from its name: the name the
// system gives it on the desktop, such as "DualSense Wireless Controller" or
// "Pro Controller", or the one a browser gives it, which holds the maker's
// USB vendor number, as in "... (STANDARD GAMEPAD Vendor: 054c Product:
// 0ce6)" or "054c-0ce6-DualSense Wireless Controller".
func gamepadTypeOf(name string) GamepadType {
	name = strings.ToLower(strings.TrimSpace(name))
	product := productName(name)
	vendor := func(number string) bool {
		return strings.Contains(name, "vendor: "+number) || strings.HasPrefix(name, number+"-")
	}
	containsAny := func(words ...string) bool {
		for _, word := range words {
			if strings.Contains(name, word) {
				return true
			}
		}
		return false
	}
	switch {
	case vendor(sonyVendor) || containsAny("playstation", "dualsense", "dualshock", "sony", "ps3", "ps4", "ps5") ||
		product == "wireless controller": // a PlayStation 4 controller, on macOS and Windows
		return GamepadTypePlayStation
	case vendor(nintendoVendor) || containsAny("nintendo", "switch", "joy-con") || slices.Contains(nintendoNames, product):
		return GamepadTypeNintendo
	}
	return GamepadTypeXbox
}

// productName returns a gamepad's name, in lower case, without the vendor
// and product numbers a browser adds to it: "horipad s" for "horipad s
// (vendor: 0f0d product: 00c1)" and for "0f0d-00c1-horipad s".
func productName(name string) string {
	if i := strings.Index(name, " ("); i > 0 && strings.HasSuffix(name, ")") {
		if rest := name[i:]; strings.Contains(rest, "vendor: ") || strings.Contains(rest, "gamepad") {
			return name[:i]
		}
	}
	if parts := strings.SplitN(name, "-", 3); len(parts) == 3 && hexNumber(parts[0]) && hexNumber(parts[1]) {
		return parts[2]
	}
	return name
}

// hexNumber reports whether s is a number of one to four hexadecimal digits,
// as the vendor and product numbers in a browser's gamepad names are.
func hexNumber(s string) bool {
	if len(s) == 0 || len(s) > 4 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

// playingWithGamepad is whether the player is playing with a gamepad now. Run
// keeps it in step at the start of every frame, and golib shot sets it from
// its input script.
var playingWithGamepad atomic.Bool

// PlayingWithGamepad reports whether the player is playing with a gamepad,
// which is what decides whether a game shows gamepad buttons or keys in the
// prompts on its screen:
//
//	button := "Space"
//	if golib.PlayingWithGamepad() {
//		button = "A" // or the label input.GamepadType(0) says the bottom button has
//	}
//	hint := button + ": jump"
//
// It follows the player: false until a gamepad button is pressed or a stick is
// tilted, then true until a key, the mouse or a finger is used, and so on.
// Between the two it holds its answer, so the prompts don't change while the
// player does nothing. Under golib shot it is true when the --input script
// has gamepad items in it.
func PlayingWithGamepad() bool {
	return playingWithGamepad.Load()
}

// followGamepadPlaying moves PlayingWithGamepad to what the player last used.
// A frame in which they use nothing leaves it as it was.
func followGamepadPlaying(fingers, keyboard, mouse, gamepad bool) {
	switch {
	case gamepad:
		playingWithGamepad.Store(true)
	case keyboard || mouse || fingers:
		playingWithGamepad.Store(false)
	}
}
