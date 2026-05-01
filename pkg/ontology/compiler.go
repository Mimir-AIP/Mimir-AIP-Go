package ontology

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

const (
	rdfType        = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	owlClass       = "http://www.w3.org/2002/07/owl#Class"
	rdfsClass      = "http://www.w3.org/2000/01/rdf-schema#Class"
	owlObjectProp  = "http://www.w3.org/2002/07/owl#ObjectProperty"
	owlDataProp    = "http://www.w3.org/2002/07/owl#DatatypeProperty"
	owlAnnotation  = "http://www.w3.org/2002/07/owl#AnnotationProperty"
	rdfsLabel      = "http://www.w3.org/2000/01/rdf-schema#label"
	rdfsComment    = "http://www.w3.org/2000/01/rdf-schema#comment"
	rdfsSubClassOf = "http://www.w3.org/2000/01/rdf-schema#subClassOf"
	rdfsDomain     = "http://www.w3.org/2000/01/rdf-schema#domain"
	rdfsRange      = "http://www.w3.org/2000/01/rdf-schema#range"
	owlInverseOf   = "http://www.w3.org/2002/07/owl#inverseOf"
)

var prefixLineRE = regexp.MustCompile(`(?m)^\s*@prefix\s+([A-Za-z][A-Za-z0-9_-]*|):\s*<([^>]+)>\s*\.\s*$`)

// CompileOntology converts supported Turtle/OWL terms into Mimir's canonical ontology graph.
// It intentionally supports the subset Mimir can enforce today instead of pretending to be a full OWL reasoner.
func CompileOntology(record *models.Ontology) (*models.CompiledOntology, error) {
	if record == nil {
		return nil, fmt.Errorf("ontology cannot be nil")
	}
	return CompileOntologyContent(record.ID, record.ProjectID, record.Name, record.Version, record.Content)
}

func CompileOntologyContent(ontologyID, projectID, name, version, content string) (*models.CompiledOntology, error) {
	compiled := &models.CompiledOntology{
		OntologyID:  strings.TrimSpace(ontologyID),
		ProjectID:   strings.TrimSpace(projectID),
		Name:        strings.TrimSpace(name),
		Version:     strings.TrimSpace(version),
		ContentHash: hashContent(content),
		Prefixes:    map[string]string{},
		CompiledAt:  time.Now().UTC(),
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		compiled.Diagnostics = append(compiled.Diagnostics, diag(models.OntologyDiagnosticError, "empty_content", "ontology content is required", 1, 1, ""))
		return compiled, fmt.Errorf("ontology content is required")
	}
	for _, match := range prefixLineRE.FindAllStringSubmatch(trimmed, -1) {
		compiled.Prefixes[match[1]] = match[2]
	}
	if len(compiled.Prefixes) == 0 {
		compiled.Diagnostics = append(compiled.Diagnostics, diag(models.OntologyDiagnosticError, "missing_prefix", "ontology content must declare at least one Turtle @prefix", 1, 1, ""))
		return compiled, fmt.Errorf("ontology content must declare at least one Turtle @prefix")
	}
	addCommonPrefixes(compiled.Prefixes)

	statements, splitDiagnostics := splitStatements(trimmed)
	compiled.Diagnostics = append(compiled.Diagnostics, splitDiagnostics...)
	triples := make([]triple, 0)
	for _, statement := range statements {
		parsed, parseDiagnostics := parseStatement(statement, compiled.Prefixes)
		compiled.Diagnostics = append(compiled.Diagnostics, parseDiagnostics...)
		triples = append(triples, parsed...)
	}

	classes, properties := compileTerms(triples)
	compiled.Classes = finalizeClasses(classes)
	compiled.Properties = finalizeProperties(properties)
	compiled.Relations = compileRelations(compiled.Properties)
	compiled.SearchTerms = compileSearchTerms(compiled.Classes, compiled.Properties, compiled.Relations)
	compiled.Diagnostics = append(compiled.Diagnostics, semanticDiagnostics(compiled)...)
	sortDiagnostics(compiled.Diagnostics)
	if hasDiagnosticSeverity(compiled.Diagnostics, models.OntologyDiagnosticError) {
		return compiled, fmt.Errorf("ontology contains validation errors")
	}
	return compiled, nil
}

