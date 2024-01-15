package contextdb // import github.com/chrlic/otelcol-cust/collector/receiver/shared/db

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/antchfx/jsonquery"
	xj "github.com/basgys/goxml2json"
	"github.com/hashicorp/go-memdb"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

//=====================================================================================
// SingleValueFieldIndexer - indexes based on field references, returns one value per
// object
//=====================================================================================

type SingleValueFieldIndexer struct {
	Fields []string
}

func (s *SingleValueFieldIndexer) FromObject(raw interface{}) (bool, []byte, error) {

	obj, ok := raw.(ContextRecord)
	if !ok {
		return false, nil, fmt.Errorf("Invalid type of object %T in SingleValueFieldIndexer, %v", raw, raw)
	}
	val := ""
	for _, f := range s.Fields {
		field := jsonquery.FindOne(obj.Data, f)
		if field != nil {
			val += "\x01" + fmt.Sprintf("%s", field.Value())
		}
	}

	if val == "" {
		return false, nil, nil
	}

	// Add the null character as a terminator
	val += "\x00"
	return true, []byte(val), nil
}

// FromArgs takes in a slice of args and returns its byte form.
func (s *SingleValueFieldIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != len(s.Fields) {
		return nil, fmt.Errorf("wrong number of args %d, expected %d", len(args), len(s.Fields))
	}
	val := ""
	for _, arg := range args {
		s := ""
		switch reflect.TypeOf(arg).String() {
		case "int":
			s = fmt.Sprintf("%d", arg)
		case "bool":
			s = fmt.Sprintf("%t", arg)
		case "string":
			s = arg.(string)
		default:
			s = fmt.Sprintf("%v", arg)
		}
		val += "\x01" + s
	}
	// Add the null character as a terminator
	val += "\x00"
	return []byte(val), nil
}

//=====================================================================================
// MultiValueFieldIndexer - indexes based on field references, returns one or more values per
// object
//=====================================================================================

type MultiValueFieldIndexer struct {
	Fields []string
}

func (s *MultiValueFieldIndexer) FromObject(raw interface{}) (bool, [][]byte, error) {

	obj, ok := raw.(ContextRecord)
	if !ok {
		return false, nil, fmt.Errorf("Invalid type of object %T in SingleValueFieldIndexer, %v", raw, raw)
	}

	// We need cartesian product of all field values in all combinations
	// First we build the vectors (slices) with all values for each field
	fieldValueVectors := [][]interface{}{}
	for _, f := range s.Fields {
		fvv := []interface{}{}
		items := jsonquery.Find(obj.Data, f)
		for _, itmPtr := range items {
			itm := (*itmPtr).Value().([]any)
			for _, idx := range itm {
				// fmt.Printf("-------------------------3 %v %T\n", idx, idx)
				fvv = append(fvv, fmt.Sprintf("%s", idx))
			}
		}
		fieldValueVectors = append(fieldValueVectors, fvv)
	}

	products := cartesianProduct(fieldValueVectors...)
	retval := [][]byte{}
	for _, product := range products {
		retval = append(retval, indexValue(product))
	}

	// fmt.Printf("-------------------------1 %v\n", string(retval[0]))
	// fmt.Printf("-------------------------2 %v\n", fieldValueVectors)

	return true, retval, nil
}

// FromArgs takes in a slice of args and returns its byte form.
func (s *MultiValueFieldIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != len(s.Fields) {
		return nil, fmt.Errorf("wrong number of args %d, expected %d", len(args), len(s.Fields))
	}
	retval := indexValue(args)
	return retval, nil
}

func indexValue(vals []interface{}) []byte {
	retval := ""
	for _, v := range vals {
		retval = retval + "\x01" + fmt.Sprintf("%v", v)
	}
	retval = retval + "\x00"
	return []byte(retval)
}

