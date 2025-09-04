package server

import "errors"

var (
	ErrClientDisconnected = errors.New("client is disconnected")
	ErrServerNotRunning   = errors.New("server is not running")
	ErrServerFull         = errors.New("server is full")
	ErrInvalidCommand     = errors.New("invalid command")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrCharacterNotFound  = errors.New("character not found")
	ErrPlayerNotFound     = errors.New("player not found")
)