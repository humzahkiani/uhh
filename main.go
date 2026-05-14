// Command uhh searches saved shell commands by phrase and prints matches ranked by similarity.
package main

import (
	"cmp"
	"errors"
	"flag"
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
	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("no args provided")
	}

	// cli subcommand/arg handling
	switch args[0] {
	case "save":
		fs := flag.NewFlagSet("save", flag.ContinueOnError)
		cmd := fs.String("cmd", "", "command shell code")
		phrases := fs.String("phrases", "", "comma-delimited phrases that describe what the command does")

		err := fs.Parse(args[1:])
		if err != nil {
			return err
		}

		if len(fs.Args()) > 0 {
			return fmt.Errorf("unexpected positional argument(s): %q", fs.Args())
		}
		if *cmd == "" || *phrases == "" {
			return fmt.Errorf("all required args not provided: --cmd, --phrases ")
		}

		err = save(*cmd, strings.Split(*phrases, ","))
		if err != nil {
			return err
		}
		return nil
	default:
		fs := flag.NewFlagSet("find", flag.ContinueOnError)
		_ = fs.Parse(args[0:])
		if len(fs.Args()) != 1 {
			fmt.Println(fs.Args())
			return errors.New("unexpected no. of positional args; provide exactly one search phrase argument in quotations")
		}
	}

	// find command (implicit default)

	savedCommands, err := readSavedCommands()
	if err != nil {
		return err
	}

	if len(savedCommands) == 0 {
		return errors.New("no saved commands found")
	}

	searchPhrase := args[0]
	matchedAndRankedCommands := matchAndRank(searchPhrase, savedCommands)

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

func save(command string, phrases []string) error {
	if strings.TrimSpace(command) == "" {
		return errors.New("save: input command is empty/whitespace only")
	}

	for _, p := range phrases {
		if strings.TrimSpace(p) == "" {
			return errors.New("save: one or more phrases is empty/whitespace only")
		}
	}

	savedCommands, err := readSavedCommands()
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	for _, c := range savedCommands {
		if strings.EqualFold(strings.TrimSpace(c.Cmd), strings.TrimSpace(command)) {
			return errors.New("save: command already exists")
		}
	}

	newCommand := Command{Cmd: command, Phrases: phrases}
	updatedCommands := append(savedCommands, newCommand)

	data, err := yaml.Marshal(updatedCommands)
	if err != nil {
		return fmt.Errorf("save: marshal commands: %w", err)
	}

	path, err := getCommandsFilePath()
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save: write commands file: %w", err)
	}

	fmt.Printf("successfully saved command: '%v'\n", newCommand.Cmd)
	return nil
}

func readSavedCommands() ([]Command, error) {
	path, err := getCommandsFilePath()
	if err != nil {
		return nil, fmt.Errorf("read saved commands: %w", err)
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read saved commands: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("read saved commands: %w", err)
	}
	var commands []Command
	err = yaml.Unmarshal(data, &commands)
	if err != nil {
		return nil, fmt.Errorf("read saved commands: %w", err)
	}
	return commands, nil
}

func getCommandsFilePath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate home dir: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	path := filepath.Join(configDir, "uhh", "commands.yaml")

	return path, nil
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