type triple struct {
	Subject   term
	Predicate term
	Object    term
	Line      int
}

type term struct {
	Raw     string
	URI     string
	Literal string
}

type mutableClass struct {
	id, uri, name, label, description string
	subClassOf                        map[string]struct{}
}

type mutableProperty struct {
	id, uri, name, label, description, kind, inverseOf string
	domain, valueRange                                 map[string]struct{}
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func diag(severity models.OntologyDiagnosticSeverity, code, message string, line, column int, subject string) models.OntologyDiagnostic {
	return models.OntologyDiagnostic{Severity: severity, Code: code, Message: message, Line: line, Column: column, Subject: subject}
}

func addCommonPrefixes(prefixes map[string]string) {
	defaults := map[string]string{
		"rdf":  "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
		"rdfs": "http://www.w3.org/2000/01/rdf-schema#",
		"owl":  "http://www.w3.org/2002/07/owl#",
		"xsd":  "http://www.w3.org/2001/XMLSchema#",
	}
	for prefix, iri := range defaults {
		if _, ok := prefixes[prefix]; !ok {
			prefixes[prefix] = iri
		}
	}
}

func splitStatements(content string) ([]statement, []models.OntologyDiagnostic) {
	var statements []statement
	var diagnostics []models.OntologyDiagnostic
	var b strings.Builder
	line := 1
	statementLine := 1
	inIRI := false
	inString := false
	escaped := false
	for _, r := range content {
		if b.Len() == 0 && !unicode.IsSpace(r) {
			statementLine = line
		}
		switch {
		case escaped:
			escaped = false
		case inString && r == '\\':
			escaped = true
		case r == '"' && !inIRI:
			inString = !inString
		case r == '<' && !inString:
			inIRI = true
		case r == '>' && !inString:
			inIRI = false
		case r == '.' && !inIRI && !inString:
			text := strings.TrimSpace(b.String())
			if text != "" && !strings.HasPrefix(text, "@prefix") {
				statements = append(statements, statement{Text: text, Line: statementLine})
			}
			b.Reset()
			continue
		}
		b.WriteRune(r)
		if r == '\n' {
			line++
		}
	}
	if strings.TrimSpace(b.String()) != "" {
		diagnostics = append(diagnostics, diag(models.OntologyDiagnosticError, "unterminated_statement", "Turtle statement is missing a terminating period", statementLine, 1, ""))
	}
	return statements, diagnostics
}

type statement struct {
	Text string
	Line int
}

func parseStatement(stmt statement, prefixes map[string]string) ([]triple, []models.OntologyDiagnostic) {
	parts := splitTopLevel(stmt.Text, ';')
	if len(parts) == 0 {
		return nil, nil
	}
	subjectToken, predicateObject, ok := firstToken(parts[0])
	if !ok {
		return nil, []models.OntologyDiagnostic{diag(models.OntologyDiagnosticError, "invalid_statement", "statement must include subject, predicate, and object", stmt.Line, 1, "")}
	}
	subject, err := parseTerm(subjectToken, prefixes)
	if err != nil {
		return nil, []models.OntologyDiagnostic{diag(models.OntologyDiagnosticError, "invalid_subject", err.Error(), stmt.Line, 1, subjectToken)}
	}
	segments := append([]string{predicateObject}, parts[1:]...)
	triples := make([]triple, 0, len(segments))
	diagnostics := make([]models.OntologyDiagnostic, 0)
	for _, segment := range segments {
		predToken, objectText, ok := firstToken(segment)
		if !ok {
			continue
		}
		predicate, err := parsePredicate(predToken, prefixes)
		if err != nil {
			diagnostics = append(diagnostics, diag(models.OntologyDiagnosticError, "invalid_predicate", err.Error(), stmt.Line, 1, predToken))
			continue
		}
		for _, objectToken := range splitTopLevel(objectText, ',') {
			objectToken = strings.TrimSpace(objectToken)
			if objectToken == "" {
				continue
			}
			object, err := parseTerm(objectToken, prefixes)
			if err != nil {
				diagnostics = append(diagnostics, diag(models.OntologyDiagnosticError, "invalid_object", err.Error(), stmt.Line, 1, objectToken))
				continue
			}
			triples = append(triples, triple{Subject: subject, Predicate: predicate, Object: object, Line: stmt.Line})
		}
	}
	return triples, diagnostics
}

func firstToken(text string) (string, string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", "", false
	}
	inIRI := false
	inString := false
	escaped := false
	for i, r := range text {
		switch {
		case escaped:
			escaped = false
		case inString && r == '\\':
			escaped = true
		case r == '"' && !inIRI:
			inString = !inString
		case r == '<' && !inString:
			inIRI = true
		case r == '>' && !inString:
			inIRI = false
		case unicode.IsSpace(r) && !inIRI && !inString:
			return strings.TrimSpace(text[:i]), strings.TrimSpace(text[i:]), strings.TrimSpace(text[i:]) != ""
		}
	}
	return "", "", false
}

