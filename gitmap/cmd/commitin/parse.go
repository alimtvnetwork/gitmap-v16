package commitin

import (
	"flag"
	"io"
	"strings"

	"github.com/alimtvnetwork/gitmap-v16/gitmap/constants"
)

// Parse converts an argv slice (already stripped of the leading
// `commit-in` / `cin` token) into a fully-validated RawArgs.
//
// Pure: zero git, zero filesystem, zero DB. Caller layers profile
// load + interactive prompts on top of the returned RawArgs per
// spec §5.6.
//
// Error contract: every failure returns a *ParseError carrying the
// exit code from constants.CommitInExit*. Caller writes
// `parseErr.Message` to STDERR and exits with `parseErr.ExitCode` —
// no other side effects happen here.
func Parse(args []string) (*RawArgs, *ParseError) {
	fs, raw, csv := newFlagSet()
	if err := fs.Parse(reorder(args)); err != nil {
		return nil, newBadArgs("%v", err)
	}
	if perr := finalizeFlagFanout(raw, csv); perr != nil {
		return nil, perr
	}
	if perr := splitPositional(raw, fs.Args()); perr != nil {
		return nil, perr
	}
	if perr := validateAll(raw); perr != nil {
		return nil, perr
	}
	return raw, nil
}

// csvHolder collects every CSV-shaped flag so finalizeFlagFanout can
// run the comma split + per-flag validation in one pass. Keeps the
// flag-registration func short.
type csvHolder struct {
	exclude          string
	messageExclude   string
	messagePrefix    string
	messageSuffix    string
	overrideMessages string
	weakWords        string
	languages        string
}

