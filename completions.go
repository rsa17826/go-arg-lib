package argparse

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	completionShell string

	completionTimerMu sync.Mutex
	completionTimer   *time.Timer

	// doComplete is set when the hidden --complete flag is present.
	// It is checked directly from cachedArgs so it never appears in help.
	doComplete bool
)

func init() {
	// --completions is user-visible; register it so it shows up in --help.
	defs := []ArgumentData{
		{
			Keys:        []string{"completions"},
			AfterCount:  1,
			Target:      &completionShell,
			Description: "print shell completion script (bash|zsh|fish) and exit",
			ExampleArgs: []string{"shell"},
		},
	}

	mu.Lock()
	// Store alongside other argparse-internal defs (same key the help def uses
	// when getCallerPackageName is called from this package's own init).
	const internalKey = "argparse"
	ParseArgsData[internalKey] = append(ParseArgsData[internalKey], defs...)
	mu.Unlock()

	applyDefaults(defs)
	parseForDefs(cachedArgs, defs)

	// --complete is intentionally hidden: don't add it to ParseArgsData.
	for _, arg := range cachedArgs {
		if arg == "--complete" {
			doComplete = true
			break
		}
	}

	if completionShell != "" || doComplete {
		scheduleCompletionAction()
	}
}

// scheduleCompletionAction arms (or resets) a short timer, mirroring the help
// timer pattern so we only act after every package has called ParseArgs.
func scheduleCompletionAction() {
	completionTimerMu.Lock()
	defer completionTimerMu.Unlock()
	if completionTimer != nil {
		completionTimer.Reset(5 * time.Millisecond)
	} else {
		completionTimer = time.AfterFunc(5*time.Millisecond, func() {
			mu.RLock()
			defer mu.RUnlock()
			if doComplete {
				runComplete()
			} else {
				printCompletionScript(completionShell)
			}
			os.Exit(0)
		})
	}
}

// allFlags collects every registered flag (formatted as -f / --flag) across
// all packages, sorted and deduplicated.  The hidden --complete flag is
// excluded so it never appears as a suggestion.
func allFlags() []string {
	seen := map[string]struct{}{}
	for _, defs := range ParseArgsData {
		for _, def := range defs {
			for _, key := range def.Keys {
				prefix := "--"
				if len(key) == 1 {
					prefix = "-"
				}
				seen[prefix+key] = struct{}{}
			}
		}
	}
	flags := make([]string, 0, len(seen))
	for f := range seen {
		flags = append(flags, f)
	}
	sort.Strings(flags)
	return flags
}

// runComplete prints one candidate per line.  The shell's own prefix-matching
// (compgen -W / compadd / complete) does the filtering; we just supply the
// full candidate list.
func runComplete() {
	for _, f := range allFlags() {
		fmt.Println(f)
	}
}

// printCompletionScript writes a ready-to-source completion script for the
// requested shell to stdout.
func printCompletionScript(shell string) {
	bin := filepath.Base(os.Args[0])
	// Sanitise for use as a shell function name.
	fnName := "_" + strings.NewReplacer("-", "_", ".", "_").Replace(bin) + "_completions"

	switch strings.ToLower(shell) {
	case "bash":
		fmt.Printf(`# Bash completion for %s
# Usage: source <(%s --completions bash)
#   or add to ~/.bash_completion / /etc/bash_completion.d/
%s() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    COMPREPLY=($(compgen -W "$(%s --complete 2>/dev/null)" -- "$cur"))
    return 0
}
complete -o default -F %s %s
`, bin, bin, fnName, bin, fnName, bin)

	case "zsh":
		fmt.Printf(`# Zsh completion for %s
# Usage: source <(%s --completions zsh)
#   or save to a file in your $fpath, e.g. ~/.zfunc/_%s
#compdef %s
%s() {
    local -a flags
    flags=($(%s --complete 2>/dev/null))
    compadd -a flags
}
%s "$@"
`, bin, bin, bin, bin, fnName, bin, fnName)

	case "fish":
		fmt.Printf(`# Fish completion for %s
# Usage: %s --completions fish > ~/.config/fish/completions/%s.fish
complete -c %s -f -a "(%s --complete 2>/dev/null)"
`, bin, bin, bin, bin, bin)

	default:
		fmt.Fprintf(os.Stderr, "argparse: unknown shell %q — supported shells: bash, zsh, fish\n", shell)
		os.Exit(1)
	}
}
