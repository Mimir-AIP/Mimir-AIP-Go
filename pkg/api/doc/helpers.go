package doc

// ── Schema helpers ────────────────────────────────────────────────────────────

// Ref returns a $ref schema pointing at a named component schema.
func Ref(name string) M { return M{"$ref": "#/components/schemas/" + name} }

// ArrOf returns an array schema whose items are a $ref to name.
func ArrOf(name string) M { return M{"type": "array", "items": Ref(name)} }

// Arr returns an array schema with inline item schema.
func Arr(items M) M { return M{"type": "array", "items": items} }

// Str returns a string schema with description.
func Str(desc string) M { return M{"type": "string", "description": desc} }

// Bool returns a boolean schema with description.
func Bool(desc string) M { return M{"type": "boolean", "description": desc} }

// Int returns an integer schema with description.
func Int(desc string) M { return M{"type": "integer", "description": desc} }

// Obj returns an object schema with description and no fixed properties.
func Obj(desc string) M { return M{"type": "object", "description": desc} }

// ObjMap returns an object schema with additionalProperties of the given type.
func ObjMap(valueType string) M {
	return M{"type": "object", "additionalProperties": M{"type": valueType}}
}

// Props returns an object schema with explicit properties.
func Props(required []string, props M) M {
	s := M{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = toS(required)
	}
	return s
}

// ── Parameter helpers ─────────────────────────────────────────────────────────

// PParam returns a required path parameter.
func PParam(name, desc string) Param {
	return Param{Name: name, In: "path", Required: true, Description: desc}
}

// QParam returns a query parameter.
func QParam(name, desc string, required bool) Param {
	return Param{Name: name, In: "query", Required: required, Description: desc}
}

// ── Request body helpers ──────────────────────────────────────────────────────

// JsonBody wraps a schema as an application/json required request body.
func JsonBody(schema M) M {
	return M{
		"required": true,
		"content":  M{"application/json": M{"schema": schema}},
	}
}

// ── Response helpers ──────────────────────────────────────────────────────────

// R merges any number of response maps into one map[string]M.
func R(maps ...map[string]M) map[string]M {
	out := map[string]M{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// OK returns a 200 response with a JSON body schema.
func OK(schema M) map[string]M {
	return map[string]M{"200": jsonResp("200 OK", schema)}
}

// Created returns a 201 response with a JSON body schema.
func Created(schema M) map[string]M {
	return map[string]M{"201": jsonResp("Created", schema)}
}

// NoContent returns a 204 No Content response.
func NoContent() map[string]M {
	return map[string]M{"204": {"description": "No content"}}
}

// NotFound returns a 404 Not Found response.
func NotFound() map[string]M {
	return map[string]M{"404": {"description": "Resource not found"}}
}

// BadRequest returns a 400 Bad Request response.
func BadRequest() map[string]M {
	return map[string]M{"400": {"description": "Bad request — invalid input"}}
}

// Accepted returns a 202 Accepted response with a JSON body schema.
func Accepted(schema M) map[string]M {
	return map[string]M{"202": jsonResp("Accepted — processing in background", schema)}
}

// ServerError returns a 500 Internal Server Error response.
func ServerError() map[string]M {
	return map[string]M{"500": {"description": "Internal server error"}}
}

// Unprocessable returns a 422 response (used for compile failures etc.).
func Unprocessable() map[string]M {
	return map[string]M{"422": {"description": "Processing failed — see error body"}}
}

func jsonResp(desc string, schema M) M {
	return M{
		"description": desc,
		"content":     M{"application/json": M{"schema": schema}},
	}
}

// ── Misc ──────────────────────────────────────────────────────────────────────

func toS(ss []string) S {
	out := make(S, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
