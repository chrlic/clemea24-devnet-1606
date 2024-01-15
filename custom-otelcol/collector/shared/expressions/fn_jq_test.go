package expressions

import (
	"strings"
	"testing"

	"github.com/antchfx/jsonquery"
)

func TestJqFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	jsonqueryDoc, err := jsonquery.Parse(strings.NewReader(jsonDoc))
	if err != nil {
		t.Fatalf("Cannot parse test json doc - %v", err)
	}
	env.JqSetDoc(jsonqueryDoc)
	args := map[string]interface{}{
		"expr1": "/imdata//fvTenant/attributes/name",
	}
	ret, err := env.EvaluateExpression(`jqs("imdata//fvTenant/attributes/name")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	val, ok := (*ret).Value().(string)
	if !ok {
		t.Fatalf("jqf returned invalid type of %T", (*ret).Value())
	}
	expect := "aaa_600_aci_a"
	if expect != val {
		t.Fatalf("expected: %v != actual: %v", expect, val)
	}
}

var jsonDoc = `
{
    "totalCount": "17",
    "imdata": [
        {
            "fvTenant": {
                "attributes": {
                    "annotation": "orchestrator:aci-containers-controller",
                    "childAction": "",
                    "descr": "",
                    "dn": "uni/tn-aaa_600_aci_a",
                    "extMngdBy": "",
                    "lcOwn": "local",
                    "modTs": "2020-03-23T15:11:32.840+02:00",
                    "monPolDn": "uni/tn-common/monepg-default",
                    "name": "aaa_600_aci_a",
                    "nameAlias": "",
                    "ownerKey": "",
                    "ownerTag": "",
                    "status": "",
                    "uid": "15374",
                    "userdom": "all"
                },
                "children": [
                    {
                        "healthInst": {
                            "attributes": {
                                "childAction": "",
                                "chng": "0",
                                "cur": "100",
                                "maxSev": "cleared",
                                "modTs": "never",
                                "prev": "100",
                                "rn": "health",
                                "status": "",
                                "twScore": "100",
                                "updTs": "2023-04-08T10:55:02.268+02:00"
                            }
                        }
                    }
                ]
            }
        },
        {
            "fvTenant": {
                "attributes": {
                    "annotation": "",
                    "childAction": "",
                    "descr": "",
                    "dn": "uni/tn-mgmt",
                    "extMngdBy": "",
                    "lcOwn": "local",
                    "modTs": "2017-11-07T06:17:01.685+02:00",
                    "monPolDn": "uni/tn-common/monepg-default",
                    "name": "mgmt",
                    "nameAlias": "",
                    "ownerKey": "",
                    "ownerTag": "",
                    "status": "",
                    "uid": "0",
                    "userdom": "all"
                },
                "children": [
                    {
                        "healthInst": {
                            "attributes": {
                                "childAction": "",
                                "chng": "2",
                                "cur": "100",
                                "maxSev": "cleared",
                                "modTs": "never",
                                "prev": "98",
                                "rn": "health",
                                "status": "",
                                "twScore": "100",
                                "updTs": "2023-03-28T02:17:08.771+02:00"
                            }
                        }
                    }
                ]
            }
        }
    ]
}
`
