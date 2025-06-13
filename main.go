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
	"fmt"
	"os/exec"

	gobirdc "github.com/StatCan/go-birdc"
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
		return false
	}
	// Parse the JSON output
	parsedOutput, err := parseJSON(output)
	if err != nil {
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

// main function to execute the command and parse the output
func main() {
	// Connect to the BIRD daemon using go-birdc
	b := gobirdc.New(&gobirdc.BirdClientOptions{
		Path: "/run/bird/bird.ctl"})

	// Check if the Tailscale PrimaryRouter is active
	isPrimaryRouter := checkTailscalePrimaryRouter()
	if isPrimaryRouter {
		// If "PrimaryRoutes" is found, enable the "tailscale" protocol
		_, _, err := b.EnableProtocol("tailscale")
		if err != nil {
			fmt.Println("Error enabling protocol:", err)
		}
	} else {
		// If "PrimaryRoutes" is not found, disable the "tailscale" protocol
		_, _, err := b.DisableProtocol("tailscale")
		if err != nil {
			fmt.Println("Error disabling protocol:", err)
		}
	}
}
