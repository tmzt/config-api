package config

import (
	"strings"

	"github.com/itchyny/json2yaml"
	"github.com/tmzt/config-api/util"
	"github.com/wI2L/jsondiff"

	"github.com/mrk21/go-diff-fmt/difffmt"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func toYaml(data *util.Data) string {
	logger := util.NewLogger("config.toYaml", 0)

	var out strings.Builder
	in := strings.NewReader(util.ToJson(data))
	if err := json2yaml.Convert(&out, in); err != nil {
		logger.Error("Failed to convert JSON to YAML", err)
		return ""
	}
	return out.String()
}

func unifiedDiff(a string, b string) string {
	logger := util.NewLogger("config.unifiedDiff", 0)

	dmp := diffmatchpatch.New()
	runes1, runes2, lineArray := dmp.DiffLinesToRunes(a, b)
	diffs := dmp.DiffMainRunes(runes1, runes2, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	lineDiffs := difffmt.MakeLineDiffsFromDMP(diffs)
	hunks := difffmt.MakeHunks(lineDiffs, 3)
	unifiedFmt := difffmt.NewUnifiedFormat(difffmt.UnifiedFormatOption{
		ColorMode: difffmt.ColorNone,
	})

	targetA := difffmt.NewDiffTarget("old-config")
	targetB := difffmt.NewDiffTarget("new-config")

	udiff := unifiedFmt.Sprint(targetA, targetB, hunks)

	logger.Printf("\nunifiedDiff: udiff: \n%s\n\n", udiff)

	return udiff
}

func yamlDiff(a *util.Data, b *util.Data) string {
	logger := util.NewLogger("config.yamlDiff", 0)

	ad, bd := toYaml(a), toYaml(b)
	if ad == "" || bd == "" {
		logger.Error("Failed to convert JSON to YAML")
		return ""
	}

	// diffs := dmp.DiffMain(ad, bd, false)
	// diffs = dmp.DiffCleanupSemantic(diffs)

	// return dmp.DiffPrettyText(diffs)

	udiff := unifiedDiff(ad, bd)

	logger.Printf("\nyamlDiff: udiff: \n%s\n\n", udiff)

	return udiff
}

func AnnotateHistory(entries *[]*ConfigDiffVersionHistoryEntry) {
	logger := util.NewLogger("config.AnnotateHistory", 0)
	if entries == nil {
		logger.Error("Entries is nil")
		return
	}

	maxIndex := len(*entries) - 1

	for i, entry := range *entries {
		startingValues := &util.Data{}

		if i < maxIndex && (*entries)[i+1] != nil && (*entries)[i+1].RecordContents != nil {
			startingValues = (*entries)[i+1].RecordContents
		}

		newContents := entry.RecordContents

		logger.Printf("\nAnnotateHistory: startingValues: %+v\n", startingValues)
		logger.Printf("\nAnnotateHistory: newContents: %+v\n", newContents)

		diff, err := jsondiff.Compare(startingValues, entry.RecordContents)
		if err != nil {
			logger.Error("Failed to compare JSON objects", err)
			continue
		}
		entry.RecordContentsPatch = &diff
		entry.RecordContentsTextPatch = yamlDiff(startingValues, newContents)
	}
}
