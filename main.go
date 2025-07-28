package main

import (
        "bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
        "math/rand"
	"strings"
        "time"
        "github.com/TrollsonSpeaks/Pok-dex/internal/pokecache"
)

func cleanInput(input string) []string {
    input = strings.TrimSpace(input)
    input = strings.ToLower(input)
    return strings.Fields(input)
}

var cache *pokecache.Cache

type Config struct {
	NextURL     *string
	PreviousURL *string
        Pokedex     map[string]Pokemon
}

type cliCommand struct {
	name        string
	description string
	callback    func(*Config, ...string) error
}

type LocationResponse struct {
	Count    int `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
	} `json:"results"`
}

type Stat struct {
    Name      string
    BaseStat  int
}

type Pokemon struct {
    Name           string
    BaseExperience int
    Height         int
    Weight         int
    Stats          []Stat
    Types          []string   
}

var pokedex = map[string]Pokemon{}

var commands = make(map[string]cliCommand)

func main() {
        cache = pokecache.NewCache(5 * time.Second)

	config := &Config{}

        config.Pokedex = make(map[string]Pokemon)

	commands["help"] = cliCommand{
		name:        "help",
		description: "Show all available commands",
		callback: func(config *Config, args ...string) error {
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
		callback: func(config *Config, args ...string) error {
			os.Exit(0)
			return nil
		},
	}

	commands["map"] = cliCommand{
		name:        "map",
		description: "Show the next 20 Pokémon location areas",
		callback:    func(config *Config, args ...string) error {
                    return mapCommand(config)
	        },
        }
   
	commands["mapb"] = cliCommand{
		name:        "mapb",
		description: "Show the previous 20 Pokémon location areas",
		callback:    func(config *Config, args ...string) error {
                    return mapCommand(config)
	        },
        }

        commands["explore"] = cliCommand{
            name:        "explore",
            description: "Explore a specific location area to see Pokémon",
            callback:    exploreCommand,
        }

        commands["catch"] = cliCommand{
            name:        "catch",
            description: "Try to catch a Pokémon by name",
            callback:    catchCommand,
        }

        commands["inspect"] = cliCommand{
            name:        "inspect",
            description: "Inspect a caught Pokemon's details",
            callback:    inspectCommand,
        }

        commands["pokedex"] = cliCommand{
            name:        "pokedex",
            description: "List all caught Pokemon",
            callback:    pokedexCommand,
        }

        scanner := bufio.NewScanner(os.Stdin)

	for {
	    fmt.Print("Pokedex > ")

            if !scanner.Scan() {
                break
            }

            line := scanner.Text()
            words := cleanInput(line)

            if len(words) == 0 {
                continue
            }

            cmdName := words[0]
            args := words[1:]

            if cmd, ok := commands[cmdName]; ok {
                if err := cmd.callback(config, args...); err != nil {
		    fmt.Println("Error:", err)
		}
	    } else {
		fmt.Println("Unknown command")
	   }
     }
}

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

func exploreCommand(config *Config, args ...string) error {
    if len(args) < 1 {
        fmt.Println("Please provide a location area name. Example: explore pastoria-city-area")
        return nil
    }

    area := args[0]
    url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s", area)

    if data, found := cache.Get(url); found {
        fmt.Println("Using cached data for:", area)
        return printPokemonFromLocation(data)
    }

    fmt.Println("Fetching data for:", area)
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("Error fetching data:", err)
        return err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("Error reading response:", err)
        return err
    }

    cache.Add(url, body)

    return printPokemonFromLocation(body)
}

func printPokemonFromLocation(data []byte) error {
    var parsed struct {
        PokemonEncounters []struct {
            Pokemon struct {
                Name string `json:"name"`
            } `json:"pokemon"`
        } `json:"pokemon_encounters"`
    }

    if err := json.Unmarshal(data, &parsed); err != nil {
        fmt.Println("Error parsing data:", err)
        return err
    }

    if len(parsed.PokemonEncounters) == 0 {
        fmt.Println("No Pokémon found in this area.")
        return nil
    }

   fmt.Println("Found Pokémon:")
   for _, encounter := range parsed.PokemonEncounters {
       fmt.Printf(" - %s\n", encounter.Pokemon.Name)
   }

   return nil
}

func catchCommand(config *Config, args ...string) error {
    if len(args) < 1 {
        fmt.Println("You must specify the name of the Pokemon to catch .")
        return nil
    }

    pokemonName := strings.ToLower(args[0])
    fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)

    url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", pokemonName)
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("Failed to reach the Pokemon API:", err)
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        fmt.Printf("%s could not be found!\n", pokemonName)
        return nil
    }

    var result struct {
        Name           string `json:"name"`
        BaseExperience int    `json:"base_experience"`
        Height         int    `json:"height"`
        Weight         int    `json:"weight"`
        Stats []struct {
            BaseStat   int `json:"base_stat"`
            Stat struct {
                Name string `json:"name"`
            } `json:"stat"`
        } `json:"stats"`
        Types []struct {
            Type struct {
                Name string `json:"name"`
            } `json:"type"`
        } `json:"types"`
    }

    err = json.NewDecoder(resp.Body).Decode(&result)
    if err != nil {
        fmt.Println("Error decoding response:", err)
        return err
    }

    rand.Seed(time.Now().UnixNano())
    catchChance := 100 - result.BaseExperience
    if catchChance < 10 {
        catchChance = 10
    }

    roll := rand.Intn(100) + 1
    if roll <= catchChance {
        fmt.Printf("%s was caught!\n", result.Name)
        fmt.Println("You may now inspect it with the inspect command.")

        stats := []Stat{}
        for _, s := range result.Stats {
            stats = append(stats, Stat{
                Name:      s.Stat.Name,
                BaseStat:  s.BaseStat,
            })
        }

        types := []string{}
        for _, t := range result.Types {
            types = append(types, t.Type.Name)
        }

        config.Pokedex[result.Name] = Pokemon{
            Name:            result.Name,
            BaseExperience:  result.BaseExperience,
            Height:          result.Height,
            Weight:          result.Weight,
            Stats:           stats,
            Types:           types,
        }
    } else {
        fmt.Printf("%s escaped!\n", result.Name)
    }

    return nil

}


func inspectCommand(cfg *Config, args ...string) error {
    if len(args) < 1 {
        fmt.Println("Please provide the name of the Pokemon to inspect.")
        return nil
    }

    name := strings.ToLower(args[0])
    p, ok := cfg.Pokedex[name]
    if !ok {
        fmt.Println("you have not caught that pokemon")
        return nil
    }

    fmt.Println("Name:", p.Name)
    fmt.Println("Height:", p.Height)
    fmt.Println("Weight:", p.Weight)

    fmt.Println("Stats:")
    for _, stat := range p.Stats {
        fmt.Printf("  -%s: %d\n", stat.Name, stat.BaseStat)
    }

    fmt.Println("Types:")
    for _, t := range p.Types {
        fmt.Printf("  - %s\n", t)
    }

    return nil

}

func pokedexCommand(cfg *Config, args ...string) error {
    if len(cfg.Pokedex) == 0 {
        fmt.Println("Your Pokedex is empty.")
        return nil
    }

    fmt.Println("Your Pokedex:")
    for name := range cfg.Pokedex {
        fmt.Printf(" - %s\n", name)
    }

    return nil

}
