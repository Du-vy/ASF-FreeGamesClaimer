package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

func logInfo(format string, v ...interface{}) {
	log.Printf(colorCyan+"[INFO] "+colorReset+format, v...)
}

func logSuccess(format string, v ...interface{}) {
	log.Printf(colorGreen+"[SUCCESS] "+colorReset+format, v...)
}

func logWarning(format string, v ...interface{}) {
	log.Printf(colorYellow+"[WARN] "+colorReset+format, v...)
}

func logError(format string, v ...interface{}) {
	log.Printf(colorRed+"[ERROR] "+colorReset+format, v...)
}

type GistFile struct {
	Content string `json:"content"`
}

type GistResponse struct {
	Files map[string]GistFile `json:"files"`
}

type ASFRequest struct {
	Command string `json:"Command"`
}

type ASFResponse struct {
	Success bool            `json:"Success"`
	Message string          `json:"Message"`
	Result  json.RawMessage `json:"Result"`
}

func main() {
	logInfo("Starting ASF Free Games Claimer (Go version)...")
	loadEnv(".env")

	// Run initially on startup
	checkGame()

	// Run every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		checkGame()
	}
}

func loadEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		// Environment file is optional, environment variables might be set via system/Docker.
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Trim comments from the value
			if idx := strings.Index(value, "#"); idx != -1 {
				value = strings.TrimSpace(value[:idx])
			}
			// Trim optional quotes
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			// Only set if not already present in OS environment
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}

func readLastLength() int {
	data, err := os.ReadFile("lastlength")
	if err != nil {
		if os.IsNotExist(err) {
			err = os.WriteFile("lastlength", []byte("0"), 0644)
			if err != nil {
				logError("Error creating lastlength file: %v", err)
			}
			return 0
		}
		logError("Error reading lastlength: %v", err)
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		logError("Error parsing lastlength content '%s': %v", string(data), err)
		return 0
	}
	return val
}

func writeLastLength(val int) {
	err := os.WriteFile("lastlength", []byte(strconv.Itoa(val)), 0644)
	if err != nil {
		logError("Error writing lastlength: %v", err)
	}
}

func checkGame() {
	logInfo("Checking for free game licenses...")

	// 1. Fetch gist from GitHub
	req, err := http.NewRequest("GET", "https://api.github.com/gists/e8c5cf365d816f2640242bf01d8d3675", nil)
	if err != nil {
		logError("Error creating Gist request: %v", err)
		return
	}
	req.Header.Set("User-Agent", "ASF-FreeGamesClaimer-Go")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logError("Error fetching Gist: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logError("Gist request failed with status: %d", resp.StatusCode)
		return
	}

	var gist GistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		logError("Error decoding Gist response: %v", err)
		return
	}

	file, ok := gist.Files["Steam Codes"]
	if !ok {
		logError("Error: 'Steam Codes' file not found in Gist")
		return
	}

	// Split codes and filter empty lines
	rawCodes := strings.Split(file.Content, "\n")
	var codes []string
	for _, code := range rawCodes {
		trimmed := strings.TrimSpace(code)
		if trimmed != "" {
			codes = append(codes, trimmed)
		}
	}

	lastLength := readLastLength()
	totalCodes := len(codes)

	if lastLength < totalCodes {
		lastLengthBeforeRun := lastLength
		if lastLength+40 < totalCodes {
			logWarning("Only runs on the last 40 games")
			lastLength = totalCodes - 40
		}

		// Retrieve environment variables with defaults
		asfPort := getEnv("ASF_PORT", "1242")
		asfHost := getEnv("ASF_HOST", "localhost")
		password := getEnv("ASF_PASSWORD", "")
		commandPrefix := getEnv("ASF_COMMAND_PREFIX", "!")
		asfHTTPSEnv := getEnv("ASF_HTTPS", "false")
		asfHTTPS := asfHTTPSEnv == "true"
		asfBots := getEnv("ASF_BOTS", "asf")

		// 2. Build the ASF IPC command
		asfCommand := fmt.Sprintf("%saddlicense %s ", commandPrefix, asfBots)
		var codesToClaim []string
		for i := lastLength; i < totalCodes; i++ {
			codesToClaim = append(codesToClaim, codes[i])
		}
		asfCommand += strings.Join(codesToClaim, ",")

		// 3. Send the command to ASF IPC
		scheme := "http"
		if asfHTTPS {
			scheme = "https"
		}
		asfURL := fmt.Sprintf("%s://%s:%s/Api/Command", scheme, asfHost, asfPort)

		asfReqBody, err := json.Marshal(ASFRequest{Command: asfCommand})
		if err != nil {
			logError("Error marshalling ASF request body: %v", err)
			return
		}

		req, err = http.NewRequest("POST", asfURL, bytes.NewBuffer(asfReqBody))
		if err != nil {
			logError("Error creating ASF request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if password != "" {
			req.Header.Set("Authentication", password)
		}

		asfResp, err := http.DefaultClient.Do(req)
		if err != nil {
			logError("Error running command '%s': %v", asfCommand, err)
			logWarning("Trying again in six hours")
			return
		}
		defer asfResp.Body.Close()

		var body ASFResponse
		if err := json.NewDecoder(asfResp.Body).Decode(&body); err != nil {
			logError("Error decoding ASF response: %v", err)
			return
		}

		if body.Success {
			logSuccess("Command sent successfully: %s", asfCommand)
			logInfo("Details: %s", string(body.Result))
			writeLastLength(totalCodes)
		} else {
			logError("ASF Error: Success: false. Message: %s. Result: %s", body.Message, string(body.Result))
			// Rollback to retry next time
			writeLastLength(lastLengthBeforeRun)
		}
	} else {
		logInfo("No new games. Found in Gist: %d, already claimed: %d", totalCodes, lastLength)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
