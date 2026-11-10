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
	"nist-csf-2.0",
	"cmmc-2.0",
	"fedramp-rev5",
	"cisa-cpg-2.0",
	"cisa-vendor-scrm",
	"cjis-version-5.9.5",
	"dfs-500-2023-11",
	"cyber_essentials",
	"anssi-guide-hygiene",
	"anssi-guide-hygiene-detail",
	"anssi-guide-admin-securisee-si",
	"anssi-referentiel-general-de-securite-annexe-b2",
	"cnil-guide-securite",
	"3cf-v2",
	"bs-it-gs-2023-isms",
	"bs-it-gs-2023-orp",
	"bs-it-gs-2023-con",
	"bs-it-gs-2023-sys",
	"bs-it-gs-2023-app",
	"bs-it-gs-2023-inf",
	"bs-it-gs-2023-net",
	"bs-it-gs-2023-ops",
	"bs-it-gs-2023-ind",
	"bs-it-gs-2023-der",
	"bsi-c5-2020",
	"bsi-externer-cloud-dienste",
	"e-its-2024",
	"nis2",
	"dora",
	"eu-ai-act",
	"iso-iec-42001",
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
	"verify-parity",
	"controls",
	"compare",
	"ontology",
	"graph",
	"feeds",
	"serve",
	"ui",
	"quickstart",
	"doctor",
	"completion",
	"seed",
	"utcp",
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
        -framework|-from|-to)
            COMPREPLY=( $(compgen -W "${frameworks}" -- "${cur}") )
            return 0
            ;;
        -jurisdiction)
            COMPREPLY=( $(compgen -W "ca us usa uk fr france de germany eu" -- "${cur}") )
            return 0
            ;;
        -target)
            COMPREPLY=( $(compgen -W "go-utcp python-utcp typescript-utcp rs-utcp utcp-mcp CVE-2024-3400 T1190 cccs-itsg-33" -- "${cur}") )
            return 0
            ;;
        -format)
            COMPREPLY=( $(compgen -W "jsonld cytoscape cypher" -- "${cur}") )
            return 0
            ;;
        -feed)
            COMPREPLY=( $(compgen -W "cccs-nist-eu-crosswalk cisa-kev-active-threats mitre-attack-800-53 sovereign-regulatory-lifecycle" -- "${cur}") )
            return 0
            ;;
        -kind)
            COMPREPLY=( $(compgen -W "catalog profile ssp component-definition assessment-plan assessment-results poam mapping" -- "${cur}") )
            return 0
            ;;
        -file|-base|-head|-in|-out|-db|-dsn|-manual-file|-oscal-cli|-checkpoint-dir)
            _filedir
            return 0
            ;;
    esac

    case "${words[1]}" in
        frameworks)
            COMPREPLY=( $(compgen -W "list export compare seed verify -jurisdiction -framework -out -json" -- "${cur}") )
            return 0
            ;;
        controls)
            COMPREPLY=( $(compgen -W "search get list -framework -family -limit -json" -- "${cur}") )
            return 0
            ;;
        compare)
            COMPREPLY=( $(compgen -W "-from -to -markdown -json ${frameworks}" -- "${cur}") )
            return 0
            ;;
        ontology)
            COMPREPLY=( $(compgen -W "classes predicates spec -json" -- "${cur}") )
            return 0
            ;;
        graph)
            COMPREPLY=( $(compgen -W "stats crosswalk blast-radius export reconcile -control -target -depth -format -json" -- "${cur}") )
            return 0
            ;;
        feeds)
            COMPREPLY=( $(compgen -W "list sync inspect verify-checkpoint -feed -offline -checkpoint-dir -json" -- "${cur}") )
            return 0
            ;;
        generate)
            COMPREPLY=( $(compgen -W "-dsn -db -snapshot -framework -out -assessment-results-only -validate -validate-constraints -oscal-cli" -- "${cur}") )
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
        utcp)
            COMPREPLY=( $(compgen -W "manual validate mcp-config sdk ecosystem -target -out -base-url -grpc-host -manual-url -manual-file -file -json" -- "${cur}") )
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
        'verify-parity:Verify verbatim parity between framework and OSCAL artifacts'
        'controls:Search and inspect controls across registered frameworks'
        'compare:Compare two frameworks and compute sovereign transition delta'
        'ontology:Inspect formal OWL/JSON-LD regulatory & threat ontology'
        'graph:Query 36-framework property graph, crosswalks, and 6D blast radius'
        'feeds:List, sync, and inspect sovereign regulatory & threat intelligence feeds'
        'serve:Start local zero-dependency xOSCAL Web Studio & Explorer'
        'ui:Launch local xOSCAL Web Studio in default browser'
        'quickstart:Guided onboarding walkthrough'
        'doctor:Run deep runtime, schema, and registry diagnostics'
        'completion:Generate shell autocompletions (bash, zsh, fish)'
        'seed:Seed Knowledge Graph with authoritative baseline frameworks'
        'utcp:Generate, validate, and bridge Universal Tool Calling Protocol (UTCP) manuals & SDKs'
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
                'verify:Verify verbatim parity against OSCAL artifacts'
                '-jurisdiction:Filter by jurisdiction code'
                '-framework:Target framework identifier'
                '-out:Output directory'
                '-json:Output frameworks in JSON format'
            )
            _describe -t fw_cmds 'frameworks subcommand or flag' fw_cmds
            ;;
        ontology)
            local -a ont_cmds
            ont_cmds=(
                'classes:List 12 OWL/JSON-LD node classes'
                'predicates:List 16 typed semantic predicates'
                'spec:Inspect full OWL/JSON-LD ontology specification'
                '-json:Output as JSON-LD'
            )
            _describe -t ont_cmds 'ontology subcommand or flag' ont_cmds
            ;;
        graph)
            local -a graph_cmds
            graph_cmds=(
                'stats:Display 36-framework property graph statistics'
                'crosswalk:Query multi-jurisdiction control equivalency and harmonization'
                'blast-radius:Compute 6D Vector-State blast radius across frameworks'
                'export:Export subgraph as jsonld, cytoscape, or cypher'
                'reconcile:Verify Day-2 graph invariants, ontology triples, and SHA-256 coherence'
                '-control:Target control ID or framework:control'
                '-target:Target CVE, ATT&CK technique, framework, or control'
                '-depth:Maximum traversal depth'
                '-format:Export format (jsonld, cytoscape, cypher)'
                '-json:Output as JSON'
            )
            _describe -t graph_cmds 'graph subcommand or flag' graph_cmds
            ;;
        feeds)
            local -a feed_cmds
            feed_cmds=(
                'list:List authoritative sovereign and threat intelligence feeds'
                'sync:Synchronize feeds and emit SHA-256 execution receipts'
                'inspect:Inspect feed nodes, edges, and SHA-256 digest'
                'verify-checkpoint:Verify atomic SHA-256 receipt and hash-chained flight recorder'
                '-feed:Target feed ID'
                '-offline:Air-gap offline deterministic sync mode'
                '-checkpoint-dir:Directory for atomic sync checkpoints and flight recorder'
                '-json:Output as JSON'
            )
            _describe -t feed_cmds 'feeds subcommand or flag' feed_cmds
            ;;
        utcp)
            local -a utcp_cmds
            utcp_cmds=(
                'manual:Emit UTCP v1.0.1 manual JSON'
                'validate:Validate UTCP manual against utcp-specification and CKODEX rules'
                'mcp-config:Generate utcp-mcp bridge configuration'
                'sdk:Generate SDK harness for go-utcp, python-utcp, typescript-utcp, or rs-utcp'
                'ecosystem:List supported UTCP ecosystem repositories'
                '-target:Target UTCP SDK (go-utcp, python-utcp, typescript-utcp, rs-utcp, utcp-mcp)'
                '-out:Output file path'
                '-base-url:Base HTTP URL for HTTP tool call templates'
                '-grpc-host:gRPC host:port for gRPC tool call templates'
                '-manual-url:HTTP URL of UTCP manual'
                '-manual-file:Local file path of UTCP manual'
                '-file:Path to UTCP manual JSON file to validate'
                '-json:Emit output as JSON'
            )
            _describe -t utcp_cmds 'utcp subcommand or flag' utcp_cmds
            ;;
        controls)
            local -a ctrl_cmds
            ctrl_cmds=(
                'search:Search controls matching query'
                'get:Display detailed control card'
                'list:List all controls in framework'
                '-framework:Framework identifier'
                '-family:Filter by control family code'
                '-limit:Maximum search results'
                '-json:Output controls in JSON format'
            )
            _describe -t ctrl_cmds 'controls subcommand or flag' ctrl_cmds
            ;;
        compare)
            local -a cmp_flags
            cmp_flags=(
                '-from:Base framework identifier'
                '-to:Target framework identifier'
                '-markdown:Output Markdown transition report'
                '-json:Output comparison delta in JSON format'
            )
            _describe -t cmp_flags 'compare flag' cmp_flags
            ;;
        generate)
            local -a gen_flags
            gen_flags=(
                '-dsn:SQLite database DSN'
                '-db:Alias for -dsn'
                '-snapshot:Knowledge Graph snapshot name'
                '-framework:Framework identifier'
                '-out:Output directory'
                '-assessment-results-only:Emit only assessment-results.json'
                '-validate:Schema-validate generated artifacts'
                '-validate-constraints:Run oscal-cli Metaschema constraint validation'
                '-oscal-cli:Path to oscal-cli binary'
            )
            _describe -t gen_flags 'generate flag' gen_flags
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
	lines := []string{
		fmt.Sprintf(`complete -c %s -f`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "init" -d "Scaffold OSCAL Component Definition"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "generate" -d "Generate OSCAL 1.2.3 artifacts"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "ingest" -d "Ingest raw regulatory requirements JSON into Knowledge Graph"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "verify" -d "Evaluate Vector State"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "validate" -d "Validate OSCAL JSON schemas"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "diff" -d "GitOps semantic compliance diff"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "derogate" -d "Manage formal risk derogations"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "tabular" -d "Bridge OSCAL and CSV tables"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "bundle-export" -d "Package air-gapped audit bundle"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "frameworks" -d "List, export, compare, verify frameworks"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "verify-parity" -d "Verify verbatim framework parity"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "controls" -d "Search and inspect controls"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "compare" -d "Compare frameworks transition delta"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "ontology" -d "Inspect formal OWL/JSON-LD regulatory & threat ontology"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "graph" -d "Query 36-framework property graph, crosswalks, and blast radius"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "feeds" -d "List, sync, and inspect sovereign & threat feeds"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "serve" -d "Start local Web Studio"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "ui" -d "Open local Web Studio in browser"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "quickstart" -d "Guided onboarding tour"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "doctor" -d "Run deep runtime, schema, and registry diagnostics"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "seed" -d "Seed Knowledge Graph with authoritative baseline frameworks"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "utcp" -d "Universal Tool Calling Protocol (UTCP) manuals & SDKs"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "completion" -d "Generate shell completion"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_use_subcommand" -a "version" -d "Print version"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_seen_subcommand_from ontology" -a "classes predicates spec -json"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_seen_subcommand_from graph" -a "stats crosswalk blast-radius export reconcile -control -target -depth -format -json"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_seen_subcommand_from feeds" -a "list sync inspect verify-checkpoint -feed -offline -checkpoint-dir -json"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_seen_subcommand_from utcp" -a "manual validate mcp-config sdk ecosystem -target -out -base-url -grpc-host -manual-url -manual-file -file -json"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_seen_subcommand_from controls" -a "search get list -framework -family -limit -json"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_seen_subcommand_from compare" -a "-from -to -markdown -json"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_seen_subcommand_from frameworks" -a "list export compare seed verify -jurisdiction -framework -out -json"`, bin),
		fmt.Sprintf(`complete -c %s -n "__fish_seen_subcommand_from generate" -a "-dsn -db -snapshot -framework -out -assessment-results-only -validate -validate-constraints -oscal-cli"`, bin),
	}
	script := strings.Join(lines, "\n") + "\n"

	_, err := io.WriteString(w, script)
	return err
}