func splitTopLevel(text string, sep rune) []string {
	var parts []string
	var b strings.Builder
	inIRI := false
	inString := false
	escaped := false
	for _, r := range text {
		switch {
		case escaped:
			escaped = false
		case inString && r == '\\':
			escaped = true
		case r == '"' && !inIRI:
			inString = !inString
		case r == '<' && !inString:
			inIRI = true
		case r == '>' && !inString:
			inIRI = false
		case r == sep && !inIRI && !inString:
			parts = append(parts, strings.TrimSpace(b.String()))
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	parts = append(parts, strings.TrimSpace(b.String()))
	return parts
}

func parsePredicate(token string, prefixes map[string]string) (term, error) {
	if token == "a" {
		return term{Raw: token, URI: rdfType}, nil
	}
	return parseTerm(token, prefixes)
}

func parseTerm(token string, prefixes map[string]string) (term, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return term{}, fmt.Errorf("empty term")
	}
	if strings.HasPrefix(token, "\"") {
		literal, err := parseLiteral(token)
		if err != nil {
			return term{}, err
		}
		return term{Raw: token, Literal: literal}, nil
	}
	if strings.HasPrefix(token, "<") && strings.Contains(token, ">") {
		end := strings.Index(token, ">")
		return term{Raw: token, URI: token[1:end]}, nil
	}
	if strings.Contains(token, ":") {
		parts := strings.SplitN(token, ":", 2)
		base, ok := prefixes[parts[0]]
		if !ok {
			return term{}, fmt.Errorf("undeclared prefix %q", parts[0])
		}
		return term{Raw: token, URI: base + parts[1]}, nil
	}
	return term{}, fmt.Errorf("unsupported term %q", token)
}

func parseLiteral(token string) (string, error) {
	var b strings.Builder
	escaped := false
	for i, r := range token[1:] {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			return b.String(), nil
		}
		b.WriteRune(r)
		if i == len(token)-2 {
			break
		}
	}
	return "", fmt.Errorf("unterminated string literal")
}

func compileTerms(triples []triple) (map[string]*mutableClass, map[string]*mutableProperty) {
	classes := map[string]*mutableClass{}
	properties := map[string]*mutableProperty{}
	for _, tr := range triples {
		if tr.Predicate.URI == rdfType {
			switch tr.Object.URI {
			case owlClass, rdfsClass:
				ensureClass(classes, tr.Subject)
			case owlObjectProp:
				ensureProperty(properties, tr.Subject).kind = "object"
			case owlDataProp:
				ensureProperty(properties, tr.Subject).kind = "datatype"
			case owlAnnotation:
				ensureProperty(properties, tr.Subject).kind = "annotation"
			}
		}
	}
	for _, tr := range triples {
		subjectURI := tr.Subject.URI
		if subjectURI == "" {
			continue
		}
		if class, ok := classes[subjectURI]; ok {
			applyClassTriple(class, tr)
		}
		if property, ok := properties[subjectURI]; ok {
			applyPropertyTriple(property, tr)
		}
	}
	return classes, properties
}

func ensureClass(classes map[string]*mutableClass, t term) *mutableClass {
	if class, ok := classes[t.URI]; ok {
		return class
	}
	class := &mutableClass{id: localName(t.URI), uri: t.URI, name: localName(t.URI), subClassOf: map[string]struct{}{}}
	classes[t.URI] = class
	return class
}

