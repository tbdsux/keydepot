package auth

import (
	"errors"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const (
	// LAPolicyDeviceOwnerAuthentication: Touch ID, falling back to the login
	// password after failed attempts or while the sensor can't be reached.
	policyDeviceOwner = 2
	policyBiometrics  = 1
	biometryTouchID   = 1

	laErrorUserCancel          = -2
	laErrorBiometryNotEnrolled = -7
)

func newAuthContext() (objc.ID, error) {
	if _, err := purego.Dlopen("/System/Library/Frameworks/LocalAuthentication.framework/LocalAuthentication", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		return 0, err
	}

	ctx := objc.ID(objc.GetClass("LAContext")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
	if ctx == 0 {
		return 0, errors.New("Touch ID is not available.")
	}

	return ctx, nil
}

// touchIDReady reports a sensor with fingerprints enrolled. The biometry type
// also covers one locked out after failed attempts, or under a closed lid:
// those must still prompt, not open keys freely.
func touchIDReady(ctx objc.ID) bool {
	var nserr objc.ID
	if objc.Send[bool](ctx, objc.RegisterName("canEvaluatePolicy:error:"), int64(policyBiometrics), &nserr) {
		return true
	}

	if nserr != 0 && objc.Send[int64](nserr, objc.RegisterName("code")) == laErrorBiometryNotEnrolled {
		return false
	}

	return objc.Send[int64](ctx, objc.RegisterName("biometryType")) == biometryTouchID
}

// HasTouchID reports whether the system prompt includes Touch ID.
func HasTouchID() bool {
	ctx, err := newAuthContext()
	if err != nil {
		return false
	}

	defer ctx.Send(objc.RegisterName("release"))

	return touchIDReady(ctx)
}

// HasSystemAuth reports whether Authenticate can prompt at all: a Mac with no
// login password can't.
func HasSystemAuth() bool {
	ctx, err := newAuthContext()
	if err != nil {
		return false
	}

	defer ctx.Send(objc.RegisterName("release"))

	var nserr objc.ID

	return objc.Send[bool](ctx, objc.RegisterName("canEvaluatePolicy:error:"), int64(policyDeviceOwner), &nserr)
}

// Authenticate asks for Touch ID, or the login password on a Mac without it.
// It blocks until the user passes or declines the system prompt, so call it off
// the main thread. Anything but a passed prompt is an error.
func Authenticate(reason string) error {
	ctx, err := newAuthContext()
	if err != nil {
		return err
	}

	defer ctx.Send(objc.RegisterName("release"))

	return evaluate(ctx, reason)
}

func evaluate(ctx objc.ID, reason string) error {
	// alloc and release: this goroutine's thread has no autorelease pool.
	ns := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("initWithUTF8String:"), reason+"\x00")
	defer ns.Send(objc.RegisterName("release"))

	result := make(chan error, 1)

	reply := objc.NewBlock(func(_ objc.Block, passed bool, nserr objc.ID) {
		switch {
		case passed:
			result <- nil

		case nserr != 0 && objc.Send[int64](nserr, objc.RegisterName("code")) == laErrorUserCancel:
			result <- ErrCanceled

		case nserr != 0:
			desc := nserr.Send(objc.RegisterName("localizedDescription"))
			result <- errors.New(objc.Send[string](desc, objc.RegisterName("UTF8String")))

		default:
			result <- errors.New("Touch ID failed.")
		}
	})

	defer reply.Release()

	ctx.Send(objc.RegisterName("evaluatePolicy:localizedReason:reply:"), int64(policyDeviceOwner), ns, reply)

	return <-result
}
