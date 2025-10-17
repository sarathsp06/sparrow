package jobs

// DataProcessingArgs represents arguments for data processing jobs
type DataProcessingArgs struct {
	DataID   int    `json:"data_id"`
	DataType string `json:"data_type"`
}

// Kind returns the job type name
func (DataProcessingArgs) Kind() string { return "data_processing" }
