package deepstudent

import (
	"encoding/json"

	"github.com/helixnow/deep-student-go/internal/research"
	"github.com/helixnow/deep-student-go/internal/translate"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func anyTypes(in []string) []vfs.ResourceType {
	out := make([]vfs.ResourceType, len(in))
	for i, s := range in {
		out[i] = vfs.ResourceType(s)
	}
	return out
}

func anyEngines(in []string) []research.Engine {
	out := make([]research.Engine, len(in))
	for i, s := range in {
		out[i] = research.Engine(s)
	}
	return out
}

func jsonRaw(s string) json.RawMessage { return json.RawMessage(s) }

func collectStream(ch <-chan string) string {
	out := ""
	for s := range ch {
		out += s
	}
	return out
}

func toTranslateRequest(text, src, tgt, domain, custom string, glossary []map[string]string) translate.Request {
	out := translate.Request{Text: text, Source: src, Target: tgt, Domain: translate.Domain(domain), CustomPrompt: custom}
	for _, g := range glossary {
		out.Glossary = append(out.Glossary, translate.GlossaryEntry{Source: g["source"], Target: g["target"]})
	}
	return out
}
