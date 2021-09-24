package registry

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrExistingKey = Error("a key with this name already exists")
	ErrUnkownKey   = Error("an API key with this ID does not exist")
)
