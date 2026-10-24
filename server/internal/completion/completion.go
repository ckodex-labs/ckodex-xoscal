package completion

import (
	"fmt"
	"io"
	"strings"
)

// SupportedShells lists shells for which autocompletion can be generated.
var SupportedShells = []string{"bash", "zsh", "fish"}

// CommonFrameworkIDs provides canonical framework IDs for completion.
var CommonFrameworkIDs = []string{
	"cccs-itsp-10-033",
	"cccs-itsg-33",
	"cybersecure-canada",
	"cccs-itsp-10-171",
	"cccs-medium-cloud-pbmm",
}

// CommandsList provides primary subcommands for completion.
var CommandsList = []string{
	"init",
	"generate",
	"ingest",
	"verify",
	"check",
	"validate",
	"diff",
	"derogate",
	"tabular",
	"bundle-export",
	"export-bundle",
	"frameworks",
	"controls",
	"compare",
	"serve",
	"ui",
	"quickstart",
	"completion",
	"seed",
	"version",
}

// Generate writes the autocompletion script for the target shell to w.
func Generate(shell, binaryName string, w io.Writer) error {
	switch strings.ToLower(shell) {
	case "bash":
		return generateBash(binaryName, w)
	case "zsh":
		return generateZsh(binaryName, w)
	case "fish":
		return generateFish(binaryName, w)
	default:
		return fmt.Errorf("unsupported shell '%s'; supported shells are: %s", shell, strings.Join(SupportedShells, ", "))
	}
}

func generateBash(bin string, w io.Writer) error {
	script := fmt.Sprintf(`_%s_completions() {
    local cur prev words cword
    _init_completion || return

    local commands="%s"
    local frameworks="%s"

    if [[ ${cword} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return 0
    fi

    case "${prev}" in
        -framework)
            COMPREPLY=( $(compgen -W "${frameworks}" -- "${cur}") )
            return 0
            ;;
        -jurisdiction)
            COMPREPLY=( $(compgen -W "ca us eu" -- "${cur}") )
            return 0
            ;;
        -kind)
            COMPREPLY=( $(compgen -W "catalog profile ssp component-definition assessment-plan assessment-results poam mapping" -- "${cur}") )
            return 0
            ;;
        -file|-base|-head|-in|-out|-db|-dsn)
            _filedir
            return 0
            ;;
    esac

    case "${words[1]}" in
        frameworks)
            COMPREPLY=( $(compgen -W "list export compare seed" -- "${cur}") )
            return 0
            ;;
        controls)
            COMPREPLY=( $(compgen -W "search get list" -- "${cur}") )
            return 0
            ;;
        derogate)
            COMPREPLY=( $(compgen -W "add list revoke" -- "${cur}") )
            return 0
            ;;
        tabular)
            COMPREPLY=( $(compgen -W "export import" -- "${cur}") )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") )
            return 0
            ;;
    esac
}
complete -F _%s_completions %s
`, bin, strings.Join(CommandsList, " "), strings.Join(CommonFrameworkIDs, " "), bin, bin)

	_, err := io.WriteString(w, script)
	return err
}

func generateZsh(bin string, w io.Writer) error {
	script := fmt.Sprintf(`#compdef %s

_%s() {
    local -a commands
    commands=(
        'init:Scaffold OSCAL Component Definition based on repository inspection'
        'generate:Generate OSCAL 1.2.3 artifacts from Knowledge Graph snapshot'
        'ingest:Ingest raw regulatory requirements JSON into Knowledge Graph'
        'verify:Evaluate artifact schema and compute multi-dimensional Vector State'
        'validate:Fast standalone validation of OSCAL JSON artifacts'
        'diff:Compute semantic compliance delta between base and PR head'
        'derogate:Manage formal time-bounded risk derogations'
        'tabular:Convert between OSCAL JSON and CSV/Excel tables'
        'bundle-export:Package verifiable audit bundle with offline HTML viewer'
        'frameworks:List, export, or compare authoritative compliance frameworks'
        'controls:Search and inspect controls across registered frameworks'
        'compare:Compare two frameworks and compute sovereign transition delta'
        'serve:Start local zero-dependency xOSCAL Web Studio & Explorer'
        'ui:Launch local xOSCAL Web Studio in default browser'
        'quickstart:Guided onboarding walkthrough'
        'completion:Generate shell autocompletions (bash, zsh, fish)'
        'seed:Seed Knowledge Graph with authoritative baseline frameworks'
        'version:Print version information'
    )

    if (( CURRENT == 2 )); then
        _describe -t commands '%s command' commands
        return
    fi

    case "$words[2]" in
        frameworks)
            local -a fw_cmds
            fw_cmds=(
                'list:List registered frameworks'
                'export:Export framework catalogs and profiles'
                'compare:Compare two frameworks'
                'seed:Seed database with baseline frameworks'
            )
            _describe -t fw_cmds 'frameworks subcommand' fw_cmds
            ;;
        controls)
            local -a ctrl_cmds
            ctrl_cmds=(
                'search:Search controls matching query'
                'get:Display detailed control card'
                'list:List all controls in framework'
            )
            _describe -t ctrl_cmds 'controls subcommand' ctrl_cmds
            ;;
        completion)
            _values 'shell' 'bash' 'zsh' 'fish'
            ;;
        *)
            _files
            ;;
    esac
}

_%s "$@"
`, bin, bin, bin, bin)

	_, err := io.WriteString(w, script)
	return err
}

func generateFish(bin string, w io.Writer) error {
	script := fmt.Sprintf(`complete -c %s -f
complete -c %s -n "__fish_use_subcommand" -a "init" -d "Scaffold OSCAL Component Definition"
complete -c %s -n "__fish_use_subcommand" -a "generate" -d "Generate OSCAL 1.2.3 artifacts"
complete -c %s -n "__fish_use_subcommand" -a "verify" -d "Evaluate Vector State"
complete -c %s -n "__fish_use_subcommand" -a "validate" -d "Validate OSCAL JSON schemas"
complete -c %s -n "__fish_use_subcommand" -a "diff" -d "GitOps semantic compliance diff"
complete -c %s -n "__fish_use_subcommand" -a "derogate" -d "Manage formal risk derogations"
complete -c %s -n "__fish_use_subcommand" -a "tabular" -d "Bridge OSCAL and CSV tables"
complete -c %s -n "__fish_use_subcommand" -a "bundle-export" -d "Package air-gapped audit bundle"
complete -c %s -n "__fish_use_subcommand" -a "frameworks" -d "List, export, compare frameworks"
complete -c %s -n "__fish_use_subcommand" -a "controls" -d "Search and inspect controls"
complete -c %s -n "__fish_use_subcommand" -a "compare" -d "Compare frameworks transition delta"
complete -c %s -n "__fish_use_subcommand" -a "serve" -d "Start local Web Studio"
complete -c %s -n "__fish_use_subcommand" -a "ui" -d "Open local Web Studio in browser"
complete -c %s -n "__fish_use_subcommand" -a "quickstart" -d "Guided onboarding tour"
complete -c %s -n "__fish_use_subcommand" -a "completion" -d "Generate shell completion"
complete -c %s -n "__fish_use_subcommand" -a "version" -d "Print version"
`, bin, bin, bin, bin, bin, bin, bin, bin, bin, bin, bin, bin, bin, bin, bin, bin, bin)

	_, err := io.WriteString(w, script)
	return err
}
