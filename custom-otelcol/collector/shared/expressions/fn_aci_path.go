package expressions

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

var pathSplitFunction = cel.Function("acipath",
	cel.Overload("acipath_string",
		[]*cel.Type{cel.StringType},
		cel.MapType(cel.StringType, cel.StringType),
		pathFunctionImpl,
	),
)

var pathSplitMemberFunction = cel.Function("acipath",
	cel.MemberOverload("string_acipath",
		[]*cel.Type{cel.StringType},
		cel.MapType(cel.StringType, cel.StringType),
		pathFunctionImpl,
	),
)

var pathNodesFunction = cel.Function("acipathnodes",
	cel.Overload("acipathnodes_arr_of_string",
		[]*cel.Type{cel.ListType(cel.StringType)},
		cel.ListType(cel.StringType),
		pathNodesImpl,
	),
)

var pathNodesMemberFunction = cel.Function("acipathnodes",
	cel.MemberOverload("arr_of_string_acipathnodes",
		[]*cel.Type{cel.ListType(cel.StringType)},
		cel.ListType(cel.StringType),
		pathNodesImpl,
	),
)

var pathParseFunction = cel.Function("acipathparse",
	cel.Overload("acipathparse_arr_of_string",
		[]*cel.Type{cel.ListType(cel.StringType)},
		cel.DynType,
		pathsParseImpl,
	),
)

var pathParseMemberFunction = cel.Function("acipathparse",
	cel.MemberOverload("arr_of_string_acipathparse",
		[]*cel.Type{cel.ListType(cel.StringType)},
		cel.DynType,
		pathsParseImpl,
	),
)

var pathFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	ss := args[0].Value().(string)
	parts := strings.Split(ss, "-[")
	if len(parts) != 2 {
		ret := map[string]string{"fail": "1"}
		return types.NewStringStringMap(StringAdapter{}, ret)
	}
	target := strings.Split(parts[1], "]")
	pathParts := strings.Split(parts[0], "/")
	if len(pathParts) < 2 {
		ret := map[string]string{"fail": "2"}
		return types.NewStringStringMap(StringAdapter{}, ret)
	}
	ret := map[string]string{
		"path":   parts[0],
		"target": target[0],
		"pod":    pathParts[1],
		"podId":  strings.Split(pathParts[1], "-")[1],
		"node":   pathParts[2],
		"nodeId": strings.Join(strings.Split(pathParts[2], "-")[1:], "-"),
	}
	// fmt.Printf("Got here: %v\n", ret)
	return types.NewStringStringMap(StringAdapter{}, ret)
})

var pathNodesImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	result := []string{}
	resultMap := map[string]string{}
	pod := ""
	paths, ok := args[0].(traits.Lister)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - should be list of strings", args[0].Type())
	}

	iter := paths.Iterator()
	for iter.HasNext().Value().(bool) {
		i := iter.Next()
		path := i.Value().(string)
		elems := strings.Split(path, "/")
		if len(elems) < 3 {
			continue
		}
		pod = elems[1]
		nodes := elems[2]
		nodeElems := strings.Split(nodes, "-")
		if len(nodeElems) == 0 {
			continue
		}
		switch nodeElems[0] {
		case "paths": // Single node path
			resultMap[nodeElems[1]] = nodeElems[1]
		case "protpaths": // Port Channel or vPC path
			resultMap[nodeElems[1]] = nodeElems[1]
			resultMap[nodeElems[2]] = nodeElems[2]
		case "pathgrp": // VMM learned -> VMM manager like vCenter, ignore
		default: // should not get here, but ignore
		}
	}

	for key := range resultMap {
		result = append(result, fmt.Sprintf("topology/%s/node-%s", pod, key))
	}

	return types.NewStringList(StringAdapter{}, result)
})

var pathsParseImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	nodes := []string{}
	nodeMap := map[string]string{}
	logIfs := []string{}
	logIfMap := map[string]string{}
	phyIfs := []string{}
	phyIfMap := map[string]string{}
	pod := ""
	paths, ok := args[0].(traits.Lister)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - should be list of strings", args[0].Type())
	}

	iter := paths.Iterator()
	for iter.HasNext().Value().(bool) {
		i := iter.Next()
		path := i.Value().(string)
		elems := strings.Split(path, "/")
		if len(elems) < 3 {
			continue
		}
		pod = elems[1]
		nodes := elems[2]
		nodeElems := strings.Split(nodes, "-")
		if len(nodeElems) == 0 {
			continue
		}
		switch nodeElems[0] {
		case "paths": // Single node path
			nodeMap[nodeElems[1]] = nodeElems[1]
			phyIfMap[path] = path
		case "protpaths": // Port Channel or vPC path
			nodeMap[nodeElems[1]] = nodeElems[1]
			nodeMap[nodeElems[2]] = nodeElems[2]
			logIfMap[path] = path
		case "pathgrp": // VMM learned -> VMM manager like vCenter, ignore
		default: // should not get here, but ignore
		}
	}

	for key := range nodeMap {
		nodes = append(nodes, fmt.Sprintf("topology/%s/node-%s", pod, key))
	}

	for key := range phyIfMap {
		phyIfs = append(phyIfs, key)
	}

	for key := range logIfMap {
		logIfs = append(logIfs, key)
	}

	result := map[string]any{
		"nodes":  nodes,
		"phyIfs": phyIfs,
		"logIfs": logIfs,
	}

	return types.NewStringInterfaceMap(StringListAdapter{}, result)
})

type StringListAdapter struct {
	ref.TypeAdapter
}

func (s StringListAdapter) NativeToValue(in any) ref.Val {

	input, ok := in.([]string)
	if !ok {
		return types.ValOrErr(types.NullType, "unsupported input type, has to be []string")
	}

	result := types.NewStringList(StringAdapter{}, input)
	return result
}
