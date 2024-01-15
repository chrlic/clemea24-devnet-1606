package ciscointersight

import (
	"context"
	"fmt"
	"os"

	contextdb "github.com/chrlic/otelcol-cust/collector/shared/contextdb"
	"github.com/chrlic/otelcol-cust/collector/shared/jsonscraper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type intersightReceiver struct {
	metricConsumer   consumer.Metrics
	logConsumer      consumer.Logs
	logger           *zap.Logger
	cancel           context.CancelFunc
	config           component.Config
	ctx              context.Context
	receiverID       string
	isMetricReceiver bool
	isLogReceiver    bool
	contextDb        contextdb.ContextDb
}

func (r *intersightReceiver) Start(ctx context.Context, host component.Host) error {

	r.ctx, r.cancel = context.WithCancel(context.Background())

	cfg := r.config.(*Config)

	err := r.initContextDb()
	if err != nil {
		r.logger.Sugar().Errorf("Cannot initialize Context DB - %v", err)
		return err
	}

	r.subscribeToExtensions(host, cfg)

	intersightClient, err := getIntersightSDKClient(cfg, r.logger)
	if err != nil {
		r.logger.Sugar().Errorf("Cannot initialize Intersight SDK client - %v", err)
		return err
	}
	emitter := jsonscraper.NewEmitter(ctx, r.logger, r.metricConsumer, r.logConsumer)
	scraper := jsonscraper.NewScraper(r.receiverID, r.logger, intersightClient, emitter, cfg.ScraperConfig, cfg.Interval, &r.contextDb)
	scraper.Run()

	return nil
}

func (r *intersightReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func (r *intersightReceiver) initContextDb() error {
	config := r.config.(*Config)
	ctxDb := contextdb.ContextDb{}

	dbJsonSchemas := []*contextdb.ContextTableSchema{}

	for _, schema := range config.DbSchemas {
		schemaConfig, err := os.ReadFile(schema)
		if err != nil {
			return fmt.Errorf("cannot read db schema yaml %s - %v", schema, err)
		}

		dbJsonSchema, err := contextdb.ParseDbJsonSchema(schemaConfig)
		if err != nil {
			return fmt.Errorf("cannot parse db schema file %s - %v", schema, err)
		}

		dbJsonSchemas = contextdb.AppendDbJsonSchema(dbJsonSchemas, dbJsonSchema)
	}

	if len(dbJsonSchemas) > 0 {
		dbSchema, err := contextdb.GetDbSchema(dbJsonSchemas)
		if err != nil {
			return fmt.Errorf("cannot convert schema to memdb schema %v - %v", dbSchema, err)
		}
		err = ctxDb.Init(dbSchema, r.logger)
		if err != nil {
			return fmt.Errorf("cannot init DB %v - %v", dbSchema, err)
		}

		r.contextDb = ctxDb
	} else {
		r.logger.Sugar().Info("Context DB node not defined")
	}

	return nil
}

func (r *intersightReceiver) subscribeToExtensions(host component.Host, config *Config) {
	extensions := host.GetExtensions()
	for _, ctxProvider := range config.ContextProviders {
		var compID component.ID
		compID.UnmarshalText([]byte(ctxProvider.Name))
		extension, ok := extensions[compID]
		if !ok {
			r.logger.Sugar().Errorf("Failed to find Extension %s", ctxProvider.Name)
			continue
		}
		r.logger.Sugar().Infof("Extension found %s - %T", ctxProvider.Name, extension)
		extInstance, ok := interface{}(extension).(contextdb.ContextProviderExtension)

		if ok {
			for _, subscr := range ctxProvider.Subscriptions {
				subContext, err := extInstance.SubscribeToContext(r.receiverID, subscr.Topic)
				if err != nil {
					r.logger.Sugar().Errorf("Failed to subscribe to Extension %s - %v", ctxProvider.Name, err)
					continue
				}
				r.logger.Sugar().Infof("Subscribed to Extension %s", ctxProvider.Name)
				if r.contextDb.Db != nil {
					subContext.AttachContextDb(r.contextDb, subscr.Topic)
				} else {
					r.logger.Sugar().Warn("Subscribed for data from extension but no context DB schema defined")
				}
			}
		} else {
			r.logger.Sugar().Errorf("Extension %s does not implement ContextProviderExtension interface", ctxProvider.Name)
			continue
		}
	}
}
