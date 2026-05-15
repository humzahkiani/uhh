# uhh
A personal command discovery tool for those who also possess the memory of a goldfish

100% manually hand-coded by yours truly.

## Usage

Save commands in `~/.config/uhh/commands.yaml` (or `$XDG_CONFIG_HOME/uhh/commands.yaml` if set):

```yaml
- cmd: "ls"
  phrases:
    - "list all files/folders in current directory"
```

See `commands.example.yaml` for a fuller set of examples.

Then search by phrase:

```sh
uhh "list files"
```

Matches are scored by the number of shared tokens between your search phrase and each command's saved phrases, then printed in a table sorted by score.

### Saving commands

Add a new command from the CLI with `save`:

```sh
uhh save --cmd "ls" --phrases "list all files/folders in current directory,list files"
```

`--phrases` takes a comma-delimited list. The command is appended to your `commands.yaml`; duplicate commands (case-insensitive, trimmed) are rejected.
