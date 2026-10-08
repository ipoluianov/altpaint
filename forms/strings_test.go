package forms

import (
	"testing"

	"github.com/ipoluianov/altpaint/paint"
)

// Every language has all the texts: a new field in Strings without a
// translation fails here instead of silently showing English
func TestAllTranslated(t *testing.T) {
	for lang, fields := range catalog.Missing() {
		t.Errorf("%s: not translated: %v", lang, fields)
	}
}

// Every translation is offered in the settings
func TestLanguagesOffered(t *testing.T) {
	offered := make(map[string]bool)
	for _, l := range languages {
		offered[l.tag] = true
	}
	for _, lang := range catalog.Languages() {
		if !offered[lang] {
			t.Errorf("%s: not in the languages list", lang)
		}
	}
}

// The maps of names have every key in every language, and English has
// every tool, filter and parameter
func TestNamesComplete(t *testing.T) {
	for _, td := range toolDefs {
		if en.Tools[td.id] == "" || en.ToolHints[td.id] == "" {
			t.Errorf("en: no name or hint for the tool %s", td.id)
		}
	}
	for _, list := range [][]*paint.Filter{paint.Adjustments, paint.Effects} {
		for _, f := range list {
			if en.Filters[f.ID] == "" {
				t.Errorf("en: no name for the filter %s", f.ID)
			}
			for _, p := range f.Params {
				if en.Params[p.ID] == "" {
					t.Errorf("en: no name for the parameter %s", p.ID)
				}
			}
		}
	}
	for lang, s := range map[string]Strings{"ru": ru} {
		for k := range en.Tools {
			if s.Tools[k] == "" || s.ToolHints[k] == "" {
				t.Errorf("%s: tool %s", lang, k)
			}
		}
		for k := range en.Filters {
			if s.Filters[k] == "" {
				t.Errorf("%s: filter %s", lang, k)
			}
		}
		for k := range en.Params {
			if s.Params[k] == "" {
				t.Errorf("%s: parameter %s", lang, k)
			}
		}
		for k := range en.History {
			if s.History[k] == "" {
				t.Errorf("%s: history step %s", lang, k)
			}
		}
		for i, b := range s.Blends {
			if b == "" {
				t.Errorf("%s: blend mode %d", lang, i)
			}
		}
	}
}
