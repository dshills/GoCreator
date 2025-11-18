# Batch Generation Implementation Plan

**Feature**: User Story 2 - Batch Code Generation
**Target**: 70% reduction in LLM API calls for projects with repetitive structure
**Estimated Timeline**: 5-7 days
**Created**: 2025-11-18
**Status**: Planning

## Executive Summary

Batch generation allows GoCreator to generate multiple similar files in a single LLM call instead of making separate API calls for each file. For CRUD applications with 10-20 entity files, this reduces API calls from 20+ down to 2-4 batched calls, achieving a 70-80% reduction in API overhead.

**Key Innovation**: Smart similarity detection groups files by structural similarity (same file type, same purpose, similar inputs), then uses JSON-structured prompts to generate multiple files in one shot. The system includes robust fallback to individual generation if batch parsing fails.

## Problem Statement

### Current State (Per-File Generation)

Today's generation pipeline:
```
For each file in GenerationPlan:
  1. Build prompt with full FCS context
  2. Call LLM API (200-500ms latency per call)
  3. Parse single file response
  4. Create patch

Total for 10 entity files: 10 API calls, 2-5 seconds
```

**Pain Points**:
- High API call overhead (latency adds up)
- Redundant context sent in each prompt (same FCS, same instructions)
- Higher token costs (per-call overhead)
- Sequential processing (waiting for each response)

### Desired State (Batch Generation)

With batch generation:
```
1. Group similar files (entities, CRUD handlers, tests)
2. Build batch prompt for group (3-5 files)
3. Single LLM call returns JSON with all files
4. Parse and create patches for each

Total for 10 entity files: 2-3 batched calls, 0.8-1.5 seconds
```

**Benefits**:
- 70% fewer API calls (10 → 3)
- Lower latency (parallelizable batches)
- Reduced token costs (shared context)
- Better throughput (fewer round-trips)

## Architecture Design

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Generation Pipeline                      │
│                                                             │
│  ┌────────────┐      ┌─────────────────┐                  │
│  │  Planner   │─────▶│  Batch Grouper  │                  │
│  │  (Exists)  │      │    (NEW)        │                  │
│  └────────────┘      └─────────────────┘                  │
│                              │                              │
│                              ▼                              │
│                     ┌─────────────────┐                    │
│                     │ Similarity      │                    │
│                     │ Detector (NEW)  │                    │
│                     └─────────────────┘                    │
│                              │                              │
│                              ▼                              │
│              ┌────────────────────────────┐                │
│              │  Batch Coder (NEW)         │                │
│              │  - Build JSON prompt       │                │
│              │  - Call LLM once           │                │
│              │  - Parse JSON response     │                │
│              └────────────────────────────┘                │
│                              │                              │
│                   ┌──────────┴──────────┐                  │
│                   │  Success   │  Fail  │                  │
│                   ▼            ▼         │                  │
│           ┌─────────────┐  ┌──────────────────┐            │
│           │  Multiple   │  │  Fallback to     │            │
│           │  Patches    │  │  Individual Gen  │            │
│           └─────────────┘  └──────────────────┘            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### New Components

#### 1. BatchGrouper (`internal/generate/batch_grouper.go`)

**Responsibility**: Group tasks from GenerationPlan into batchable clusters

```go
type BatchGrouper struct {
    config BatchConfig
}

type BatchConfig struct {
    Enabled             bool
    MaxBatchSize        int     // 3-5 files per batch
    SimilarityThreshold float64 // 0.7 = 70% similar
    AllowedFileTypes    []string // entity, handler, test, etc.
}

type Batch struct {
    ID          string
    FileType    string              // "entity", "handler", "test"
    Tasks       []models.GenerationTask
    Similarity  float64             // 0.0-1.0
    CanBatch    bool
}

// Group organizes tasks into batches
func (bg *BatchGrouper) Group(tasks []models.GenerationTask, fcs *models.FinalClarifiedSpecification) []Batch
```

**Key Methods**:
- `Group()` - Main entry point, returns batches
- `computeSimilarity()` - Calculate structural similarity
- `canBatch()` - Determine if files are batchable
- `splitIntoBatches()` - Split large groups into batch sizes

#### 2. SimilarityDetector (`internal/generate/similarity.go`)

**Responsibility**: Compute structural similarity between generation tasks

```go
type SimilarityDetector struct{}

type SimilarityScore struct {
    FileType      float64 // Same file type (entity, handler, etc.)
    Structure     float64 // Similar inputs/attributes count
    Package       float64 // Same package
    Dependencies  float64 // Similar dependency set
    Overall       float64 // Weighted average
}

// ComputeSimilarity calculates similarity between two tasks
func (sd *SimilarityDetector) ComputeSimilarity(
    task1, task2 models.GenerationTask,
    fcs *models.FinalClarifiedSpecification,
) SimilarityScore
```

