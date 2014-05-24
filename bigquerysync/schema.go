package bigquerysync

import (
	"fmt"
	"time"

	"code.google.com/p/google-api-go-client/bigquery/v2"

	"appengine"
	"appengine/datastore"
)

const (
	StatByPropertyKind = "__Stat_PropertyType_PropertyName_Kind__"
	StatByKindKind     = "__Stat_Kind__"
)

// StatByProperty holds the statistic information about an
// entity property.
type StatByProperty struct {
	Count      int64     `datastore:"count"`
	Bytes      int64     `datastore:"bytes"`
	Type       string    `datastore:"property_type"`
	Name       string    `datastore:"property_name"`
	Kind       string    `datastore:"kind_name"`
	IndexBytes int64     `datastore:"builtin_index_bytes"`
	IndexCount int64     `datastore:"builtin_index_count"`
	Timestamp  time.Time `datastore:"timestamp"`
}

// StatByKind holds the statistic information about an entity kind.
type StatByKind struct {
	Count               int64     `datastore:"count"`
	EntityBytes         int64     `datastore:"entity_bytes"`
	Kind                string    `datastore:"kind_name"`
	IndexBytes          int64     `datastore:"builtin_index_bytes"`
	IndexCount          int64     `datastore:"builtin_index_count"`
	CompositeIndexBytes int64     `datastore:"composite_index_count"`
	CompositeIndexCount int64     `datastore:"composite_index_bytes"`
	Timestamp           time.Time `datastore:"timestamp"`
}

// SchemaForKind guess the schema based on the datastore
// statistics for the specified entity kind.
func SchemaForKind(c appengine.Context, kind string) (*bigquery.TableSchema, error) {
	var (
		k         *datastore.Key
		err       error
		kindStats *StatByKind
	)
	schema := bigquery.TableSchema{
		Fields: make([]*bigquery.TableFieldSchema, 0),
	}

	// Query for kind stats
	k = datastore.NewKey(c, StatByKindKind, kind, 0, nil)
	kindStats = new(StatByKind)
	err = datastore.Get(c, k, kindStats)
	if err != nil {
		return nil, fmt.Errorf("no stats for '%s': %s", kind, err.Error())
	}
	// Parse fields
	q := datastore.NewQuery(StatByPropertyKind).
		Filter("kind_name =", kind).
		Order("property_name")
	for it := q.Run(c); ; {
		s := new(StatByProperty)
		k, err = it.Next(s)
		if err == datastore.Done {
			break
		}
		if err != nil {
			err := fmt.Errorf("can't load property stats %s: %s", kind, err.Error())
			return nil, err
		}
		if !containsField(&schema, s.Name) {
			f := new(bigquery.TableFieldSchema)
			f.Name = s.Name
			if s.Count > kindStats.Count {
				// More property values than entities: must be repeated
				f.Mode = "REPEATED"
			}
			switch s.Type {
			case "Blob", "BlobKey", "Category", "Email", "IM", "Key", "Link",
				"PhoneNumber", "PostalAddress", "Rating", "ShortBlob", "String":
				f.Type = "STRING"
			case "Date/Time":
				f.Type = "TIMESTAMP"
			case "Boolean":
				f.Type = "BOOLEAN"
			case "Float":
				f.Type = "FLOAT"
			case "Integer":
				f.Type = "Integer"
			}
			if f.Type != "" {
				schema.Fields = append(schema.Fields, f)
			}
		}
	}
	return &schema, nil
}

// containsField Checks if we have a field detected already
func containsField(s *bigquery.TableSchema, n string) bool {
	for _, f := range s.Fields {
		if f.Name == n {
			return true
		}
	}
	return false
}
