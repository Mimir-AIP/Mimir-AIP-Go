package digitaltwin

import (
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (s *Service) initializeFromOntology(twin *models.DigitalTwin, ont *models.Ontology) error {
	if ont.Content == "" {
		return nil
	}

	classes := parseOntologyClasses(ont.Content)
	if len(classes) == 0 {
		return nil
	}

	if twin.Metadata == nil {
		twin.Metadata = make(map[string]interface{})
	}

	entityTypes := make(map[string]interface{}, len(classes))
	for _, cls := range classes {
		entityTypes[cls.Name] = map[string]interface{}{
			"label":      cls.Label,
			"properties": cls.Properties,
		}
	}
	twin.Metadata["entity_types"] = entityTypes

	return s.store.SaveDigitalTwin(twin)
}

// parseOntologyClasses extracts class and property definitions from Turtle content
func parseOntologyClasses(turtleContent string) []ontologyClass {
	var classes []ontologyClass
	lines := strings.Split(turtleContent, "\n")

	classMap := make(map[string]*ontologyClass)
	domainMap := make(map[string]string) // property → class name

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect owl:Class declarations: ":ClassName a owl:Class" or ":ClassName rdf:type owl:Class"
		if (strings.Contains(line, "owl:Class") || strings.Contains(line, "owl:class")) &&
			(strings.Contains(line, " a ") || strings.Contains(line, "rdf:type")) {
			name := extractTurtleSubject(line)
			if name != "" && classMap[name] == nil {
				cls := &ontologyClass{Name: name, Label: name}
				classMap[name] = cls
			}
		}

		// Detect rdfs:label: ":ClassName rdfs:label "Label""
		if strings.Contains(line, "rdfs:label") {
			subject := extractTurtleSubject(line)
			label := extractStringLiteral(line)
			if subject != "" && label != "" {
				if cls, ok := classMap[subject]; ok {
					cls.Label = label
				}
			}
		}

		// Detect owl:DatatypeProperty declarations with rdfs:domain
		if strings.Contains(line, "owl:DatatypeProperty") || strings.Contains(line, "owl:ObjectProperty") {
			name := extractTurtleSubject(line)
			if name != "" {
				domainMap[name] = "" // register property, domain resolved later
			}
		}

		// Detect rdfs:domain: ":property rdfs:domain :ClassName"
		if strings.Contains(line, "rdfs:domain") {
			subject := extractTurtleSubject(line)
			domain := extractTurtleObject(line)
			if subject != "" && domain != "" {
				domainMap[subject] = domain
				if cls, ok := classMap[domain]; ok {
					if !containsString(cls.Properties, subject) {
						cls.Properties = append(cls.Properties, subject)
					}
				}
			}
		}
	}

	// Post-process: assign any unresolved properties to classes
	for prop, domain := range domainMap {
		if domain == "" {
			continue
		}
		if cls, ok := classMap[domain]; ok {
			if !containsString(cls.Properties, prop) {
				cls.Properties = append(cls.Properties, prop)
			}
		}
	}

	for _, cls := range classMap {
		classes = append(classes, *cls)
	}
	return classes
}

func extractTurtleSubject(line string) string {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	// Strip leading colon if present
	s = strings.TrimPrefix(s, ":")
	// Strip trailing colon if prefix:local format
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.Trim(s, "<>.,;")
}

func extractTurtleObject(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return ""
	}
	s := parts[len(parts)-1]
	s = strings.TrimPrefix(s, ":")
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.Trim(s, "<>.,;")
}

func extractStringLiteral(line string) string {
	start := strings.Index(line, `"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// syncFromStorage retrieves CIR data from a storage source and creates or
// merges entities in the digital twin.
//
// Entity resolution: before creating a new entity, the function checks whether
// an entity of the same type with the same key-field value already exists
// (e.g. from a previous pipeline that wrote to a different storage).  If found,
// the new attributes are merged into the existing entity rather than creating
