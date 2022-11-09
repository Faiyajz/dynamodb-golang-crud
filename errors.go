package main

import "errors"

var (
	ErrInternal = errors.New("internal")
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)
