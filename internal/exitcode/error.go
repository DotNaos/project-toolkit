package exitcode

type Error struct {
	Code int
}

func (e *Error) Error() string {
	return "command exited with non-zero status"
}
