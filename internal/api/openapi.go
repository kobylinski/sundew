package api

import (
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

// schema derives object properties from the same JSON tags used by handlers.
func schema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return schema(t.Elem())
	}
	if t == reflect.TypeFor[time.Time]() {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.Struct:
		properties := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" || !f.IsExported() {
				continue
			}
			name := tag[0]
			if name == "" {
				name = f.Name
			}
			properties[name] = schema(f.Type)
			if len(tag) < 2 || tag[1] != "omitempty" {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": properties, "required": required}
	case reflect.Map:
		return map[string]any{"type": []string{"object", "null"}, "additionalProperties": schema(t.Elem())}
	case reflect.Slice:
		return map[string]any{"type": []string{"array", "null"}, "items": schema(t.Elem())}
	case reflect.Int, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	default:
		return map[string]any{"type": "string"}
	}
}
func ref(name string) map[string]any { return map[string]any{"$ref": "#/components/schemas/" + name} }
func response(description string, s any) map[string]any {
	r := map[string]any{"description": description}
	if s != nil {
		r["content"] = map[string]any{"application/json": map[string]any{"schema": s}}
	}
	return r
}
func (a *api) document() map[string]any {
	paths := map[string]any{}
	for _, e := range a.endpoints() {
		path, ok := paths[e.path].(map[string]any)
		if !ok {
			path = map[string]any{}
			paths[e.path] = path
		}
		responses := map[string]any{"400": response("Invalid input", ref("Error")), "404": response("Not found", ref("Error")), "405": response("Method not allowed", ref("Error")), "500": response("Operation failed", ref("Error"))}
		op := map[string]any{"summary": e.summary, "responses": responses}
		parameters := []any{}
		if strings.Contains(e.path, "{id}") {
			parameters = append(parameters, map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
		}
		if e.method == "GET" && (e.path == "/api/v1/messages" || strings.HasSuffix(e.path, "/latest") || strings.HasSuffix(e.path, "/stream")) {
			for _, name := range []string{"to", "from", "body", "account", "provider", "since", "limit", "cursor"} {
				s := map[string]any{"type": "string"}
				description := "Exact match"
				switch name {
				case "since":
					s["format"] = "date-time"
					description = "Inclusive lower bound on created_at (RFC 3339)"
				case "body":
					description = "Case-insensitive substring"
				case "limit":
					s = map[string]any{"type": "integer", "minimum": 1, "maximum": 500}
					description = "Page size (list only)"
				case "cursor":
					description = "Opaque next_cursor from the preceding page (list only)"
				}
				parameters = append(parameters, map[string]any{"name": name, "in": "query", "description": description, "schema": s})
			}
		}
		if len(parameters) > 0 {
			op["parameters"] = parameters
		}
		switch {
		case e.method == "DELETE" || e.path == "/api/v1/reset":
			responses["204"] = response("Store updated", nil)
		case e.path == "/api/v1/inbound":
			input := schema(reflect.TypeFor[inboundInput]())
			input["required"] = []string{}
			input["additionalProperties"] = false
			input["properties"].(map[string]any)["provider"].(map[string]any)["default"] = "twilio"
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": input}}}
			responses["201"] = response("Inbound webhook sent and message stored", map[string]any{"type": "object", "required": []string{"message", "response_status"}, "properties": map[string]any{"message": ref("Message"), "response_status": map[string]any{"type": "integer"}}})
		case strings.HasSuffix(e.path, "/stream"):
			responses["200"] = map[string]any{"description": "SSE event names: message.created, message.updated, message.deleted, store.reset. Each data field is a JSON Event. Reset is always emitted regardless of filters; heartbeat comments every 15 seconds. No replay: connect before sending; query the store to recover missed events.", "content": map[string]any{"text/event-stream": map[string]any{"schema": map[string]any{"type": "string"}}}}
		case e.path == "/api/v1/messages":
			responses["200"] = response("Page, newest first", map[string]any{"type": "object", "required": []string{"items", "next_cursor"}, "properties": map[string]any{"items": map[string]any{"type": "array", "items": ref("Message")}, "next_cursor": map[string]any{"type": "string"}}})
		case e.path == "/api/v1/openapi.json":
			responses["200"] = response("OpenAPI 3.1 document", map[string]any{"type": "object"})
		default:
			responses["200"] = response("Caught message", ref("Message"))
		}
		path[strings.ToLower(e.method)] = op
	}
	return map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "Sundew API for tests", "version": "1.0.0"}, "paths": paths, "components": map[string]any{"schemas": map[string]any{
		"Message": schema(reflect.TypeFor[core.Message]()), "Event": schema(reflect.TypeFor[core.Event]()),
		"Error": map[string]any{"type": "object", "required": []string{"error"}, "properties": map[string]any{"error": map[string]any{"type": "object", "required": []string{"code", "message"}, "properties": map[string]any{"code": map[string]any{"type": "string"}, "message": map[string]any{"type": "string"}}}}},
	}}}
}
func (a *api) openapi(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, a.document()) }