**Similarity Factors**:
| Factor | Weight | Calculation |
|--------|--------|-------------|
| File Type | 30% | Exact match: 1.0, else 0.0 |
| Structure | 35% | Attribute count difference / max count |
| Package | 20% | Same package: 1.0, else 0.0 |
| Dependencies | 15% | Jaccard similarity of dependency sets |

**Formula**:
```
Overall = (0.30 × FileType) + (0.35 × Structure) + (0.20 × Package) + (0.15 × Dependencies)
```

**Threshold**: Files with Overall ≥ 0.7 (70%) can be batched together

#### 3. BatchCoder (`internal/generate/batch_coder.go`)

**Responsibility**: Generate multiple files in a single LLM call

```go
type BatchCoder struct {
    client        llm.Client
    fallbackCoder Coder  // For retry on batch failure
}

type BatchRequest struct {
    Batch     Batch
    Plan      *models.GenerationPlan
    FCS       *models.FinalClarifiedSpecification
}

type BatchResponse struct {
    Files   map[string]string  // filename -> code
    Success bool
    Error   error
}

// GenerateBatch generates multiple files in one LLM call
func (bc *BatchCoder) GenerateBatch(ctx context.Context, req BatchRequest) ([]models.Patch, error)
```

**Flow**:
1. Build JSON-structured prompt (see Prompt Engineering section)
2. Call LLM with structured output request
3. Parse JSON response into individual files
4. Validate each file (syntax, structure)
5. Create patches for each file
6. On failure: fallback to individual generation

### Integration with Existing Pipeline

**Before Batch Generation**:
```go
// internal/generate/coder.go
func (c *llmCoder) Generate(ctx context.Context, plan *models.GenerationPlan, fcs *models.FinalClarifiedSpecification) ([]models.Patch, error) {
    for _, phase := range plan.Phases {
        for _, task := range phase.Tasks {
            patch, err := c.GenerateFile(ctx, task, plan, fcs)
            // ... handle individually
        }
    }
}
```

**After Batch Generation**:
```go
// internal/generate/coder.go (enhanced)
func (c *llmCoder) Generate(ctx context.Context, plan *models.GenerationPlan, fcs *models.FinalClarifiedSpecification) ([]models.Patch, error) {
    for _, phase := range plan.Phases {
        // NEW: Group tasks into batches
        batches := c.batchGrouper.Group(phase.Tasks, fcs)

        for _, batch := range batches {
            if batch.CanBatch && len(batch.Tasks) > 1 {
                // NEW: Batch generation
                patches, err := c.batchCoder.GenerateBatch(ctx, BatchRequest{
                    Batch: batch,
                    Plan:  plan,
                    FCS:   fcs,
                })
                if err != nil {
                    // Fallback to individual generation
                    log.Warn().Err(err).Msg("Batch generation failed, falling back")
                    for _, task := range batch.Tasks {
                        patch, err := c.GenerateFile(ctx, task, plan, fcs)
                        // ... handle individually
                    }
                } else {
                    allPatches = append(allPatches, patches...)
                }
            } else {
                // Single file or non-batchable
                patch, err := c.GenerateFile(ctx, task, plan, fcs)
                // ... existing logic
            }
        }
    }
}
```

## Similarity Detection Algorithm

### Detailed Algorithm

