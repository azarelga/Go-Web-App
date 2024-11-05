package main

import (
	"encoding/json"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

// Card represents either a prompt or response card
type Card struct {
	Content string
}

// GameState represents the current game state
type GameState struct {
	CurrentPrompt Card
	Responses     []Card
	History       []Submission
	Message       string
}

// Submission represents a saved prompt-response pair
type Submission struct {
	Timestamp time.Time
	Prompt    string
	Response  string
}

// Global variables
var (
	history []Submission
	mutex   sync.Mutex
)

// LoadCards loads cards from a JSON file
func LoadCards(filename string) (map[string]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cards map[string]string
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, err
	}

	return cards, nil
}

// GetRandomCards returns n random cards from the provided map
func GetRandomCards(cards map[string]string, n int) []Card {
	values := make([]string, 0, len(cards))
	for _, v := range cards {
		values = append(values, v)
	}

	rand.Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})

	result := make([]Card, n)
	for i := 0; i < n && i < len(values); i++ {
		result[i] = Card{Content: values[i]}
	}
	return result
}

// SaveSubmission adds a new submission to history
func SaveSubmission(prompt, response string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	isAdd := false
	submission := Submission{
		Timestamp: time.Now(),
		Prompt:    prompt,
		Response:  response,
	}

	if len(history) > 0 {
		if history[0].Prompt != prompt || history[0].Response != response {
			history = append([]Submission{submission}, history...) // Prepend to show newest first
			isAdd = true
		}
	} else {
		history = append([]Submission{submission}, history...) // Prepend to show newest first
		isAdd = true
	}
	// Keep only last 100 submissions
	if len(history) > 100 {
		history = history[:100]
	}
	return isAdd
}

func main() {
	prompts, err := LoadCards("asset/prompt.json")
	if err != nil {
		panic(err)
	}

	responses, err := LoadCards("asset/response.json")
	if err != nil {
		panic(err)
	}

	tmpl, _ := template.ParseFiles("template/index.html")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var gameState GameState
		var message string

		if r.Method == "POST" {
			r.ParseForm()
			action := r.FormValue("action")
			prompt := r.FormValue("prompt")
			response := r.FormValue("response")

			if action == "submit" && response != "" {
				if SaveSubmission(prompt, response) {
					message = "Your answer has been submitted!"
				}
			}
		}

		promptCards := GetRandomCards(prompts, 1)
		responseCards := GetRandomCards(responses, 6)

		mutex.Lock()
		gameState = GameState{
			CurrentPrompt: promptCards[0],
			Responses:     responseCards,
			History:       history,
			Message:       message,
		}
		mutex.Unlock()

		tmpl.Execute(w, gameState)
	})

	println("Server starting on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
