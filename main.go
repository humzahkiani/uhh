// Command uhh searches saved shell commands by phrase and prints matches ranked by similarity.
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

	searchPhrase := args[0]

	searchPhraseRankedCommandMatches := matchPhraseToCommandsAndRank(searchPhrase, commands)

	if len(searchPhraseRankedCommandMatches.CommandMatches) == 0 {
		fmt.Printf("Could not find any stored commands which match '%s'", searchPhrase)
		return
	}

	fmt.Printf("Commands that match the phrase: '%s' ranked by similarity\n\n", searchPhrase)

	outputTableWriter := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(outputTableWriter, "COMMAND\tMATCH_SCORE")

	for _, match := range searchPhraseRankedCommandMatches.CommandMatches {
		_, _ = fmt.Fprintf(outputTableWriter, "%s\t%d\n", match.Command.Cmd, match.MatchScore)
	}

	_ = outputTableWriter.Flush()
}

func matchPhraseToCommandsAndRank(phrase string, commands Commands) SearchPhraseCommandMatches {
	phraseTokens := strings.Split(phrase, " ")

	matches := SearchPhraseCommandMatches{SearchPhrase: phrase}

	for _, command := range commands.Commands {
		var commandMaxMatchScore int

		for _, cmdPhrase := range command.Phrases {
			matchScore := 0

			cmdTokens := strings.Split(cmdPhrase, " ")

			for _, phraseToken := range phraseTokens {
				if slices.Contains(cmdTokens, phraseToken) {
					matchScore++
				}
			}
			if matchScore == 0 {
				continue
			}
			commandMaxMatchScore = max(commandMaxMatchScore, matchScore)
		}

		if commandMaxMatchScore > 0 {
			cmdMatch := CommandMatch{Command: command, MatchScore: commandMaxMatchScore}
			matches.CommandMatches = append(matches.CommandMatches, cmdMatch)
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