// newFlagSet registers every spec §2.5 flag in one place. Returns the
// flag set, the destination RawArgs, and the CSV holder for the
// post-parse fan-out step. Help output is silenced — Parse is meant
// to be called by an outer command that owns the help dispatch.
func newFlagSet() (*flag.FlagSet, *RawArgs, *csvHolder) {
	fs := flag.NewFlagSet(constants.CmdCommitIn, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	raw := &RawArgs{}
	csv := &csvHolder{}
	registerBoolFlags(fs, raw)
	registerStringFlags(fs, raw)
	registerCsvFlags(fs, csv)
	return fs, raw, csv
}

// registerBoolFlags wires every bool-typed spec §2.5 flag.
func registerBoolFlags(fs *flag.FlagSet, raw *RawArgs) {
	fs.BoolVar(&raw.UseDefaultProfile, constants.CommitInFlagDefault, false, constants.CommitInDescDefault)
	fs.BoolVar(&raw.UseDefaultProfile, constants.CommitInFlagDefaultShort, false, constants.CommitInDescDefault)
	fs.BoolVar(&raw.SaveProfileOverwrite, constants.CommitInFlagSaveProfileOverwrite, false, constants.CommitInDescSaveProfileOverwrite)
	fs.BoolVar(&raw.SetDefault, constants.CommitInFlagSetDefault, false, constants.CommitInDescSetDefault)
	fs.BoolVar(&raw.OverrideOnlyWeak, constants.CommitInFlagOverrideOnlyWeak, false, constants.CommitInDescOverrideOnlyWeak)
	fs.BoolVar(&raw.IsNoPrompt, constants.CommitInFlagNoPrompt, false, constants.CommitInDescNoPrompt)
	fs.BoolVar(&raw.IsDryRun, constants.CommitInFlagDryRun, false, constants.CommitInDescDryRun)
	fs.BoolVar(&raw.IsKeepTemp, constants.CommitInFlagKeepTemp, false, constants.CommitInDescKeepTemp)
}

// registerStringFlags wires every string-typed spec §2.5 flag.
func registerStringFlags(fs *flag.FlagSet, raw *RawArgs) {
	fs.StringVar(&raw.ProfileName, constants.CommitInFlagProfile, "", constants.CommitInDescProfile)
	fs.StringVar(&raw.SaveProfileName, constants.CommitInFlagSaveProfile, "", constants.CommitInDescSaveProfile)
	fs.StringVar(&raw.AuthorName, constants.CommitInFlagAuthorName, "", constants.CommitInDescAuthorName)
	fs.StringVar(&raw.AuthorEmail, constants.CommitInFlagAuthorEmail, "", constants.CommitInDescAuthorEmail)
	fs.StringVar(&raw.ConflictMode, constants.CommitInFlagConflict, "", constants.CommitInDescConflict)
	fs.StringVar(&raw.TitlePrefix, constants.CommitInFlagTitlePrefix, "", constants.CommitInDescTitlePrefix)
	fs.StringVar(&raw.TitleSuffix, constants.CommitInFlagTitleSuffix, "", constants.CommitInDescTitleSuffix)
	fs.StringVar(&raw.FunctionIntel, constants.CommitInFlagFunctionIntel, "", constants.CommitInDescFunctionIntel)
}

// registerCsvFlags wires every CSV-shaped spec §2.5 flag into the
// holder so the parser can split + validate uniformly.
func registerCsvFlags(fs *flag.FlagSet, csv *csvHolder) {
	fs.StringVar(&csv.exclude, constants.CommitInFlagExclude, "", constants.CommitInDescExclude)
	fs.StringVar(&csv.messageExclude, constants.CommitInFlagMessageExclude, "", constants.CommitInDescMessageExclude)
	fs.StringVar(&csv.messagePrefix, constants.CommitInFlagMessagePrefix, "", constants.CommitInDescMessagePrefix)
	fs.StringVar(&csv.messageSuffix, constants.CommitInFlagMessageSuffix, "", constants.CommitInDescMessageSuffix)
	fs.StringVar(&csv.overrideMessages, constants.CommitInFlagOverrideMessages, "", constants.CommitInDescOverrideMessages)
	fs.StringVar(&csv.weakWords, constants.CommitInFlagWeakWords, "", constants.CommitInDescWeakWords)
	fs.StringVar(&csv.languages, constants.CommitInFlagLanguages, "", constants.CommitInDescLanguages)
}

// finalizeFlagFanout splits every CSV holder into its typed slice and
// runs the message-rule shape validator. Other validators run later
// once positional args are bound.
func finalizeFlagFanout(raw *RawArgs, csv *csvHolder) *ParseError {
	raw.Exclude = splitCSV(csv.exclude)
	raw.MessagePrefix = splitCSV(csv.messagePrefix)
	raw.MessageSuffix = splitCSV(csv.messageSuffix)
	raw.OverrideMessages = splitCSV(csv.overrideMessages)
	raw.WeakWords = splitCSV(csv.weakWords)
	raw.Languages = splitCSV(csv.languages)
	rules, perr := parseMessageRules(splitCSV(csv.messageExclude))
	if perr != nil {
		return perr
	}
	raw.MessageRules = rules
	return nil
}

// splitPositional consumes the leftover argv: first token is <source>,
// remainder is the input list. KEYWORD detection runs on a SINGLE
// remainder token; multi-token remainders go straight to splitInputs.
func splitPositional(raw *RawArgs, positional []string) *ParseError {
	if len(positional) == 0 {
		return newBadArgs("%s", "missing <source>")
	}
	raw.Source = positional[0]
	rest := positional[1:]
	if len(rest) == 1 {
		kw, tail, isKw, perr := classifyKeyword(rest[0])
		if perr != nil {
			return perr
		}
		if isKw {
			raw.Keyword = kw
			raw.KeywordTail = tail
			return nil
		}
	}
	raw.Inputs = splitInputs(rest)
	return nil
}

// validateAll runs the cross-cutting validators once positional args
// are bound. Order chosen so the most informative error wins when
// several would fire.
func validateAll(raw *RawArgs) *ParseError {
	if perr := requireSourceAndInputs(raw.Source, raw.Inputs, raw.Keyword); perr != nil {
		return perr
	}
	if perr := rejectMixedKeyword(raw.Keyword, raw.Inputs); perr != nil {
		return perr
	}
	if perr := validateAuthorPair(raw.AuthorName, raw.AuthorEmail); perr != nil {
		return perr
	}
	if perr := validateConflictMode(raw.ConflictMode); perr != nil {
		return perr
	}
	if perr := validateFunctionIntelToggle(raw.FunctionIntel); perr != nil {
		return perr
	}
	return validateLanguages(raw.Languages)
}

// reorder lifts every flag token to the front so Go's stdlib `flag`
// package can parse positional args that follow flags. Mirrors the
// existing project pattern documented in mem://tech/flag-parsing-logic.
//
// Tokens beginning with `-` are flags; the next token is treated as
// the flag's value when the flag form is `--name value` (i.e. no `=`
// in the flag token AND the flag is not in the bool set).
func reorder(args []string) []string {
	flags, positional := splitFlagsAndPositional(args)
	out := make([]string, 0, len(args))
	out = append(out, flags...)
	out = append(out, positional...)
	return out
}

// splitFlagsAndPositional walks argv once, classifying each token.
// Bool flags consume zero values; other flags consume one value when
// the form is `--name value`.
func splitFlagsAndPositional(args []string) ([]string, []string) {
	bools := boolFlagSet()
	flags := make([]string, 0, len(args))
	positional := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		tok := args[i]
		if !strings.HasPrefix(tok, "-") || tok == "-" {
			positional = append(positional, tok)
			continue
		}
		flags = append(flags, tok)
		if needsValue(tok, bools) && i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return flags, positional
}

// needsValue reports whether the next argv token belongs to this flag.
// `--name=value` forms embed the value, so they need no companion. A
// flag in the bool set never consumes a value.
func needsValue(tok string, bools map[string]bool) bool {
	if strings.Contains(tok, "=") {
		return false
	}
	name := strings.TrimLeft(tok, "-")
	return !bools[name]
}

// boolFlagSet enumerates every flag registered as bool above. Kept
// adjacent to registerBoolFlags so adding a bool flag can never desync.
func boolFlagSet() map[string]bool {
	return map[string]bool{
		constants.CommitInFlagDefault:              true,
		constants.CommitInFlagDefaultShort:         true,
		constants.CommitInFlagSaveProfileOverwrite: true,
		constants.CommitInFlagSetDefault:           true,
		constants.CommitInFlagOverrideOnlyWeak:     true,
		constants.CommitInFlagNoPrompt:             true,
		constants.CommitInFlagDryRun:               true,
		constants.CommitInFlagKeepTemp:             true,
	}
}