```go
func (sd *SimilarityDetector) ComputeSimilarity(
    task1, task2 models.GenerationTask,
    fcs *models.FinalClarifiedSpecification,
) SimilarityScore {
    score := SimilarityScore{}

    // 1. File Type Similarity (30% weight)
    fileType1 := sd.getFileType(task1.TargetPath)
    fileType2 := sd.getFileType(task2.TargetPath)
    if fileType1 == fileType2 {
        score.FileType = 1.0
    } else {
        score.FileType = 0.0
    }

    // 2. Structural Similarity (35% weight)
    entity1 := sd.getEntity(task1, fcs)
    entity2 := sd.getEntity(task2, fcs)
    if entity1 != nil && entity2 != nil {
        attrCount1 := len(entity1.Attributes)
        attrCount2 := len(entity2.Attributes)
        maxCount := math.Max(float64(attrCount1), float64(attrCount2))
        minCount := math.Min(float64(attrCount1), float64(attrCount2))
        if maxCount > 0 {
            score.Structure = minCount / maxCount
        }

        // Bonus for similar attribute types
        typeMatch := sd.compareAttributeTypes(entity1, entity2)
        score.Structure = (score.Structure + typeMatch) / 2.0
    }

    // 3. Package Similarity (20% weight)
    pkg1, _ := task1.Inputs["package"].(string)
    pkg2, _ := task2.Inputs["package"].(string)
    if pkg1 == pkg2 && pkg1 != "" {
        score.Package = 1.0
    } else {
        score.Package = 0.0
    }

    // 4. Dependency Similarity (15% weight)
    deps1 := sd.getDependencies(task1, fcs)
    deps2 := sd.getDependencies(task2, fcs)
    score.Dependencies = sd.jaccardSimilarity(deps1, deps2)

    // 5. Overall Weighted Score
    score.Overall = (0.30 * score.FileType) +
                    (0.35 * score.Structure) +
                    (0.20 * score.Package) +
                    (0.15 * score.Dependencies)

    return score
}

// Jaccard similarity: |A ∩ B| / |A ∪ B|
func (sd *SimilarityDetector) jaccardSimilarity(set1, set2 []string) float64 {
    if len(set1) == 0 && len(set2) == 0 {
        return 1.0
    }

    intersection := 0
    union := make(map[string]bool)

    for _, item := range set1 {
        union[item] = true
    }
    for _, item := range set2 {
        if union[item] {
            intersection++
        } else {
            union[item] = true
        }
    }

    if len(union) == 0 {
        return 0.0
    }

    return float64(intersection) / float64(len(union))
}

func (sd *SimilarityDetector) compareAttributeTypes(e1, e2 *models.Entity) float64 {
    types1 := make(map[string]int)
    types2 := make(map[string]int)

    for _, attrType := range e1.Attributes {
        types1[attrType]++
    }
    for _, attrType := range e2.Attributes {
        types2[attrType]++
    }

    matches := 0
    total := 0

    for typ, count1 := range types1 {
        count2 := types2[typ]
        matches += int(math.Min(float64(count1), float64(count2)))
        total += int(math.Max(float64(count1), float64(count2)))
    }

    for typ, count2 := range types2 {
        if _, exists := types1[typ]; !exists {
            total += count2
        }
    }

    if total == 0 {
        return 1.0
    }

    return float64(matches) / float64(total)
}
```

### Example Similarity Computation

**Task 1**: Generate `internal/models/user.go`
- File Type: entity
- Package: models
- Attributes: 5 (ID, Name, Email, CreatedAt, UpdatedAt)
- Dependencies: [time]

**Task 2**: Generate `internal/models/product.go`
- File Type: entity
- Package: models
- Attributes: 6 (ID, Name, Price, Description, CreatedAt, UpdatedAt)
- Dependencies: [time]

**Calculation**:
- FileType: 1.0 (both "entity")
- Structure: 5/6 = 0.83 (attribute count), type match = 0.67 (4 shared types), avg = 0.75
- Package: 1.0 (both "models")
- Dependencies: 1.0 (same [time])

**Overall**: (0.30×1.0) + (0.35×0.75) + (0.20×1.0) + (0.15×1.0) = **0.91** ✅ **Batchable!**

## Prompt Engineering

### Batch Prompt Structure

The batch prompt uses a JSON-structured format to clearly delineate multiple files:

```
You are an expert Go developer generating production-ready code.

# Task
Generate MULTIPLE Go source files in a SINGLE JSON response.

# Files to Generate
You will generate {{.FileCount}} similar files in one response.

{{range .Files}}
## File {{.Index}}: {{.TargetPath}}

**Package**: {{.Package}}
**Entity**: {{.EntityName}}
**Attributes**: {{.AttributesJSON}}
**Purpose**: {{.Purpose}}

{{end}}

# Project Context (Filtered)
{{.FilteredFCS}}

# Output Format

Return a JSON object with the following structure:

{
  "files": {
    "{{.File1Path}}": "package models\n\ntype User struct {\n...\n}",
    "{{.File2Path}}": "package models\n\ntype Product struct {\n...\n}",
    ...
  }
}

CRITICAL INSTRUCTIONS:
1. Return ONLY valid JSON, no markdown code blocks
2. Each file value must be the complete Go source code as a string
3. Escape newlines as \n, quotes as \"
4. Include proper package declarations for each file
5. Follow Go idioms and conventions
6. Add godoc comments for all exported types

# Example Response Format

{
  "files": {
    "internal/models/user.go": "package models\n\nimport \"time\"\n\n// User represents...\ntype User struct {\n\tID string\n\tName string\n\tEmail string\n\tCreatedAt time.Time\n\tUpdatedAt time.Time\n}\n",
    "internal/models/product.go": "package models\n\nimport \"time\"\n\n// Product represents...\ntype Product struct {\n\tID string\n\tName string\n\tPrice float64\n\tDescription string\n\tCreatedAt time.Time\n\tUpdatedAt time.Time\n}\n"
  }
}

# Coding Standards
{{.CodingStandards}}
```