func ensureProperty(properties map[string]*mutableProperty, t term) *mutableProperty {
	if property, ok := properties[t.URI]; ok {
		return property
	}
	property := &mutableProperty{id: localName(t.URI), uri: t.URI, name: localName(t.URI), kind: "unknown", domain: map[string]struct{}{}, valueRange: map[string]struct{}{}}
	properties[t.URI] = property
	return property
}

func applyClassTriple(class *mutableClass, tr triple) {
	switch tr.Predicate.URI {
	case rdfsLabel:
		if tr.Object.Literal != "" {
			class.label = tr.Object.Literal
		}
	case rdfsComment:
		if tr.Object.Literal != "" {
			class.description = tr.Object.Literal
		}
	case rdfsSubClassOf:
		if tr.Object.URI != "" {
			class.subClassOf[localName(tr.Object.URI)] = struct{}{}
		}
	}
}

func applyPropertyTriple(property *mutableProperty, tr triple) {
	switch tr.Predicate.URI {
	case rdfsLabel:
		if tr.Object.Literal != "" {
			property.label = tr.Object.Literal
		}
	case rdfsComment:
		if tr.Object.Literal != "" {
			property.description = tr.Object.Literal
		}
	case rdfsDomain:
		if tr.Object.URI != "" {
			property.domain[localName(tr.Object.URI)] = struct{}{}
		}
	case rdfsRange:
		if tr.Object.URI != "" {
			property.valueRange[localName(tr.Object.URI)] = struct{}{}
		}
	case owlInverseOf:
		if tr.Object.URI != "" {
			property.inverseOf = localName(tr.Object.URI)
		}
	}
}

