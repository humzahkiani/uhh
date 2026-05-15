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

type command struct {
	Cmd     string
	Phrases []string
}

var (
	errCommandBlankPhrase = errors.New("command phrase is empty/whitespace-only")
	errCommandEmptyCmd    = errors.New("command cmd is empty/whitespace-only")
	errCommandNoPhrases   = errors.New("command has no phrases")
)

func newCommand(cmd string, phrases []string) (command, error) {
	cmd = strings.TrimSpace(cmd)
	if len(cmd) == 0 {
		return command{}, errCommandEmptyCmd
	}

	trimmed := make([]string, 0, len(phrases))
	for _, p := range phrases {
		p := strings.TrimSpace(p)
		if len(p) != 0 {
			trimmed = append(trimmed, p)
			continue
		}
		return command{}, errCommandBlankPhrase
	}
	if len(trimmed) == 0 {
		return command{}, errCommandNoPhrases
	}
	return command{Cmd: cmd, Phrases: trimmed}, nil
}

type commandMatches struct {
	searchPhrase string
	matches      []commandMatch
}

type commandMatch struct {
	command    command
	matchScore int
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
	default: // implicit find command
		fs := flag.NewFlagSet("find", flag.ContinueOnError)
		_ = fs.Parse(args[0:])
		if len(fs.Args()) != 1 {
			return errors.New("unexpected no. of positional args; provide exactly one search phrase argument in quotations")
		}
		searchPhrase := fs.Args()[0]
		err := find(searchPhrase)
		if err != nil {
			return err
		}
	}

	return nil
}

func find(searchPhrase string) error {
	savedCommands, err := readSavedCommands()
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}

	if len(savedCommands) == 0 {
		return errors.New("find: no saved commands found")
	}

	matchedAndRankedCommands := matchAndRank(searchPhrase, savedCommands)

	if len(matchedAndRankedCommands.matches) == 0 {
		return errNoMatches
	}

	fmt.Printf("Commands that match the phrase: '%s' ranked by similarity\n\n", matchedAndRankedCommands.searchPhrase)

	outputTableWriter := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(outputTableWriter, "COMMAND\tMATCH_SCORE")

	for _, match := range matchedAndRankedCommands.matches {
		_, _ = fmt.Fprintf(outputTableWriter, "%s\t%d\n", match.command.Cmd, match.matchScore)
	}

	return outputTableWriter.Flush()
}

func save(cmd string, phrases []string) error {
	savedCommands, err := readSavedCommands()
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	for _, c := range savedCommands {
		if strings.EqualFold(strings.TrimSpace(c.Cmd), strings.TrimSpace(cmd)) {
			return errors.New("save: command already exists")
		}
	}

	created, err := newCommand(cmd, phrases)
	if err != nil {
		return fmt.Errorf("save: create new command: %w", err)
	}
	updatedCommands := append(savedCommands, created)

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

	fmt.Printf("successfully saved command: '%v'\n", created.Cmd)
	return nil
}

func readSavedCommands() ([]command, error) {
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
	var raw []struct {
		Cmd     string
		Phrases []string
	}
	err = yaml.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("read saved commands: %w", err)
	}

	saved := make([]command, 0, len(raw))
	for _, r := range raw {
		c, err := newCommand(r.Cmd, r.Phrases)
		if err != nil {
			return nil, fmt.Errorf("read saved commands: invalid command in file: %w", err)
		}
		saved = append(saved, c)
	}
	return saved, nil
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

func matchAndRank(phrase string, commands []command) commandMatches {
	matches := commandMatches{searchPhrase: phrase}

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
			cmdMatch := commandMatch{command: command, matchScore: commandMaxMatchScore}
			matches.matches = append(matches.matches, cmdMatch)
		}
	}

	slices.SortFunc(matches.matches, func(a, b commandMatch) int {
		return cmp.Compare(b.matchScore, a.matchScore)
	})

	return matches
}
