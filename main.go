// Command uhh searches saved shell commands by phrase and prints matches ranked by similarity.
package main

import (
	"cmp"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"go.yaml.in/yaml/v4"
)

type Command struct {
	Cmd     string
	Phrases []string
}

type CommandMatches struct {
	SearchPhrase   string
	CommandMatches []CommandMatch
}

type CommandMatch struct {
	Command    Command
	MatchScore int
}

func main() {
	data, err := os.ReadFile("commands.yaml")
	if err != nil {
		log.Fatal("Could not find saved commands.")
	}

	var commands []Command
	err = yaml.Unmarshal(data, &commands)
	if err != nil {
		log.Fatalf("Could not parse saved commands: %v", err)
	}
	if len(commands) == 0 {
		log.Fatal("No saved commands found. Please save a command first in `commands.yaml`.")
	}

	args := os.Args[1:]

	if len(args) != 1 {
		log.Fatal("Must supply exactly one search phrase in quotes")
	}

	searchPhrase := args[0]

	matchedAndRankedCommands := matchAndRank(searchPhrase, commands)

	if len(matchedAndRankedCommands.CommandMatches) == 0 {
		fmt.Printf("Could not find any saved commands which match '%s'", matchedAndRankedCommands.SearchPhrase)
		return
	}

	fmt.Printf("Commands that match the phrase: '%s' ranked by similarity\n\n", matchedAndRankedCommands.SearchPhrase)

	outputTableWriter := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(outputTableWriter, "COMMAND\tMATCH_SCORE")

	for _, match := range matchedAndRankedCommands.CommandMatches {
		_, _ = fmt.Fprintf(outputTableWriter, "%s\t%d\n", match.Command.Cmd, match.MatchScore)
	}

	_ = outputTableWriter.Flush()
}

func matchAndRank(phrase string, commands []Command) CommandMatches {
	matches := CommandMatches{SearchPhrase: phrase}

	phraseLower := strings.ToLower(phrase)
	phraseTokens := strings.Split(phraseLower, " ")

	for _, command := range commands {
		var commandMaxMatchScore int

		for _, cmdPhrase := range command.Phrases {
			matchScore := 0
			cmdPhrase = strings.ToLower(cmdPhrase)

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
