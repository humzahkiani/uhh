// Command uhh searches saved shell commands by phrase and prints matches ranked by similarity.
package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

var errNoMatches = errors.New("no matches")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "uhh:", err)

		if errors.Is(err, errNoMatches) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run() error {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("locate home dir: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	path := filepath.Join(configDir, "uhh", "commands.yaml")

	data, err := os.ReadFile(path)

	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no command file found; create one at %s", path)
	}
	if err != nil {
		return fmt.Errorf("read commands file: %w", err)
	}

	var commands []Command
	err = yaml.Unmarshal(data, &commands)
	if err != nil {
		return fmt.Errorf("parse saved commands: %w", err)
	}
	if len(commands) == 0 {
		return fmt.Errorf("no saved commands found; add one to %s", path)
	}

	args := os.Args[1:]

	if len(args) != 1 {
		return errors.New("must supply exactly one search phrase in quotes")
	}

	searchPhrase := args[0]

	matchedAndRankedCommands := matchAndRank(searchPhrase, commands)

	if len(matchedAndRankedCommands.CommandMatches) == 0 {
		return errNoMatches
	}

	fmt.Printf("Commands that match the phrase: '%s' ranked by similarity\n\n", matchedAndRankedCommands.SearchPhrase)

	outputTableWriter := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(outputTableWriter, "COMMAND\tMATCH_SCORE")

	for _, match := range matchedAndRankedCommands.CommandMatches {
		_, _ = fmt.Fprintf(outputTableWriter, "%s\t%d\n", match.Command.Cmd, match.MatchScore)
	}

	return outputTableWriter.Flush()
}

func matchAndRank(phrase string, commands []Command) CommandMatches {
	matches := CommandMatches{SearchPhrase: phrase}

	phraseTokens := strings.Split(strings.ToLower(phrase), " ")

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