// written by GPT-4 and only slightly adjusted - amazing and scary
func cartesianProduct(vectors ...[]interface{}) [][]interface{} {
	if len(vectors) == 0 {
		return [][]interface{}{}
	}

	// Initialize the result with the first vector
	result := [][]interface{}{}
	for _, element := range vectors[0] {
		result = append(result, []interface{}{element})
	}

	// Iterate over the remaining vectors and compute the Cartesian product
	for _, vector := range vectors[1:] {
		temp := [][]interface{}{}
		for _, element := range vector {
			for _, product := range result {
				temp = append(temp, append(product, element))
			}
		}
		result = temp
	}

	return result
}

//=====================================================================================
// Context Database  - the database stores tables with different sources and structures
// of related entities needed for either filtering or enhancing metric data coming through
// the receiver
//
// data records are stored as jsonquery.Node structures with timestampes (for purging)
// indexes are built on valued taken as a result of jsonquery with a query taken from a field
// definition of an index
// indexes are supported either single value or multiple value per record
//=====================================================================================

type ContextDbSchemaFile struct {
	Schemas ContextDbSchema `yaml:"schemas"`
}
type ContextDbSchema []*ContextTableSchema

type ContextTableSchema struct {
	Name    string              `yaml:"name"`
	Indexes []ContextTableIndex `yaml:"indexes"`
}

type ContextTableIndex struct {
	Name       string   `yaml:"name"`
	Unique     bool     `yaml:"unique"`
	MultiValue bool     `yaml:"multiValue"`
	Fields     []string `yaml:"fields"`
}

type ContextData *jsonquery.Node

type ContextRecord struct {
	Data              ContextData `yaml:"data"`
	LastUpdatedMillis int64       `yaml:"lastUpdatedMillis"`
}

func (mr ContextRecord) String() string {
	node := (*mr.Data)
	nodeXml := node.OutputXML()
	nodeXml = strings.ReplaceAll(nodeXml, "<nil>", "")
	nodeJson, err := xj.Convert(strings.NewReader(nodeXml))
	if err != nil {
		return fmt.Sprintf("Unconvertable %s to json", nodeXml)
	}
	return fmt.Sprintf("LastUpd: %s, Data %v",
		time.UnixMilli(mr.LastUpdatedMillis),
		nodeJson,
	)
}

type ContextDb struct {
	Schema *memdb.DBSchema
	Db     *memdb.MemDB
	logger *zap.Logger
}

func ParseDbJsonSchema(yamlSchema []byte) (ContextDbSchema, error) {
	schema := ContextDbSchemaFile{}
	err := yaml.Unmarshal(yamlSchema, &schema)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse DB YAML schema - %v", err)
	}

	return schema.Schemas, nil
}

func AppendDbJsonSchema(schema ContextDbSchema, newSchema ContextDbSchema) ContextDbSchema {
	result := ContextDbSchema{}
	for _, s := range schema {
		result = append(result, s)
	}
	for _, s := range newSchema {
		result = append(result, s)
	}

	return result
}

func GetDbSchema(schema ContextDbSchema) (*memdb.DBSchema, error) {
	tables := map[string]*memdb.TableSchema{}
	for _, tbl := range schema {
		tableSchema, err := GetTableSchema(tbl)
		if err != nil {
			return nil, err
		}
		tables[tbl.Name] = tableSchema
	}
	dbSchema := memdb.DBSchema{
		Tables: tables,
	}
	return &dbSchema, nil
}

func GetTableSchema(table *ContextTableSchema) (*memdb.TableSchema, error) {

	tableIndexes, err := GetTableIndexes(table)
	if err != nil {
		return nil, err
	}
	tableSchema := memdb.TableSchema{
		Name:    table.Name,
		Indexes: tableIndexes,
	}
	return &tableSchema, nil
}

func GetTableIndexes(table *ContextTableSchema) (map[string]*memdb.IndexSchema, error) {
	tableIndexes := map[string]*memdb.IndexSchema{}
	for _, idx := range table.Indexes {
		tableIndex, err := GetTableIndex(&idx)
		if err != nil {
			return nil, err
		}
		tableIndexes[idx.Name] = tableIndex
	}

	return tableIndexes, nil
}

