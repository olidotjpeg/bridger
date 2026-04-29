//go:build dev

package main

// In dev mode Vite serves the frontend at :5173, not the Go server at :8080.
func browserURL() string { return "http://localhost:5173" }
