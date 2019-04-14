package provision

import "github.com/txn2/es"

// GetAccountMapping
func GetAccountMapping(prefix string) es.IndexTemplate {
	template := es.Obj{
		"index_patterns": []string{prefix + IdxAccount},
		"settings": es.Obj{
			"number_of_shards": 2,
		},
		"mappings": es.Obj{
			"_doc": es.Obj{
				"_source": es.Obj{
					"enabled": true,
				},
				"properties": es.Obj{
					"id": es.Obj{
						"type": "text",
					},
					"description": es.Obj{
						"type": "text",
					},
					"display_name": es.Obj{
						"type": "text",
					},
					"active": es.Obj{
						"type": "boolean",
					},
					"org_id": es.Obj{
						"type": "integer",
					},
					"access_keys": es.Obj{
						"type": "nested",
						"properties": es.Obj{
							"name": es.Obj{
								"type": "text",
							},
							"description": es.Obj{
								"type": "text",
							},
							"key": es.Obj{
								"type": "keyword",
							},
							"active": es.Obj{
								"type": "boolean",
							},
						},
					},
				},
			},
		},
	}

	return es.IndexTemplate{
		Name:     prefix + IdxAccount,
		Template: template,
	}
}