func GetTableIndex(index *ContextTableIndex) (*memdb.IndexSchema, error) {

	var indexer memdb.Indexer
	if index.MultiValue {
		indexer = &MultiValueFieldIndexer{Fields: index.Fields}
	} else {
		indexer = &SingleValueFieldIndexer{Fields: index.Fields}
	}

	tableIndex := memdb.IndexSchema{
		Name:    index.Name,
		Indexer: indexer,
		Unique:  index.Unique,
	}

	return &tableIndex, nil
}

func (mdb *ContextDb) Init(schema *memdb.DBSchema, logger *zap.Logger) error {
	mdb.Schema = schema
	mdb.logger = logger

	db, err := memdb.NewMemDB(mdb.Schema)
	if err != nil {
		mdb.logger.Sugar().Infof("Cannot initialize memdb: %v\n", err)
		return err
	}

	mdb.Db = db

	return nil
}

func (mdb *ContextDb) InsertOrUpdateRecord(tableName string, rec *ContextRecord) error {
	txn := mdb.Db.Txn(true)
	defer txn.Commit()

	rec.LastUpdatedMillis = time.Now().UnixMilli()
	err := txn.Insert(tableName, *rec)
	if err != nil {
		mdb.logger.Sugar().Infof("Cannot insert into memdb: %v\n", err)
		return err
	}

	return nil
}

func (mdb *ContextDb) GetOneRecord(tableName string, indexName string, fields ...string) (*ContextRecord, error) {
	txn := mdb.Db.Txn(true)
	defer txn.Commit()

	fldsAny := []interface{}{}
	for _, fld := range fields {
		fldsAny = append(fldsAny, fld)
	}

	v, err := txn.First(tableName, indexName, fldsAny...)
	if err != nil {
		mdb.logger.Sugar().Infof("Cannot find record in table %s, index %s, fields %v: %v\n", tableName, indexName, fields, err)
		return nil, err
	}

	rec, ok := v.(ContextRecord)
	if !ok {
		mdb.logger.Sugar().Infof("Invalid record type in table %s, index %s, fields %v: %v\n", tableName, indexName, fields)
		return nil, err
	}

	return &rec, nil
}

func (mdb *ContextDb) GetAllRecords(tableName string, indexName string, fields ...string) ([]ContextRecord, error) {
	txn := mdb.Db.Txn(false)
	defer txn.Commit()

	recs := []ContextRecord{}
	fldsAny := []interface{}{}
	for _, fld := range fields {
		fldsAny = append(fldsAny, fld)
	}
	iterator, err := txn.LowerBound(tableName, indexName, fldsAny...)
	if err != nil {
		mdb.logger.Sugar().Infof("Cannot find record in table %s, index %s, fields %v: %v\n", tableName, indexName, fields, err)
		return nil, err
	}
	for obj := iterator.Next(); obj != nil; obj = iterator.Next() {
		rec, ok := obj.(ContextRecord)
		if !ok {
			mdb.logger.Sugar().Infof("Invalid record type in table %s, index %s, fields %v: %v\n", tableName, indexName, fields, err)
			continue
		}
		if mdb.isStillValid(tableName, indexName, fldsAny, rec) { // Check if current record still within query bounds
			recs = append(recs, rec)
		}
	}
	recs = mdb.removeRedundancies(tableName, recs)
	return recs, nil
}

func (mdb *ContextDb) DeleteRecord(tableName string, record *ContextRecord) error {
	txn := mdb.Db.Txn(true)
	defer txn.Commit()

	err := txn.Delete(tableName, record)
	if err != nil {
		mdb.logger.Sugar().Infof("Cannot delete record in table %s, record %v: %v\n", tableName, record, err)
		return err
	}
	return nil
}