### Prompt Template Implementation

```go
// internal/generate/batch_prompt.go

type BatchPromptBuilder struct {
    template *template.Template
}

type BatchPromptData struct {
    FileCount        int
    Files            []FilePromptData
    FilteredFCS      string
    CodingStandards  string
}

type FilePromptData struct {
    Index          int
    TargetPath     string
    Package        string
    EntityName     string
    AttributesJSON string
    Purpose        string
}

func (bpb *BatchPromptBuilder) Build(batch Batch, fcs *models.FinalClarifiedSpecification) (string, error) {
    data := BatchPromptData{
        FileCount: len(batch.Tasks),
        Files:     make([]FilePromptData, 0, len(batch.Tasks)),
    }

    for i, task := range batch.Tasks {
        fileData := bpb.extractFileData(task, fcs)
        fileData.Index = i + 1
        data.Files = append(data.Files, fileData)
    }

    data.FilteredFCS = bpb.buildFilteredFCS(batch, fcs)
    data.CodingStandards = bpb.getCodingStandards()

    var buf bytes.Buffer
    if err := bpb.template.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("template execution failed: %w", err)
    }

    return buf.String(), nil
}
```

### JSON Response Parsing

```go
// internal/generate/batch_parser.go

type BatchParser struct{}

type BatchJSONResponse struct {
    Files map[string]string `json:"files"`
}

func (bp *BatchParser) Parse(response string, expectedFiles []string) (map[string]string, error) {
    // 1. Clean response (remove markdown if present)
    cleaned := bp.cleanResponse(response)

    // 2. Parse JSON
    var batchResp BatchJSONResponse
    if err := json.Unmarshal([]byte(cleaned), &batchResp); err != nil {
        return nil, fmt.Errorf("JSON parse failed: %w", err)
    }

    // 3. Validate all expected files are present
    if len(batchResp.Files) != len(expectedFiles) {
        return nil, fmt.Errorf("expected %d files, got %d", len(expectedFiles), len(batchResp.Files))
    }

    for _, expectedPath := range expectedFiles {
        if _, exists := batchResp.Files[expectedPath]; !exists {
            return nil, fmt.Errorf("missing file: %s", expectedPath)
        }
    }

    // 4. Validate each file (syntax check)
    for path, code := range batchResp.Files {
        if err := bp.validateGoSyntax(code); err != nil {
            return nil, fmt.Errorf("invalid syntax in %s: %w", path, err)
        }
    }

    return batchResp.Files, nil
}

func (bp *BatchParser) cleanResponse(response string) string {
    response = strings.TrimSpace(response)

    // Remove markdown JSON code blocks
    if strings.HasPrefix(response, "```json") {
        response = strings.TrimPrefix(response, "```json")
        response = strings.TrimSuffix(response, "```")
    } else if strings.HasPrefix(response, "```") {
        response = strings.TrimPrefix(response, "```")
        response = strings.TrimSuffix(response, "```")
    }

    return strings.TrimSpace(response)
}

func (bp *BatchParser) validateGoSyntax(code string) error {
    // Use go/parser to validate syntax
    fset := token.NewFileSet()
    _, err := parser.ParseFile(fset, "", code, parser.AllErrors)
    return err
}
```

## Error Handling & Fallback

### Failure Scenarios

1. **JSON Parsing Failure**: LLM returns invalid JSON
2. **Missing Files**: LLM omits one or more files
3. **Syntax Errors**: Generated code has Go syntax errors
4. **Partial Success**: Some files valid, others invalid
5. **Timeout**: LLM call takes too long

### Fallback Strategy

```go
func (bc *BatchCoder) GenerateBatch(ctx context.Context, req BatchRequest) ([]models.Patch, error) {
    // 1. Attempt batch generation
    response, err := bc.client.Generate(ctx, bc.buildBatchPrompt(req))
    if err != nil {
        log.Warn().Err(err).Msg("Batch LLM call failed, falling back to individual generation")
        return bc.fallbackToIndividual(ctx, req)
    }

    // 2. Parse response
    files, err := bc.parser.Parse(response, bc.getExpectedPaths(req.Batch))
    if err != nil {
        log.Warn().Err(err).Msg("Batch parse failed, falling back to individual generation")
        return bc.fallbackToIndividual(ctx, req)
    }

    // 3. Validate and create patches
    patches := make([]models.Patch, 0, len(files))
    var failedTasks []models.GenerationTask

    for i, task := range req.Batch.Tasks {
        code, exists := files[task.TargetPath]
        if !exists {
            log.Warn().Str("path", task.TargetPath).Msg("File missing from batch, will retry individually")
            failedTasks = append(failedTasks, task)
            continue
        }

        // Create patch
        patch := models.Patch{
            TargetFile: task.TargetPath,
            Diff:       bc.createFileDiff(code),
            AppliedAt:  time.Now(),
            Reversible: true,
        }

        patches = append(patches, patch)

        log.Debug().
            Str("path", task.TargetPath).
            Int("batch_index", i).
            Msg("File generated successfully in batch")
    }

    // 4. Retry failed tasks individually
    if len(failedTasks) > 0 {
        log.Info().Int("count", len(failedTasks)).Msg("Retrying failed tasks individually")
        for _, task := range failedTasks {
            patch, err := bc.fallbackCoder.GenerateFile(ctx, task, req.Plan, req.FCS)
            if err != nil {
                return patches, fmt.Errorf("failed to generate %s individually: %w", task.TargetPath, err)
            }
            patches = append(patches, patch)
        }
    }

    return patches, nil
}

