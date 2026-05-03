package digitaltwin

import (
	"strings"
)

// ─── Tokenizer ───────────────────────────────────────────────────────────────

type tokenType int

const (
	tokKeyword  tokenType = iota // SELECT, WHERE, FILTER, ORDER, BY, LIMIT, OFFSET, ASC, DESC, PREFIX, OPTIONAL
	tokVariable                  // ?varName
	tokURI                       // :localName or <full-uri>
	tokLiteral                   // "string"
	tokNumber                    // 42, 3.14
	tokPunct                     // { } ( ) . , ; =
	tokDot                       // .
	tokEOF
)

type token struct {
	typ tokenType
	val string
}

var sparqlKeywords = map[string]bool{
	"SELECT": true, "WHERE": true, "FILTER": true, "ORDER": true, "BY": true,
	"LIMIT": true, "OFFSET": true, "ASC": true, "DESC": true, "PREFIX": true,
	"OPTIONAL": true, "FROM": true, "DISTINCT": true, "A": true,
	"GROUP": true, "HAVING": true,
	"COUNT": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true, "AS": true,
}

func tokenizeSPARQL(query string) []token {
	var tokens []token
	i := 0
	runes := []rune(query)
	n := len(runes)

	for i < n {
		// Skip whitespace
		if runes[i] == ' ' || runes[i] == '\t' || runes[i] == '\n' || runes[i] == '\r' {
			i++
			continue
		}
		// Comment
		if runes[i] == '#' {
			for i < n && runes[i] != '\n' {
				i++
			}
			continue
		}
		// Variable: ?name
		if runes[i] == '?' {
			i++
			start := i
			for i < n && (isAlphaNum(runes[i]) || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, token{tokVariable, string(runes[start:i])})
			continue
		}
		// Prefixed URI: :localName or prefix:local
		if runes[i] == ':' || (i+1 < n && isAlpha(runes[i]) && runes[i+1] == ':') {
			if runes[i] == ':' {
				i++ // skip leading colon
				start := i
				for i < n && (isAlphaNum(runes[i]) || runes[i] == '_' || runes[i] == '-') {
					i++
				}
				tokens = append(tokens, token{tokURI, string(runes[start:i])})
			} else {
				// prefix:local
				start := i
				for i < n && runes[i] != ' ' && runes[i] != '\t' && runes[i] != '\n' && runes[i] != '.' && runes[i] != ';' && runes[i] != ')' && runes[i] != '}' {
					i++
				}
				tokens = append(tokens, token{tokURI, string(runes[start:i])})
			}
			continue
		}
		// Full URI: <uri>
		if runes[i] == '<' {
			i++
			start := i
			for i < n && runes[i] != '>' {
				i++
			}
			uri := string(runes[start:i])
			if i < n {
				i++ // skip >
			}
			tokens = append(tokens, token{tokURI, uri})
			continue
		}
		// String literal: "..."
		if runes[i] == '"' {
			i++
			start := i
			for i < n && runes[i] != '"' {
				if runes[i] == '\\' {
					i++ // skip escape
				}
				i++
			}
			lit := string(runes[start:i])
			if i < n {
				i++ // skip closing "
			}
			tokens = append(tokens, token{tokLiteral, lit})
			continue
		}
		// Number
		if isDigit(runes[i]) || (runes[i] == '-' && i+1 < n && isDigit(runes[i+1])) {
			start := i
			if runes[i] == '-' {
				i++
			}
			for i < n && (isDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			tokens = append(tokens, token{tokNumber, string(runes[start:i])})
			continue
		}
		// Punctuation
		if runes[i] == '{' || runes[i] == '}' || runes[i] == '(' || runes[i] == ')' ||
			runes[i] == ',' || runes[i] == ';' {
			tokens = append(tokens, token{tokPunct, string(runes[i])})
			i++
			continue
		}
		if runes[i] == '.' {
			tokens = append(tokens, token{tokDot, "."})
			i++
			continue
		}
		// Comparison operators
		if runes[i] == '>' || runes[i] == '<' || runes[i] == '=' || runes[i] == '!' {
			op := string(runes[i])
			if i+1 < n && runes[i+1] == '=' {
				op += "="
				i++
			}
			tokens = append(tokens, token{tokPunct, op})
			i++
			continue
		}
		// Keyword or identifier
		if isAlpha(runes[i]) || runes[i] == '_' {
			start := i
			for i < n && (isAlphaNum(runes[i]) || runes[i] == '_') {
				i++
			}
			word := string(runes[start:i])
			upper := strings.ToUpper(word)
			if sparqlKeywords[upper] {
				tokens = append(tokens, token{tokKeyword, upper})
			} else {
				// Could be a bare URI local name after a colon was separate,
				// or a type name in a rdf:type shorthand 'a'
				if upper == "A" {
					tokens = append(tokens, token{tokKeyword, "A"})
				} else {
					tokens = append(tokens, token{tokURI, word})
				}
			}
			continue
		}
		i++ // skip unrecognized char
	}

	tokens = append(tokens, token{tokEOF, ""})
	return tokens
}

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
func isDigit(r rune) bool    { return r >= '0' && r <= '9' }
func isAlphaNum(r rune) bool { return isAlpha(r) || isDigit(r) }