func (mdb *ContextDb) PurgeRecordsOlderThan(tableName string, ageInMins int) error {

	txn := mdb.Db.Txn(true)
	defer txn.Commit()
	txn.Get(tableName, "id")
	iterator, err := txn.Get(tableName, "id")
	if err != nil {
		mdb.logger.Sugar().Infof("Cannot get records in table %s - %v\n", tableName, err)
		return err
	}
	for obj := iterator.Next(); obj != nil; obj = iterator.Next() {
		r, ok := obj.(ContextRecord)
		if !ok {
			mdb.logger.Sugar().Infof("Invalid record type in table %s of type %T: %v\n", tableName, obj, err)
			continue
		}
		// fmt.Println(r)
		if r.LastUpdatedMillis > (time.Now().UnixMilli() - (int64(ageInMins) * 60 * 1000)) {
			break
		}
		mdb.DeleteRecord(tableName, &r)
	}

	return nil
}

func (mdb *ContextDb) Dump(tableName string) {
	txn := mdb.Db.Txn(false)
	defer txn.Commit()
	cnt := 0
	txn.Get(tableName, "id")
	iterator, err := txn.Get(tableName, "id")
	if err != nil {
		log.Printf("Cannot get rels - %v\n", err)
		return
	}
	fmt.Println("Table: ", tableName)
	for obj := iterator.Next(); obj != nil; obj = iterator.Next() {
		r, ok := obj.(ContextRecord)
		if !ok {
			log.Printf("Invalid object type %T in db: %v\n", obj, obj)
			continue
		}
		cnt++
		fmt.Println(r.String())
	}
	fmt.Println("Total count: ", cnt)

	return

}

func (mdb *ContextDb) isStillValid(tableName string, indexName string, fldAny []any, rec ContextRecord) bool {
	table := mdb.Db.DBSchema().Tables[tableName]
	indexer := table.Indexes[indexName].Indexer
	indexInternal, err := indexer.FromArgs(fldAny...)
	if err != nil {
		mdb.logger.Sugar().Errorf("Error getting indexInternal for args %v - %v", fldAny, err)
		return false
	}
	if multiIndexer, ok := indexer.(*MultiValueFieldIndexer); ok {
		_, indexesRec, err := multiIndexer.FromObject(rec)
		if err != nil {
			mdb.logger.Sugar().Errorf("Error getting indexesRec for rec %v - %v", rec, err)
			return false
		}
		for _, indexRec := range indexesRec {
			if bytes.Equal(indexInternal, indexRec) {
				return true
			}
		}
		return false
	} else {
		singleIndexer := indexer.(*SingleValueFieldIndexer)
		_, indexRec, err := singleIndexer.FromObject(rec)
		if err != nil {
			mdb.logger.Sugar().Errorf("Error getting indexRec for rec %v - %v", rec, err)
			return false
		}
		return bytes.Equal(indexInternal, indexRec)
	}
}

func (mdb *ContextDb) removeRedundancies(tableName string, recs []ContextRecord) []ContextRecord {
	table := mdb.Db.DBSchema().Tables[tableName]
	idIndexer := table.Indexes["id"].Indexer.(*SingleValueFieldIndexer)

	idIndexMap := map[string]bool{}
	uniqueRecs := []ContextRecord{}
	for _, rec := range recs {
		_, index, err := idIndexer.FromObject(rec)
		if err != nil {
			mdb.logger.Sugar().Errorf("Error getting index for rec %v - %v", rec, err)
			return []ContextRecord{}
		}
		if _, ok := idIndexMap[string(index)]; ok {
			continue
		}
		idIndexMap[string(index)] = true
		uniqueRecs = append(uniqueRecs, rec)
	}
	return uniqueRecs
}

func (s *ContextDb) doc2json(doc *jsonquery.Node) string {
	xmlStr := doc.OutputXML()
	xmlStr = strings.ReplaceAll(xmlStr, "<nil>", "")
	xmlReader := strings.NewReader(xmlStr)
	jsonStr, err := xj.Convert(xmlReader)
	if err != nil {
		s.logger.Sugar().Errorf("Error converting doc to JSON - XML: %s", xmlStr, err)
		return xmlStr
	}
	return jsonStr.String()
}