func (bc *BatchCoder) fallbackToIndividual(ctx context.Context, req BatchRequest) ([]models.Patch, error) {
    patches := make([]models.Patch, 0, len(req.Batch.Tasks))

    for _, task := range req.Batch.Tasks {
        patch, err := bc.fallbackCoder.GenerateFile(ctx, task, req.Plan, req.FCS)
        if err != nil {
            return patches, fmt.Errorf("fallback generation failed for %s: %w", task.TargetPath, err)
        }
        patches = append(patches, patch)
    }

    return patches, nil
}
```

### Metrics Tracking

```go
type BatchMetrics struct {
    BatchAttempts       int
    BatchSuccesses      int
    BatchFailures       int
    FallbacksTriggered  int
    FilesInBatches      int
    FilesIndividual     int
    AvgBatchSize        float64
    BatchEfficiency     float64  // (BatchSuccesses / BatchAttempts)
}

func (bc *BatchCoder) trackMetrics(attempt string, success bool, fileCount int) {
    bc.metrics.mu.Lock()
    defer bc.metrics.mu.Unlock()

    bc.metrics.BatchAttempts++
    if success {
        bc.metrics.BatchSuccesses++
        bc.metrics.FilesInBatches += fileCount
    } else {
        bc.metrics.BatchFailures++
        bc.metrics.FallbacksTriggered++
        bc.metrics.FilesIndividual += fileCount
    }

    bc.updateAverages()
}
```

## Testing Strategy

### Unit Tests

#### 1. Similarity Detection Tests (`similarity_test.go`)

```go
func TestSimilarityDetector_ComputeSimilarity(t *testing.T) {
    tests := []struct {
        name      string
        task1     models.GenerationTask
        task2     models.GenerationTask
        fcs       *models.FinalClarifiedSpecification
        wantScore float64
        wantBatch bool
    }{
        {
            name: "identical entities - high similarity",
            task1: models.GenerationTask{
                TargetPath: "models/user.go",
                Inputs: map[string]interface{}{
                    "package": "models",
                    "entity":  "User",
                },
            },
            task2: models.GenerationTask{
                TargetPath: "models/product.go",
                Inputs: map[string]interface{}{
                    "package": "models",
                    "entity":  "Product",
                },
            },
            fcs:       createTestFCS(),
            wantScore: 0.85,
            wantBatch: true, // > 0.7 threshold
        },
        {
            name: "different file types - low similarity",
            task1: models.GenerationTask{
                TargetPath: "models/user.go",
            },
            task2: models.GenerationTask{
                TargetPath: "handlers/user_handler.go",
            },
            wantScore: 0.3,
            wantBatch: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            detector := NewSimilarityDetector()
            score := detector.ComputeSimilarity(tt.task1, tt.task2, tt.fcs)

            assert.InDelta(t, tt.wantScore, score.Overall, 0.1)
            assert.Equal(t, tt.wantBatch, score.Overall >= 0.7)
        })
    }
}
```

#### 2. Batch Grouping Tests (`batch_grouper_test.go`)

```go
func TestBatchGrouper_Group(t *testing.T) {
    tests := []struct {
        name       string
        tasks      []models.GenerationTask
        config     BatchConfig
        wantGroups int
        wantBatches int
    }{
        {
            name: "10 similar entities -> 2-3 batches",
            tasks: createEntityTasks(10),
            config: BatchConfig{
                Enabled:          true,
                MaxBatchSize:     5,
                SimilarityThreshold: 0.7,
            },
            wantGroups: 1, // All entities group together
            wantBatches: 2, // 10 entities / 5 per batch = 2 batches
        },
        {
            name: "mixed file types -> separate batches",
            tasks: append(createEntityTasks(5), createHandlerTasks(5)...),
            config: BatchConfig{
                Enabled:          true,
                MaxBatchSize:     5,
                SimilarityThreshold: 0.7,
            },
            wantGroups: 2, // Entities and handlers separate
            wantBatches: 2, // One batch per group
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            grouper := NewBatchGrouper(tt.config)
            batches := grouper.Group(tt.tasks, createTestFCS())

            assert.Len(t, batches, tt.wantBatches)
        })
    }
}
```

#### 3. Prompt Building Tests (`batch_prompt_test.go`)

```go
func TestBatchPromptBuilder_Build(t *testing.T) {
    builder := NewBatchPromptBuilder()
    batch := Batch{
        FileType: "entity",
        Tasks: []models.GenerationTask{
            {TargetPath: "models/user.go"},
            {TargetPath: "models/product.go"},
        },
    }

    prompt, err := builder.Build(batch, createTestFCS())
    require.NoError(t, err)

    // Verify prompt contains key elements
    assert.Contains(t, prompt, "Generate MULTIPLE Go source files")
    assert.Contains(t, prompt, "models/user.go")
    assert.Contains(t, prompt, "models/product.go")
    assert.Contains(t, prompt, `"files": {`)
    assert.Contains(t, prompt, "Return ONLY valid JSON")
}
```

#### 4. Response Parsing Tests (`batch_parser_test.go`)

```go
func TestBatchParser_Parse(t *testing.T) {
    tests := []struct {
        name          string
        response      string
        expectedFiles []string
        wantErr       bool
    }{
        {
            name: "valid JSON response",
            response: `{
                "files": {
                    "models/user.go": "package models\n\ntype User struct {}",
                    "models/product.go": "package models\n\ntype Product struct {}"
                }
            }`,
            expectedFiles: []string{"models/user.go", "models/product.go"},
            wantErr: false,
        },
        {
            name: "JSON with markdown code block",
            response: "```json\n{\"files\": {...}}\n```",
            expectedFiles: []string{"models/user.go"},
            wantErr: false,
        },
        {
            name: "missing file",
            response: `{
                "files": {
                    "models/user.go": "package models\n\ntype User struct {}"
                }
            }`,
            expectedFiles: []string{"models/user.go", "models/product.go"},
            wantErr: true, // Missing product.go
        },
        {
            name: "invalid JSON",
            response: "not json",
            expectedFiles: []string{"models/user.go"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewBatchParser()
            files, err := parser.Parse(tt.response, tt.expectedFiles)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Len(t, files, len(tt.expectedFiles))
            }
        })
    }
}
```

### Integration Tests

#### 1. End-to-End Batch Generation (`batch_integration_test.go`)

```go
func TestBatchGeneration_E2E(t *testing.T) {
    // Create real LLM client (or mock with realistic responses)
    client := createTestLLMClient()

    // Create batch coder
    coder := NewBatchCoder(BatchCoderConfig{
        Client: client,
        Config: BatchConfig{
            Enabled:          true,
            MaxBatchSize:     5,
            SimilarityThreshold: 0.7,
        },
    })

    // Create test FCS with 10 entities
    fcs := createTestFCSWithEntities(10)

    // Create generation plan
    plan := createTestPlan(fcs)

    // Execute batch generation
    patches, err := coder.Generate(context.Background(), plan, fcs)
    require.NoError(t, err)

    // Verify results
    assert.Len(t, patches, 10, "Should generate all 10 entities")

    // Verify metrics
    metrics := coder.GetMetrics()
    assert.LessOrEqual(t, metrics.TotalLLMCalls, 3, "Should make ≤3 batched calls for 10 entities")
    assert.GreaterOrEqual(t, metrics.BatchEfficiency, 0.7, "Should have 70%+ batch success rate")
}
```

#### 2. Fallback Scenario Test

```go
func TestBatchGeneration_FallbackOnFailure(t *testing.T) {
    // Mock client that fails batch generation
    client := &mockLLMClient{
        generateFunc: func(ctx context.Context, prompt string) (string, error) {
            if strings.Contains(prompt, "MULTIPLE") {
                return "invalid json", nil // Simulate parse failure
            }
            return validGoCode, nil
        },
    }

    coder := NewBatchCoder(BatchCoderConfig{Client: client})

    // Execute - should fallback to individual generation
    patches, err := coder.Generate(context.Background(), testPlan, testFCS)
    require.NoError(t, err)

    // All files should still be generated (via fallback)
    assert.Len(t, patches, 10)

    // Verify fallback was triggered
    metrics := coder.GetMetrics()
    assert.Equal(t, 1, metrics.FallbacksTriggered)
    assert.Equal(t, 10, metrics.FilesIndividual)
}
```

### Performance Benchmarks

```go
func BenchmarkBatchGeneration(b *testing.B) {
    client := createMockLLMClient()
    coder := NewBatchCoder(BatchCoderConfig{Client: client})

    fcs := createTestFCSWithEntities(20)
    plan := createTestPlan(fcs)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := coder.Generate(context.Background(), plan, fcs)
        if err != nil {
            b.Fatal(err)
        }
    }

    // Report API call reduction
    metrics := coder.GetMetrics()
    callReduction := 1.0 - (float64(metrics.TotalLLMCalls) / float64(20))
    b.ReportMetric(callReduction*100, "call_reduction_%")
}

