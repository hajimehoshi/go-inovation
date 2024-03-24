//go:build !android && !ios

package input

func isTouchPrimaryInput() bool {
	return false
}
