Source files:

* `client.go`
* `configuration.go`
* `signing.go`

come from https://github.com/ciscodevnet/intersight-go. The project is not using full SDK via import, since, due to its' size, it takes ages to compile.

The file `added_api.go` contains functions which are needed by the project, but are not exposed by the https://github.com/ciscodevnet/intersight-go.


