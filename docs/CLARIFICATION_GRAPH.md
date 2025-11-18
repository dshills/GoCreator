# Clarification Workflow Graph Structure

## Node Definitions

### Node 1: Start
- **ID**: `start`
- **Type**: BasicNode
- **Dependencies**: None
- **Description**: Initialize clarification workflow
- **Function**: Validates that spec exists in state, sets workflow metadata

### Node 2: AnalyzeSpec
- **ID**: `analyze_spec`
- **Type**: BasicNode
- **Dependencies**: `start`
- **Description**: Analyze specification for ambiguities
- **Function**: Uses LLM to identify 5 types of ambiguities
  - Missing constraints
  - Conflicting requirements
  - Unclear specifications
  - Ambiguous terminology
  - Underspecified features

### Node 3: CheckAmbiguities
- **ID**: `check_ambiguities`
- **Type**: ConditionalNode
- **Dependencies**: `analyze_spec`
- **Description**: Check if ambiguities were found
- **Condition**: `len(ambiguities) > 0`
- **Function**: Sets `has_ambiguities` flag in state

### Node 4: GenerateQuestions
- **ID**: `generate_questions`
- **Type**: BasicNode
- **Dependencies**: `check_ambiguities`
- **Description**: Generate clarification questions from ambiguities
- **Function**: Uses LLM to generate 2-4 option questions with:
  - Clear topic and context
  - Specific options
  - Implications for each option

### Node 5: BuildFCS
- **ID**: `build_fcs`
- **Type**: BasicNode
- **Dependencies**: `generate_questions`
- **Description**: Build Final Clarified Specification
- **Function**: Constructs FCS from spec and answers with:
  - Original specification
  - Applied clarifications
  - Requirements
  - Architecture
  - Hash for integrity

### Node 6: End
- **ID**: `end`
- **Type**: BasicNode
- **Dependencies**: `build_fcs`
- **Description**: Complete clarification workflow
- **Function**: Sets completion flag in state

## State Flow

### Initial State
```json
{
  "spec": <InputSpecification>,
  "interactive": <boolean>
}
```

### After AnalyzeSpec
```json
{
  "spec": <InputSpecification>,
  "interactive": <boolean>,
  "ambiguities": [<Ambiguity>, ...],
  "ambiguity_count": <integer>
}
```

### After CheckAmbiguities
```json
{
  "spec": <InputSpecification>,
  "interactive": <boolean>,
  "ambiguities": [<Ambiguity>, ...],
  "ambiguity_count": <integer>,
  "has_ambiguities": <boolean>
}
```

### After GenerateQuestions
```json
{
  "spec": <InputSpecification>,
  "interactive": <boolean>,
  "ambiguities": [<Ambiguity>, ...],
  "ambiguity_count": <integer>,
  "has_ambiguities": <boolean>,
  "questions": [<Question>, ...],
  "question_count": <integer>
}
```

### After BuildFCS
```json
{
  "spec": <InputSpecification>,
  "interactive": <boolean>,
  "ambiguities": [<Ambiguity>, ...],
  "ambiguity_count": <integer>,
  "has_ambiguities": <boolean>,
  "questions": [<Question>, ...],
  "question_count": <integer>,
  "answers": {<question_id>: <Answer>, ...},
  "fcs": <FinalClarifiedSpecification>,
  "fcs_complete": <boolean>
}
```

### Final State (End)
```json
{
  "spec": <InputSpecification>,
  "interactive": <boolean>,
  "ambiguities": [<Ambiguity>, ...],
  "ambiguity_count": <integer>,
  "has_ambiguities": <boolean>,
  "questions": [<Question>, ...],
  "question_count": <integer>,
  "answers": {<question_id>: <Answer>, ...},
  "fcs": <FinalClarifiedSpecification>,
  "fcs_complete": <boolean>,
  "workflow_completed": <boolean>
}
```

## Execution Order (Topological Sort)

The graph is executed in the following batches based on dependencies:

### Batch 1 (Level 0)
- `start` - No dependencies

### Batch 2 (Level 1)
- `analyze_spec` - Depends on `start`

### Batch 3 (Level 2)
- `check_ambiguities` - Depends on `analyze_spec`

### Batch 4 (Level 3)
- `generate_questions` - Depends on `check_ambiguities`

### Batch 5 (Level 4)
- `build_fcs` - Depends on `generate_questions`

### Batch 6 (Level 5)
- `end` - Depends on `build_fcs`

## LLM Interactions

### Interaction 1: Ambiguity Analysis
**Node**: `analyze_spec`
**Temperature**: 0.0 (deterministic)
**Input**: Specification content
**Output**: JSON array of ambiguities
```json
[
  {
    "type": "missing_constraint",
    "location": "requirements.FR-003",
    "description": "No upper bound for concurrent users",
    "severity": "critical"
  }
]
```

### Interaction 2: Question Generation
**Node**: `generate_questions`
**Temperature**: 0.0 (deterministic)
**Input**: Array of ambiguities
**Output**: JSON array of questions
```json
[
  {
    "topic": "Concurrency Limits",
    "context": "Requirement FR-003 mentions concurrent users...",
    "question": "What is the maximum number of concurrent users?",
    "options": [
      {
        "label": "100 concurrent users",
        "description": "Small-scale deployment",
        "implications": "Simpler architecture"
      }
    ]
  }
]
```

## Error Handling

### Node Failures
Each node can fail with specific errors:

1. **start**: "input specification not found in state"
2. **analyze_spec**: "analysis failed: <LLM error>"
3. **check_ambiguities**: Always succeeds (sets flag)
4. **generate_questions**: "question generation failed: <LLM error>"
5. **build_fcs**: "FCS construction failed: <validation error>"
6. **end**: Always succeeds (sets flag)

### Recovery Strategy
If a node fails:
1. Error is propagated up
2. Execution stops immediately
3. State is preserved in checkpoint (if enabled)
4. Can be resumed from last successful checkpoint

## Performance Characteristics

### Time Complexity
- Graph validation: O(V + E) where V = nodes, E = edges
- Topological sort: O(V + E)
- Execution: O(V) with LLM latency dominant

### Space Complexity
- State storage: O(S) where S = state size
- Checkpoint storage: O(S × C) where C = number of checkpoints
- Memory usage: O(V + S)

## Checkpointing Strategy

Checkpoints are saved after each batch completion:

1. **After start**: Initial state checkpoint
2. **After analyze_spec**: Ambiguities identified
3. **After check_ambiguities**: Decision point saved
4. **After generate_questions**: Questions available for review
5. **After build_fcs**: FCS ready
6. **After end**: Final state

Each checkpoint contains:
- Graph ID
- Last completed node
- Complete state snapshot
- Completed nodes list
- Timestamp
- Recoverable flag

## Conditional Branching

The `check_ambiguities` node implements conditional logic:

```go
condition := func(state State) bool {
    val, ok := state.Get("ambiguities")
    if !ok {
        return false
    }
    ambiguities, ok := val.([]models.Ambiguity)
    if !ok {
        return false
    }
    return len(ambiguities) > 0
}
```

If condition returns false, subsequent nodes still execute but with empty question list.

## Integration with GoCreator Pipeline

The clarification graph is the first stage in the GoCreator pipeline:

```
Input Spec → [Clarification Graph] → FCS → Generation Graph → Code Output
                     ↓
              (This Implementation)
```

Future graphs will follow the same LangGraph-Go pattern:
- Generation graph for code creation
- Validation graph for quality assurance
- Regeneration graph for iterative refinement
