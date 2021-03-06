package edit

import (
	"fmt"
	"io/ioutil"

	"github.com/xiaq/elvish/parse"
)

type tokenPart struct {
	text      string
	completed bool
}

type candidate struct {
	text  string
	parts []tokenPart
	attr  string // Attribute used for preview
}

func newCandidate() *candidate {
	return &candidate{}
}

func (c *candidate) push(tp tokenPart) {
	c.text += tp.text
	c.parts = append(c.parts, tp)
}

type completion struct {
	start, end int // The text to complete is Editor.line[start:end]
	typ        parse.ItemType
	candidates []*candidate
	current    int
}

func (c *completion) prev(cycle bool) {
	c.current--
	if c.current == -1 {
		if cycle {
			c.current = len(c.candidates) - 1
		} else {
			c.current++
		}
	}
}

func (c *completion) next(cycle bool) {
	c.current++
	if c.current == len(c.candidates) {
		if cycle {
			c.current = 0
		} else {
			c.current--
		}
	}
}

func findCandidates(p string, all []string) (cands []*candidate) {
	// Prefix match
	for _, s := range all {
		if len(s) >= len(p) && s[:len(p)] == p {
			cand := newCandidate()
			cand.push(tokenPart{p, false})
			cand.push(tokenPart{s[len(p):], true})
			cands = append(cands, cand)
		}
	}
	return
}

func fileNames(dir string) (names []string, err error) {
	infos, e := ioutil.ReadDir(".")
	if e != nil {
		err = e
		return
	}
	for _, info := range infos {
		names = append(names, info.Name())
	}
	return
}

var (
	notPlainFactor     = fmt.Errorf("not a plain FactorNode")
	notPlainTerm       = fmt.Errorf("not a plain TermNode")
	unknownContextType = fmt.Errorf("unknown context type")
)

func peekFactor(fn *parse.FactorNode) (string, error) {
	if fn.Typ != parse.StringFactor {
		return "", notPlainFactor
	}
	return fn.Node.(*parse.StringNode).Text, nil
}

func peekIncompleteTerm(tn *parse.TermNode) (string, int, error) {
	text := ""
	for _, n := range tn.Nodes {
		s, e := peekFactor(n)
		if e != nil {
			return "", 0, notPlainTerm
		}
		text += s
	}
	return text, int(tn.Pos), nil
}

func peekCurrentTerm(ctx *parse.Context, dot int) (string, int, error) {
	if ctx.Form == nil || ctx.Typ == parse.NewArgContext {
		return "", dot, nil
	}

	switch ctx.Typ {
	case parse.ArgContext:
		terms := ctx.Form.Args.Nodes
		lastTerm := terms[len(terms)-1]
		return peekIncompleteTerm(lastTerm)
	case parse.RedirFilenameContext:
		redirs := ctx.Form.Redirs
		lastRedir := redirs[len(redirs)-1]
		fnRedir, ok := lastRedir.(*parse.FilenameRedir)
		if !ok {
			return "", 0, fmt.Errorf("last redir is not FilenameRedir")
		}
		return peekIncompleteTerm(fnRedir.Filename)
	default:
		return "", 0, unknownContextType
	}
}

func startCompletion(ed *Editor, k Key) *leReturn {
	c := &completion{}
	ctx, err := parse.Complete("<completion>", ed.line[:ed.dot])
	if err != nil {
		ed.pushTip("parser error")
		return nil
	}
	term, start, err := peekCurrentTerm(ctx, ed.dot)
	if err != nil {
		ed.pushTip("cannot complete :(")
		return nil
	}
	switch ctx.Typ {
	case parse.CommandContext:
		// BUG(xiaq): When completing, CommandContext is not supported
		ed.pushTip("command context not yet supported :(")
	case parse.NewArgContext, parse.ArgContext:
		// BUG(xiaq): When completing, [New]ArgContext is treated like RedirFilenameContext
		fallthrough
	case parse.RedirFilenameContext:
		// BUG(xiaq): When completing, only the case of ctx.ThisFactor.Typ == StringFactor is supported
		names, err := fileNames(".")
		if err != nil {
			ed.pushTip(err.Error())
			return nil
		}
		c.start = start
		c.end = ed.dot
		// BUG(xiaq) When completing, completion.typ is always ItemBare
		c.typ = parse.ItemBare
		c.candidates = findCandidates(term, names)
		if len(c.candidates) > 0 {
			// XXX assumes filename candidate
			for _, c := range c.candidates {
				c.attr = defaultLsColor.determineAttr(c.text)
			}
			ed.completion = c
			ed.mode = modeCompletion
		} else {
			ed.pushTip(fmt.Sprintf("No completion for %s", term))
		}
	}
	return nil
}
