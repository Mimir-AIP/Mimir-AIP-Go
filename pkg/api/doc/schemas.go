package doc

func init() {
	RegisterSchemas(mergeSchemaGroups(
		systemProjectPipelineSchemas(),
		storageAnalysisOntologyExtractionSchemas(),
		mlModelSchemas(),
		digitalTwinSchemas(),
		workTaskSchemas(),
	))
}

func mergeSchemaGroups(groups ...M) M {
	out := M{}
	for _, group := range groups {
		for name, schema := range group {
			out[name] = schema
		}
	}
	return out
}
