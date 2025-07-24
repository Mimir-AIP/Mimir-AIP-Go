//Responsible for executing a pipeline
package utils
import (
	"fmt"
	"strings"
)
func RunPipeline(pipeline string, args ...string) error {
	//take the pipeline yaml path, read file and execute the pipeline, line by line
	if pipeline == "" {
		return fmt.Errorf("pipeline path cannot be empty")
	}
	if !strings.HasSuffix(pipeline, ".yaml") {
		return fmt.Errorf("pipeline path must end with .yaml")
	}
	