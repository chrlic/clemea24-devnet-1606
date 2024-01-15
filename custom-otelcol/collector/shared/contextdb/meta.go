package contextdb

import (
	"fmt"

	"go.uber.org/zap"
)

type Contexts struct {
	metas  map[string]Context
	logger *zap.Logger
}

type Context struct {
	Source string
	Schema *ContextDbSchema
	Db     *ContextDb
}

func metasFactory(logger *zap.Logger) *Contexts {
	metas := Contexts{}
	metas.init(logger)
	return &metas
}

func (m *Contexts) init(logger *zap.Logger) {
	m.logger = logger
	m.metas = map[string]Context{}
}

func (m *Contexts) registerContext(source string, schema ContextDbSchema) error {
	metaDb := ContextDb{}
	dbSchema, err := GetDbSchema(schema)
	if err != nil {
		return fmt.Errorf("Cannot get DB schema for source %s - %v", source, err)
	}
	metaDb.Init(dbSchema, m.logger)
	meta := Context{
		Source: source,
		Schema: &schema,
		Db:     &metaDb,
	}

	m.metas[source] = meta
	return nil
}
