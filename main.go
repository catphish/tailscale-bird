package main

// This program checks the status of the Tailscale service and enables or disables the "tailscale" protocol
// in BIRD based on the presence of "PrimaryRoutes" in the Tailscale status output.
// It uses the go-birdc library to interact with the BIRD daemon over a Unix domain socket.
// It executes the command "tailscale status --json --self" to get the Tailscale status in JSON format,
// parses the output, and checks if the "PrimaryRoutes" key exists in the "Self" section of the JSON.
// If "PrimaryRoutes" is found, it enables the "tailscale" protocol in BIRD; otherwise, it disables it.
// The program assumes that the BIRD daemon is running and accessible at the specified path "/run/bird/bird.ctl".
// It also assumes that the Tailscale command-line tool is installed and accessible in the system's PATH.
// It handles errors gracefully, printing error messages if command execution or JSON parsing fails.

import (
	"encoding/json"
	"os"
	"os/exec"
	"time"

	gobirdc "github.com/StatCan/go-birdc"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// executeCommand executes a shell command and returns its output as a string.
func executeCommand(cmd string) (string, error) {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// parseJSON parses a JSON string and returns a map representation of it.
func parseJSON(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Check the status of the tailscale Primaryrouter
func checkTailscalePrimaryRouter() bool {
	// Execute the shell command "tailscale status --json --self"
	cmd := "tailscale status --json --self"
	output, err := executeCommand(cmd)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute command")
		return false
	}
	// Parse the JSON output
	parsedOutput, err := parseJSON(output)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse JSON output")
		return false
	}

	// Check if the parsed output contains the key "PrimaryRoutes" within "Self"
	if self, ok := parsedOutput["Self"].(map[string]interface{}); ok {
		if _, exists := self["PrimaryRoutes"]; exists {
			// If "PrimaryRoutes" is found, return true
			return true
		}
	}
	// If "PrimaryRoutes" is not found, return false
	return false
}

// The program runs indefinitely in a loop, checking the Tailscale status and
// updating the BIRD protocol state.
func main() {
	// Set up a logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	log.Info().Msg("Starting Tailscale PrimaryRouter status checker")
	// Keep track of the Tailscale PrimaryRouter status in a variable. There
	// are three states: "unknown", "enabled", and "disabled".
	var tailscalePrimaryRouterStatus string = "unknown"

	// Check the status in a loop
	for {
		// Connect to the BIRD daemon using go-birdc
		b := gobirdc.New(&gobirdc.BirdClientOptions{
			Path: "/run/bird/bird.ctl"})

		// Check if the Tailscale PrimaryRouter is active
		isPrimaryRouter := checkTailscalePrimaryRouter()
		if isPrimaryRouter {
			// If "PrimaryRoutes" is found and the status is "unknown" or "disabled" then enable the "tailscale" protocol
			if tailscalePrimaryRouterStatus == "unknown" || tailscalePrimaryRouterStatus == "disabled" {
				log.Info().Msg("Tailscale PrimaryRouter is active, enabling protocol")
				_, _, err := b.EnableProtocol("tailscale")
				if err != nil {
					log.Error().Err(err).Msg("Failed to enable tailscale protocol")
				} else {
					log.Info().Msg("Tailscale PrimaryRouter protocol enabled successfully")
					tailscalePrimaryRouterStatus = "enabled"
				}
			}
		} else {
			// If "PrimaryRoutes" is not found, disable the "tailscale" protocol
			if tailscalePrimaryRouterStatus == "enabled" || tailscalePrimaryRouterStatus == "unknown" {
				log.Info().Msg("Tailscale PrimaryRouter is inactive, disabling protocol")
				_, _, err := b.DisableProtocol("tailscale")
				if err != nil {
					log.Error().Err(err).Msg("Failed to disable tailscale protocol")
				} else {
					log.Info().Msg("Tailscale PrimaryRouter protocol disabled successfully")
					tailscalePrimaryRouterStatus = "disabled"
				}
			}
		}
		// Sleep for 15 seconds before checking again
		time.Sleep(15 * time.Second)
	}
}
