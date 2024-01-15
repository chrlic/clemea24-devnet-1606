package jsonscraper

import (
	"fmt"

	"github.com/antchfx/jsonquery"
)

type scraperContext struct {
	docStack       Stack[*jsonquery.Node]
	rsrcAttrsStack Stack[map[string]any]
	itemAttrsStack Stack[map[string]any]
	scopeStack     Stack[*Scope]
	paramStack     Stack[map[string]any]
}

func newScaperContext() scraperContext {
	return scraperContext{
		docStack:       *NewStack[*jsonquery.Node](),
		rsrcAttrsStack: *NewStack[map[string]any](),
		itemAttrsStack: *NewStack[map[string]any](),
		scopeStack:     *NewStack[*Scope](),
		paramStack:     *NewStack[map[string]any](),
	}
}

func (ctx *scraperContext) cleanup() {

}

func (ctx *scraperContext) push() {
	ctx.docStack.Push(nil)
	ctx.scopeStack.Push(nil)
	ctx.rsrcAttrsStack.Push(map[string]any{})
	ctx.itemAttrsStack.Push(map[string]any{})
	ctx.paramStack.Push(map[string]any{})
}

func (ctx *scraperContext) pop() {
	ctx.docStack.Pop()
	ctx.scopeStack.Pop()
	ctx.rsrcAttrsStack.Pop()
	ctx.itemAttrsStack.Pop()
	ctx.paramStack.Pop()
}

func (ctx *scraperContext) setDoc(doc *jsonquery.Node) {
	ctx.docStack.SetTop(doc)
}

func (ctx *scraperContext) setScope(scope *Scope) {
	ctx.scopeStack.SetTop(scope)
}

func (ctx *scraperContext) addRsrcAttr(name string, value any) bool {
	rsrcMap, exists := ctx.rsrcAttrsStack.Top()
	if !exists {
		return false
	}
	rsrcMap[name] = value
	return true
}

func (ctx *scraperContext) addItemAttr(name string, value any) bool {
	itemMap, exists := ctx.itemAttrsStack.Top()
	if !exists {
		return false
	}
	itemMap[name] = value
	return true
}

func (ctx *scraperContext) addParameter(name string, value string) bool {
	paramMap, exists := ctx.paramStack.Top()
	if !exists {
		return false
	}
	paramMap[name] = value
	return true
}

func (ctx *scraperContext) initMapReducer() map[string]any {
	return map[string]any{}
}

func (ctx *scraperContext) runMapReducer(accum map[string]any, added map[string]any) map[string]any {
	new := map[string]any{}
	for key, val := range accum {
		new[key] = val
	}
	for key, val := range added {
		new[key] = val
	}
	return new
}

func (ctx *scraperContext) getRsrcAttrs() map[string]any {
	return ctx.rsrcAttrsStack.Reduce(
		ctx.initMapReducer,
		ctx.runMapReducer,
	)
}

func (ctx *scraperContext) getItemAttrs() map[string]any {
	return ctx.itemAttrsStack.Reduce(
		ctx.initMapReducer,
		ctx.runMapReducer,
	)
}

func (ctx *scraperContext) getParameters() map[string]any {
	return ctx.paramStack.Reduce(
		ctx.initMapReducer,
		ctx.runMapReducer,
	)
}

func (ctx *scraperContext) getScope() *Scope {
	return ctx.scopeStack.Reduce(
		func() *Scope {
			return &Scope{}
		},
		func(accum *Scope, added *Scope) *Scope {
			new := &Scope{}
			if accum != nil {
				new.Name = accum.Name
				new.Version = accum.Version
			}
			if added != nil {
				new.Name = added.Name
				new.Version = added.Version
			}
			return new
		},
	)
}

func (ctx *scraperContext) String() string {
	result := ""

	result += fmt.Sprintf("Doc Stack: %v\n", ctx.docStack)
	result += fmt.Sprintf("Scope Stack: %v\n", ctx.scopeStack)
	result += fmt.Sprintf("Rsrc Stack: %v\n", ctx.rsrcAttrsStack)
	result += fmt.Sprintf("Item Stack: %v\n", ctx.itemAttrsStack)
	result += fmt.Sprintf("Param Stack: %v\n", ctx.paramStack)

	return result
}
