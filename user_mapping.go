package provision

import "github.com/txn2/es"

// GetUserMapping
func GetUserMapping(prefix string) es.IndexTemplate {
	template := es.Obj{
		"index_patterns": []string{prefix + IdxUser},
		"settings": es.Obj{
			"number_of_shards": 5,
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
					"sysop": es.Obj{
						"type": "boolean",
					},
					"password": es.Obj{
						"type": "keyword",
					},
					"sections_all": es.Obj{
						"type": "boolean",
					},
				},
			},
		},
	}

	return es.IndexTemplate{
		Name:     prefix + IdxUser,
		Template: template,
	}
}
