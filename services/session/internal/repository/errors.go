package repository

import "errors"

var (
	// Session errors
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrSessionEnded         = errors.New("session has ended")
	ErrSessionFull          = errors.New("session is full")

	// Participant errors
	ErrParticipantNotFound      = errors.New("participant not found")
	ErrParticipantAlreadyJoined = errors.New("participant already joined")
	ErrParticipantNotInSession  = errors.New("participant not in session")
	ErrNotSessionOwner          = errors.New("user is not session owner")
	ErrInsufficientPermissions  = errors.New("insufficient permissions")

	// Database errors
	ErrDatabase = errors.New("database error")
)
