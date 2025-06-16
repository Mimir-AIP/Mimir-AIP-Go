# Pipeline Control Flow Extensions

## 1. Proposed Schema Extensions

### Conditional Blocks
```json
"condition": {
  "type": "object",
  "properties": {
    "expression": {"type": "string"},
    "on_true": {"enum": ["continue", "jump"]},
    "on_false": {"enum": ["continue", "jump"]},
    "jump_target": {"type": "string"}
  },
  "required": ["expression"]
}
```

### Loop Constructs
```json
"loop": {
  "type": "object",
  "properties": {
    "items": {"type": "string"},
    "as": {"type": "string"},
    "index": {"type": "string"}
  },
  "required": ["items", "as"]
}
```

### Step Labels
```json
"label": {"type": "string"}
```

## 2. Parser Modifications

- Add new fields to `StepNode` in `ast_nodes.py`
- Implement condition evaluation in AST traversal
- Add jump target resolution during AST construction
- Handle loop iteration with context scoping

## 3. Backward Compatibility

- Maintain all existing required fields
- New fields optional with default behaviors
- Old pipelines validate without changes
- Dual support for `iterate` (deprecated) and `loop`

## 4. Implementation Steps

1. Update `pipeline_schema.json` with new fields
2. Extend `StepNode` in `ast_nodes.py`
3. Enhance parser logic in `pipeline_parser.py`:
   - Conditional execution
   - Loop handling
   - Jump resolution
4. Update execution engine with control flow logic
5. Add comprehensive test cases

```mermaid
graph TD
    A[Start] --> B[Parse Pipeline]
    B --> C{Has Condition?}
    C -->|Yes| D[Evaluate Expression]
    D --> E{Condition Met?}
    E -->|True| F[Execute Step]
    E -->|False| G{Action Type?}
    G -->|Continue| H[Skip Step]
    G -->|Jump| I[Find Label]
    I --> J[Resume Execution]
    C -->|No| K{Has Loop?}
    K -->|Yes| L[Get Iterable]
    L --> M[For Each Item]
    M --> N[Set Context Vars]
    N --> O[Execute Nested Steps]
    O --> P[Next Item]
    K -->|No| Q[Execute Step]