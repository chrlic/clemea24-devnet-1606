package expressions

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/vjeantet/grok"
)

var stringGrokFunction = cel.Function("grok",
	cel.Overload("grok_string_string",
		[]*cel.Type{cel.StringType, cel.StringType},
		cel.MapType(cel.StringType, cel.StringType),
		grokFunctionImpl,
	),
)

var stringGrokMemberFunction = cel.Function("grok",
	cel.MemberOverload("string_grok_string",
		[]*cel.Type{cel.StringType, cel.StringType},
		cel.MapType(cel.StringType, cel.StringType),
		grokFunctionImpl,
	),
)

var grokFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	text := args[0].Value().(string)
	pattern := args[1].Value().(string)
	g, _ := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	grokPredefinePatterns(g)
	values, err := g.Parse(pattern, text)
	if err != nil {
		return types.NewErr("Error parsing >%s< by Grok pattern >%s<", text, pattern)
	}

	return types.NewStringStringMap(StringAdapter{}, values)
})

func grokPredefinePatterns(g *grok.Grok) {
	g.AddPattern("ACIBUNDLE", `uni/infra/funcprof/accbundle-%{GREEDYDATA:bundle}`)
	g.AddPattern("ACI_EP_LOGIF", `topology/%{DATA:pod}/protpaths-%{DATA:node1}-%{DATA:node2}/pathep-\[%{DATA:bundle}\]`)
	g.AddPattern("ACI_EP_PHYIF", `topology/%{DATA:pod}/paths-%{DATA:node}/pathep-\[%{DATA:if}\]`)
	g.AddPattern("ACIEP", `uni/tn-%{DATA:tenant}/ap-%{DATA:applicationPolicy}/epg-%{DATA:epg}/cep-%{GREEDYDATA:mac}`)
	g.AddPattern("ACIPHYIF", `topology/%{DATA:pod}/%{DATA:node}/sys/phys-\[%{DATA:if}\]`)
}