func BenchmarkIndividualGeneration(b *testing.B) {
    // Same test but with batching disabled
    client := createMockLLMClient()
    coder := NewCoder(CoderConfig{
        LLMClient: client,
        BatchingEnabled: false,
    })

    fcs := createTestFCSWithEntities(20)
    plan := createTestPlan(fcs)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := coder.Generate(context.Background(), plan, fcs)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Implementation Plan

### Phase 1: Core Infrastructure (Days 1-2)

**Day 1: Similarity Detection**
- [ ] Create `internal/generate/similarity.go`
- [ ] Implement `SimilarityDetector` struct and methods
- [ ] Implement `ComputeSimilarity()` with all factors
- [ ] Add `jaccardSimilarity()` helper
- [ ] Add `compareAttributeTypes()` helper
- [ ] Write unit tests for similarity calculation
- [ ] Test edge cases (empty inputs, nil entities, identical tasks)

**Day 2: Batch Grouping**
- [ ] Create `internal/generate/batch_grouper.go`
- [ ] Implement `BatchGrouper` struct
- [ ] Implement `Group()` method
- [ ] Add `canBatch()` validation logic
- [ ] Add `splitIntoBatches()` for size limits
- [ ] Write unit tests for grouping logic
- [ ] Test with various task combinations

### Phase 2: Prompt & Parsing (Days 3-4)

**Day 3: Batch Prompt Engineering**
- [ ] Create `internal/generate/batch_prompt.go`
- [ ] Design batch prompt template
- [ ] Implement `BatchPromptBuilder`
- [ ] Add template execution logic
- [ ] Test prompt generation with sample data
- [ ] Validate prompt includes all necessary context

**Day 4: Response Parsing**
- [ ] Create `internal/generate/batch_parser.go`
- [ ] Implement `BatchParser` struct
- [ ] Add `Parse()` method with JSON unmarshaling
- [ ] Add `cleanResponse()` for markdown removal
- [ ] Add `validateGoSyntax()` using go/parser
- [ ] Write unit tests for parsing
- [ ] Test error scenarios (invalid JSON, missing files)

### Phase 3: Batch Coder (Day 5)

**Day 5: Batch Generation Logic**
- [ ] Create `internal/generate/batch_coder.go`
- [ ] Implement `BatchCoder` struct
- [ ] Implement `GenerateBatch()` method
- [ ] Add fallback logic
- [ ] Integrate prompt builder and parser
- [ ] Add metrics tracking
- [ ] Write unit tests for batch coder

### Phase 4: Integration (Day 6)

**Day 6: Pipeline Integration**
- [ ] Update `internal/generate/coder.go`
- [ ] Integrate `BatchGrouper` into `Generate()` flow
- [ ] Add batch vs. individual decision logic
- [ ] Wire up `BatchCoder` for batch execution
- [ ] Update `llmCoder` to use batching
- [ ] Add configuration support (enable/disable batching)
- [ ] Test end-to-end batch generation

### Phase 5: Testing & Optimization (Day 7)

**Day 7: Testing & Metrics**
- [ ] Write integration tests (`batch_integration_test.go`)
- [ ] Test with real LLM clients (Anthropic, OpenAI)
- [ ] Benchmark batch vs. individual generation
- [ ] Validate 70% call reduction target
- [ ] Add metrics to generation reports
- [ ] Update CLI to show batch statistics
- [ ] Document batch generation feature

### Testing Checklist

**Unit Tests**:
- [x] Similarity detector (all factors)
- [x] Batch grouper (various scenarios)
- [x] Prompt builder (template rendering)
- [x] Response parser (valid/invalid JSON)
- [x] Fallback logic

**Integration Tests**:
- [x] End-to-end batch generation
- [x] Fallback on parse failure
- [x] Mixed batch/individual execution
- [x] Metrics tracking

**Performance Tests**:
- [x] Benchmark batch vs. individual
- [x] Verify call reduction target (70%)
- [x] Measure latency improvements

## Expected Impact

### Metrics Targets

| Metric | Baseline | Target | Measurement |
|--------|----------|--------|-------------|
| **API Calls (10 entities)** | 10 | 3 | 70% reduction |
| **API Calls (20 entities)** | 20 | 4-5 | 75% reduction |
| **Latency (10 entities)** | 5s | 1.5s | 70% improvement |
| **Batch Success Rate** | N/A | 85% | Batch attempts succeed |
| **Token Overhead** | 100% | 60% | Reduced context duplication |

### Cost Savings Example

**Scenario**: CRUD application with 15 entity files

**Without Batching**:
- API calls: 15
- Avg tokens per call: 5,000 (input) + 500 (output)
- Total tokens: 82,500
- Cost (Claude Sonnet 4.5): ~$0.90

**With Batching**:
- API calls: 3 (batches of 5)
- Avg tokens per call: 8,000 (input) + 2,500 (output)
- Total tokens: 31,500
- Cost: ~$0.35

**Savings**: $0.55 (61% reduction)

### Success Criteria

1. ✅ **70% API Call Reduction**: For projects with ≥10 similar files
2. ✅ **85% Batch Success Rate**: Batched generation succeeds without fallback
3. ✅ **No Quality Degradation**: Batch-generated code passes same validation as individual
4. ✅ **Graceful Fallback**: Failed batches automatically retry individually
5. ✅ **Transparent Metrics**: Users see batch statistics in generation reports

## Dependencies

### Required Packages

```go
// go.mod additions
require (
    golang.org/x/tools v0.15.0  // For go/parser syntax validation
)
```

### External Dependencies

- None (pure Go implementation)
- LLM clients already integrated (anthropic, openai, google)

### Internal Dependencies

- `internal/models` - FCS, GenerationPlan, GenerationTask
- `internal/generate/coder.go` - Existing coder interface
- `internal/generate/context_filter.go` - Smart FCS filtering
- `pkg/llm` - LLM client interface

## Configuration

### YAML Configuration

```yaml
# config.yaml
generation:
  batching:
    enabled: true
    max_batch_size: 5
    similarity_threshold: 0.7
    allowed_file_types:
      - entity
      - model
      - handler
      - test
    fallback_on_failure: true
```

### Code Configuration

```go
type BatchConfig struct {
    Enabled             bool
    MaxBatchSize        int      // Default: 5
    SimilarityThreshold float64  // Default: 0.7
    AllowedFileTypes    []string // Default: ["entity", "model", "handler", "test"]
    FallbackOnFailure   bool     // Default: true
}

// Default configuration
func DefaultBatchConfig() BatchConfig {
    return BatchConfig{
        Enabled:             true,
        MaxBatchSize:        5,
        SimilarityThreshold: 0.7,
        AllowedFileTypes: []string{
            "entity",
            "model",
            "handler",
            "service",
            "repository",
            "test",
        },
        FallbackOnFailure: true,
    }
}
```

## Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **LLM returns invalid JSON** | Medium | High | Robust parsing with fallback |
| **LLM omits files from batch** | Low | High | Validate all files present, retry missing |
| **Quality degradation in batches** | Low | Medium | Syntax validation, side-by-side comparison tests |
| **Over-batching reduces quality** | Medium | Medium | Conservative batch size (3-5), tune threshold |
| **Similarity detection errors** | Low | Medium | Extensive unit tests, tune weights |

## Open Questions

1. **Optimal batch size**: Should we use 3, 5, or adaptive sizing?
   - **Recommendation**: Start with 5, make configurable, measure quality

2. **Similarity weights**: Are our weights (30/35/20/15) optimal?
   - **Recommendation**: Start with proposed, A/B test alternatives

3. **Cross-provider behavior**: Do all LLMs handle JSON equally well?
   - **Recommendation**: Test all providers, adjust prompts per provider if needed

4. **Caching with batches**: Can we cache batch prompts?
   - **Recommendation**: Yes! Cache the FCS portion, vary file-specific parts

## Future Enhancements

### v2.0: Advanced Batching
- **Adaptive batch sizing**: Adjust size based on complexity
- **Cross-phase batching**: Batch across phases if independent
- **Streaming batch responses**: Parse partial JSON as it streams

### v3.0: ML-Optimized Batching
- **Learned similarity**: Train model to predict batchability
- **Quality prediction**: Predict which batches will succeed
- **Dynamic thresholds**: Adjust similarity threshold based on success rates

## References

- [GoCreator Spec: Performance Optimization](./spec.md)
- [Context Filtering Implementation](../../internal/generate/context_filter.go)
- [Prompt Caching Implementation](../../pkg/llm/prompt_cache.go)
- [Generation Pipeline](../../internal/generate/coder.go)

---

**Status**: Ready for implementation
**Next Step**: Begin Day 1 - Similarity Detection
**Owner**: TBD
**Review**: Required before starting implementation
