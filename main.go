package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"go.yaml.in/yaml/v4"
)

func main() {
	data, err := os.ReadFile("commands.yaml")
	if err != nil {
		fmt.Println("Error: Could not find saved commands.")
		os.Exit(1)
	}

	var commands Commands
	err = yaml.Unmarshal(data, &commands)
	if err != nil {
		fmt.Println("Error: Could not parse saved commands.")
		os.Exit(1)
	}
	if len(commands.Commands) == 0 {
		fmt.Println("Error: No saved commands available. Please save a command first in comands.yaml.")
		os.Exit(1)
	}

	args := os.Args[1:]

	if len(args) == 0 || len(args) > 1 {
		fmt.Println("Error: Must supply exactly one search phrase in quotes")
		os.Exit(1)
	}

	search_phrase := args[0]

	search_phrase_ranked_command_matches := match_phrase_to_commands_and_rank(search_phrase, commands)

	if len(search_phrase_ranked_command_matches.CommandMatches) == 0 {
		fmt.Printf("Could not find any stored commands which match '%s'", search_phrase)
		return
	}

	fmt.Printf("Commands that match the phrase: '%s' ranked by similarity\n\n", search_phrase)

	output_table_writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(output_table_writer, "COMMAND\tMATCH_SCORE")

	for _, match := range search_phrase_ranked_command_matches.CommandMatches {
		_, _ = fmt.Fprintf(output_table_writer, "%s\t%d\n", match.Command.Cmd, match.MatchScore)
	}

	_ = output_table_writer.Flush()
}

func match_phrase_to_commands_and_rank(phrase string, commands Commands) SearchPhraseCommandMatches {
	phrase_tokens := strings.Split(phrase, " ")

	matches := SearchPhraseCommandMatches{SearchPhrase: phrase}

	for _, command := range commands.Commands {
		var command_max_match_score int

		for _, cmd_phrase := range command.Phrases {
			match_score := 0

			cmd_tokens := strings.Split(cmd_phrase, " ")

			for _, phrase_token := range phrase_tokens {
				if slices.Contains(cmd_tokens, phrase_token) {
					match_score += 1
				}
			}
			if match_score == 0 {
				continue
			}
			command_max_match_score = max(command_max_match_score, match_score)
		}

		if command_max_match_score > 0 {
			cmd_match := CommandMatch{Command: command, MatchScore: command_max_match_score}
			matches.CommandMatches = append(matches.CommandMatches, cmd_match)
		}
	}

	slices.SortFunc(matches.CommandMatches, func(a, b CommandMatch) int {
		return cmp.Compare(b.MatchScore, a.MatchScore)
	})

	return matches
}

type Command struct {
	Cmd     string
	Phrases []string
	Tags    []string
}

type Commands struct {
	Commands []Command
}

type SearchPhraseCommandMatches struct {
	SearchPhrase   string
	CommandMatches []CommandMatch
}

type CommandMatch struct {
	Command    Command
	MatchScore int
}
