package rtc

type middleware interface {
	ID() string
	Stop()
}