func finalizeClasses(classes map[string]*mutableClass) []models.CompiledOntologyClass {
	out := make([]models.CompiledOntologyClass, 0, len(classes))
	for _, class := range classes {
		out = append(out, models.CompiledOntologyClass{
			ID:          class.id,
			URI:         class.uri,
			Name:        class.name,
			Label:       class.label,
			Description: class.description,
			SubClassOf:  sortedKeys(class.subClassOf),
			SearchTerms: uniqueNonEmpty(class.name, class.label, splitCamel(class.name)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func finalizeProperties(properties map[string]*mutableProperty) []models.CompiledOntologyProperty {
	out := make([]models.CompiledOntologyProperty, 0, len(properties))
	for _, property := range properties {
		out = append(out, models.CompiledOntologyProperty{
			ID:          property.id,
			URI:         property.uri,
			Name:        property.name,
			Label:       property.label,
			Description: property.description,
			Kind:        property.kind,
			Domain:      sortedKeys(property.domain),
			Range:       sortedKeys(property.valueRange),
			InverseOf:   property.inverseOf,
			SearchTerms: uniqueNonEmpty(property.name, property.label, splitCamel(property.name)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func compileRelations(properties []models.CompiledOntologyProperty) []models.CompiledOntologyRelation {
	var relations []models.CompiledOntologyRelation
	for _, property := range properties {
		if property.Kind != "object" || len(property.Domain) == 0 || len(property.Range) == 0 {
			continue
		}
		for _, domain := range property.Domain {
			for _, valueRange := range property.Range {
				relations = append(relations, models.CompiledOntologyRelation{PropertyID: property.ID, Name: property.Name, FromClass: domain, ToClass: valueRange, URI: property.URI})
			}
		}
	}
	sort.Slice(relations, func(i, j int) bool {
		if relations[i].FromClass != relations[j].FromClass {
			return relations[i].FromClass < relations[j].FromClass
		}
		if relations[i].Name != relations[j].Name {
			return relations[i].Name < relations[j].Name
		}
		return relations[i].ToClass < relations[j].ToClass
	})
	return relations
}

func compileSearchTerms(classes []models.CompiledOntologyClass, properties []models.CompiledOntologyProperty, relations []models.CompiledOntologyRelation) []models.OntologySearchTerm {
	terms := make([]models.OntologySearchTerm, 0, len(classes)+len(properties)+len(relations))
	for _, class := range classes {
		terms = append(terms, models.OntologySearchTerm{Term: preferredLabel(class.Name, class.Label), Kind: "class", ID: class.ID, URI: class.URI, Weight: 1.0, Aliases: class.SearchTerms, ClassID: class.ID})
	}
	for _, property := range properties {
		weight := 0.8
		if property.Kind == "object" {
			weight = 0.9
		}
		terms = append(terms, models.OntologySearchTerm{Term: preferredLabel(property.Name, property.Label), Kind: "property", ID: property.ID, URI: property.URI, Weight: weight, Aliases: property.SearchTerms, PropertyID: property.ID})
	}
	for _, relation := range relations {
		terms = append(terms, models.OntologySearchTerm{Term: relation.Name, Kind: "relation", ID: relation.PropertyID + ":" + relation.FromClass + ":" + relation.ToClass, URI: relation.URI, Weight: 0.95, ClassID: relation.FromClass, PropertyID: relation.PropertyID})
	}
	sort.Slice(terms, func(i, j int) bool {
		if terms[i].Kind != terms[j].Kind {
			return terms[i].Kind < terms[j].Kind
		}
		return terms[i].ID < terms[j].ID
	})
	return terms
}

func semanticDiagnostics(compiled *models.CompiledOntology) []models.OntologyDiagnostic {
	var diagnostics []models.OntologyDiagnostic
	classIDs := map[string]struct{}{}
	for _, class := range compiled.Classes {
		classIDs[class.ID] = struct{}{}
	}
	if len(compiled.Classes) == 0 {
		diagnostics = append(diagnostics, diag(models.OntologyDiagnosticWarning, "no_classes", "ontology does not define any owl:Class or rdfs:Class terms", 0, 0, ""))
	}
	for _, property := range compiled.Properties {
		if property.Kind == "unknown" {
			diagnostics = append(diagnostics, diag(models.OntologyDiagnosticWarning, "unknown_property_kind", "property is not typed as owl:ObjectProperty, owl:DatatypeProperty, or owl:AnnotationProperty", 0, 0, property.ID))
		}
		for _, domain := range property.Domain {
			if _, ok := classIDs[domain]; !ok {
				diagnostics = append(diagnostics, diag(models.OntologyDiagnosticWarning, "unknown_domain", "property domain does not reference a declared class", 0, 0, property.ID+" -> "+domain))
			}
		}
		if property.Kind == "object" {
			if len(property.Domain) == 0 || len(property.Range) == 0 {
				diagnostics = append(diagnostics, diag(models.OntologyDiagnosticWarning, "incomplete_relation", "object property should declare both rdfs:domain and rdfs:range for ontology-backed retrieval", 0, 0, property.ID))
			}
			for _, valueRange := range property.Range {
				if _, ok := classIDs[valueRange]; !ok {
					diagnostics = append(diagnostics, diag(models.OntologyDiagnosticWarning, "unknown_range", "object property range does not reference a declared class", 0, 0, property.ID+" -> "+valueRange))
				}
			}
		}
	}
	return diagnostics
}

func localName(uri string) string {
	uri = strings.TrimSpace(uri)
	idx := strings.LastIndexAny(uri, "#/")
	if idx >= 0 && idx < len(uri)-1 {
		return uri[idx+1:]
	}
	return uri
}

func preferredLabel(name, label string) string {
	if strings.TrimSpace(label) != "" {
		return strings.TrimSpace(label)
	}
	return strings.TrimSpace(name)
}

func splitCamel(value string) string {
	var out []rune
	for i, r := range value {
		if i > 0 && unicode.IsUpper(r) {
			out = append(out, ' ')
		}
		out = append(out, r)
	}
	return string(out)
}

func uniqueNonEmpty(values ...string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func hasDiagnosticSeverity(diagnostics []models.OntologyDiagnostic, severity models.OntologyDiagnosticSeverity) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == severity {
			return true
		}
	}
	return false
}

func sortDiagnostics(diagnostics []models.OntologyDiagnostic) {
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Severity != diagnostics[j].Severity {
			return diagnostics[i].Severity < diagnostics[j].Severity
		}
		if diagnostics[i].Line != diagnostics[j].Line {
			return diagnostics[i].Line < diagnostics[j].Line
		}
		return diagnostics[i].Code < diagnostics[j].Code
	})
}
