package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
        "time"
        "github.com/TrollsonSpeaks/pokedex/internal/pokecache"
)

// Config stores pagination state
type Config struct {
	NextURL     *string
	PreviousURL *string
}

// Struct for commands
type cliCommand struct {
	name        string
	description string
	callback    func(*Config) error
}

// Struct to decode API response
type LocationResponse struct {
	Count    int `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
	} `json:"results"`
}

// Global command map
var commands = make(map[string]cliCommand)

func main() {
        cache := pokecache.NewCache(5 * time.Second)

	config := &Config{}

	// Register commands
	commands["help"] = cliCommand{
		name:        "help",
		description: "Show all available commands",
		callback: func(config *Config) error {
			fmt.Println("Available commands:")
			for _, cmd := range commands {
				fmt.Printf("  %s - %s\n", cmd.name, cmd.description)
			}
			return nil
		},
	}

	commands["exit"] = cliCommand{
		name:        "exit",
		description: "Exit the Pokédex",
		callback: func(config *Config) error {
			os.Exit(0)
			return nil
		},
	}

	commands["map"] = cliCommand{
		name:        "map",
		description: "Show the next 20 Pokémon location areas",
		callback:    mapCommand,
	}

	commands["mapb"] = cliCommand{
		name:        "mapb",
		description: "Show the previous 20 Pokémon location areas",
		callback:    mapBackCommand,
	}

	// CLI input loop
	for {
		fmt.Print("Pokedex > ")
		var input string
		fmt.Scanln(&input)
		input = strings.ToLower(strings.TrimSpace(input))

		if cmd, ok := commands[input]; ok {
			err := cmd.callback(config)
			if err != nil {
				fmt.Println("Error:", err)
			}
		} else {
			fmt.Println("Unknown command")
		}
	}
}

// Show next 20 locations
func mapCommand(config *Config) error {
	url := "https://pokeapi.co/api/v2/location-area/"
	if config.NextURL != nil {
		url = *config.NextURL
	}

        if data, found := cache.Get(url); found {
            fmt.Println("Using cached data for:", url)

            var parsed LocationResponse
            if err := json.Unmarshal(data, &parsed); err != nil {
                fmt.Println("Error parsing cached data:", err)
                return err
            }

            for _, loc := range parsed.Results {
                fmt.Println(loc.Name)
            }

            config.NextURL = parsed.Next
            config.PreviousURL = parsed.Previous
            return nil

        }

        fmt.Println("Fetching from PokeAPI:", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching data:", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

        cache.Add(url, body)

	var data LocationResponse
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Error parsing data:", err)
		return err
	}

	for _, loc := range data.Results {
		fmt.Println(loc.Name)
	}

	config.NextURL = data.Next
	config.PreviousURL = data.Previous
	return nil
}

// Show previous 20 locations
func mapBackCommand(config *Config) error {
	if config.PreviousURL == nil {
		fmt.Println("you're on the first page")
		return nil
	}

	url := *config.PreviousURL
	
        if data, found := cache.Get(url); found {
            fmt.Println("Using cached data for:", url)

            var parsed LocationResponse
            if err := json.Unmarshal(data, &parsed); err != nil {
                fmt.Println("Error parsing cached data:", err)
                return err
            }

            for _, loc := range parsed.Results {
                fmt.Println(loc.Name)
            }

            config.NextURL = parsed.Next
            config.PreviousURL = parsed.Previous
            return nil
        }

        fmt.Println("Fetching from PokeAPI:", url)
        resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching data:", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

        cache.Add(url, body)

	var data LocationResponse
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Error parsing data:", err)
		return err
	}

	for _, loc := range data.Results {
		fmt.Println(loc.Name)
	}

	config.NextURL = data.Next
	config.PreviousURL = data.Previous
	return nil
}
