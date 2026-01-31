tinyMem Architecture Overhaul: Embedded Embeddings

 User's Brilliant Insight

 "Can we embed the embedding in tinyMem? Whatever we use (Claude, Codex, Gemini, etc.), it would be good for CoVe, Ralph, and semantic to be consistent, and it would be less
 resource intensive to run it all from the tinyMem application."

 Research confirms: YES, absolutely feasible! ✅

 Vision: Self-Contained tinyMem

 Target Architecture

 Single tinyMem Binary (~150-200 MB)
 ├── Core application (current ~50 MB)
 ├── Embedded embedding model (90-100 MB GGUF)
 └── No external dependencies for semantic features

 What This Solves

 1. ❌ No more external Ollama needed for embeddings
 2. ✅ Consistent across all features (CoVe, Ralph, semantic)
 3. ✅ Works offline completely
 4. ✅ Single binary deployment
 5. ✅ Less resource intensive (no separate Ollama process)

 Current Problems (From Exploration)

 Problem 1: Unnecessary LLM Dependency

 - CoVe and Ralph require separate LLM in MCP mode
 - In MCP mode, Claude IS the LLM - Ollama is wasteful
 - Should use conversation LLM, not separate backend

 Problem 2: External Embedding Dependency

 - Semantic search requires external Ollama/embedding service
 - Users must run separate process
 - Network dependency, potential failure point

 Problem 3: Inconsistent Configuration

 - Different settings for LLM vs embeddings
 - Unclear what needs what
 - Silent failures when services unavailable

 Proposed Solution: Two-Phase Approach

 Phase 1: Embed Embedding Model (THIS RELEASE - v0.3.1)

 Goal: Eliminate external embedding dependency

 Technical Approach

 1. Choose Library
 - Recommended: kelindar/search (no cgo, simplest)
 - Alternative: go-llama.cpp (more mature, requires cgo)

 2. Bundle Model
 - Model: nomic-embed-text-v1.5 Q4_K_M (90 MB)
 - Alternative: all-MiniLM-L6-v2 Q4 (40 MB, faster)
 - Location: internal/models/nomic-embed-text-v1.5.Q4_K_M.gguf
 - Embedding: Use Go's //go:embed directive OR lazy download

 3. Implementation
 - Create internal/embedding/local.go - local inference
 - Keep internal/semantic/embedding.go - HTTP client (fallback)
 - Add EmbeddingMode config: "local" (default) or "http"
 - Modify internal/app/modules.go to choose embedding provider

 Code Changes

 New File: internal/embedding/local.go
 package embedding

 import "github.com/kelindar/search"

 type LocalEmbedder struct {
     model *search.BertEmbedder
 }

 func NewLocalEmbedder(modelPath string) (*LocalEmbedder, error) {
     embedder := search.NewBertEmbedder()
     // Load model from embedded file or path
     return &LocalEmbedder{model: embedder}, nil
 }

 func (e *LocalEmbedder) GenerateEmbedding(text string) ([]float32, error) {
     return e.model.Embed([]string{text})[0], nil
 }

 Modified: internal/app/modules.go
 func (a *App) InitializeRecallServices() *RecallServices {
     evidenceService := evidence.NewService(a.Core.DB, a.Core.Config)

     // Create recall engine
     var recallEngine recall.Recaller
     if a.Core.Config.SemanticEnabled {
         // NEW: Choose embedding provider
         var embeddingClient semantic.EmbeddingProvider
         if a.Core.Config.EmbeddingMode == "local" {
             embeddingClient = embedding.NewLocalEmbedder(a.Core.Config.EmbeddingModelPath)
         } else {
             embeddingClient = semantic.NewEmbeddingClient(a.Core.Config)
         }

         recallEngine = semantic.NewSemanticEngine(
             a.Core.DB, a.Memory, evidenceService,
             a.Core.Config, a.Core.Logger, embeddingClient,
         )
     } else {
         recallEngine = recall.NewEngine(...)
     }

     // CoVe remains optional
     var coveVerifier *cove.Verifier
     if a.Core.Config.CoVeEnabled {
         llmClient := llm.NewClient(a.Core.Config)
         coveVerifier = cove.NewVerifier(a.Core.Config, llmClient)
         extractor.SetCoVeVerifier(coveVerifier)
     }

     return &RecallServices{...}
 }

 New Config: internal/config/config.go
 type Config struct {
     // ... existing fields ...

     // Embedding configuration
     EmbeddingMode      string // "local" (default) or "http"
     EmbeddingModelPath string // Path to local model file
     EmbeddingBaseURL   string // For HTTP mode (backward compat)
     EmbeddingModel     string // Model name
 }

 // Defaults
 EmbeddingMode:      "local",
 EmbeddingModelPath: "internal/models/nomic-embed-text-v1.5.Q4_K_M.gguf",

 Files to Create/Modify

 New Files:
 - internal/embedding/local.go - Local embedding implementation
 - internal/embedding/interface.go - Common interface
 - internal/models/README.md - Model documentation
 - Model file: Download or embed in binary

 Modified Files:
 - internal/config/config.go - Add EmbeddingMode config
 - internal/app/modules.go - Choose embedding provider
 - internal/semantic/engine.go - Accept interface, not concrete type
 - go.mod - Add github.com/kelindar/search dependency

 Benefits

 - ✅ Works out-of-box, no Ollama needed
 - ✅ Offline semantic search
 - ✅ Single binary deployment
 - ✅ Backward compatible (can still use HTTP)
 - ✅ ~150 MB binary size (acceptable)

 Phase 2: Fix CoVe/Ralph LLM Usage (Future - v0.4.0)

 Goal: Eliminate unnecessary separate LLM in MCP mode

 This is a larger architectural change:
 1. CoVe in MCP mode should call Claude for verification (not Ollama)
 2. Ralph in MCP mode should use Claude for repairs
 3. Only proxy mode needs separate LLM backend

 This is future work - not for this release.

 Implementation Plan (Phase 1 Only)

 Step 1: Add Dependency & Model

 go get github.com/kelindar/search
 # Download nomic-embed-text-v1.5 Q4_K_M GGUF
 # Place in internal/models/ OR prepare for embedded download

 Step 2: Create Local Embedding Implementation

 1. Create internal/embedding/ package
 2. Implement LocalEmbedder using kelindar/search
 3. Define common EmbeddingProvider interface
 4. Test embedding generation

 Step 3: Update Configuration

 1. Add EmbeddingMode and EmbeddingModelPath to config
 2. Set "local" as default
 3. Keep HTTP mode for backward compatibility

 Step 4: Integrate into App Initialization

 1. Modify InitializeRecallServices() to choose provider
 2. Pass interface to SemanticEngine, not concrete type
 3. Test both local and HTTP modes

 Step 5: Update Documentation

 1. Create docs/EMBEDDINGS.md explaining local vs HTTP
 2. Update docs/LLM_DEPENDENCIES.md - embeddings now built-in
 3. Update README - no Ollama needed for semantic search!

 Step 6: Test & Verify

 # Build with embedded model
 go build -tags fts5 -o tinymem ./cmd/tinymem

 # Test semantic search works without Ollama
 tinymem write --type note --summary "test" --detail "testing local embeddings"
 tinymem query "local test"  # Should use embedded model

 # Verify in logs
 tail -f .tinyMem/logs/*.log | grep -i "embedding"

 Configuration Changes

 Before (External Ollama Required)

 [recall]
 semantic_enabled = true

 [embedding]
 base_url = "http://localhost:11434"  # Ollama must be running
 model = "nomic-embed-text"

 After (Self-Contained Default)

 [recall]
 semantic_enabled = true

 [embedding]
 mode = "local"  # NEW: "local" (default) or "http"
 # model_path auto-detected from binary

 Optional: HTTP Mode (Backward Compat)

 [embedding]
 mode = "http"
 base_url = "http://localhost:11434"
 model = "nomic-embed-text"

 Documentation Plan

 New: docs/EMBEDDINGS.md

 Sections:
 1. Local Embeddings (Default)
   - How it works (embedded model)
   - No external dependencies
   - Performance characteristics
 2. HTTP Mode (Optional)
   - When to use (different model, cloud service)
   - Configuration
   - Testing
 3. Model Information
   - Bundled model: nomic-embed-text-v1.5 Q4_K_M
   - Size: 90 MB
   - Quality vs speed trade-offs

 Update: docs/LLM_DEPENDENCIES.md

 Key Changes:
 - ✅ Semantic search: No longer needs external service!
 - ⚠️ CoVe: Still needs LLM (disabled by default)
 - ⚠️ Ralph: Needs LLM (for proxy mode)
 - ✅ Proxy mode: Needs backend LLM
 - ✅ MCP mode: Works completely standalone!

 Update: README.md

 Feature Section:
 - ✨ Built-in semantic search - No external dependencies
 - 🔍 Self-contained vector embeddings
 - 📦 Single binary deployment

 Files to Modify

 Code

 - internal/embedding/local.go - NEW
 - internal/embedding/interface.go - NEW
 - internal/config/config.go - Add EmbeddingMode, EmbeddingModelPath
 - internal/app/modules.go - Choose embedding provider
 - internal/semantic/engine.go - Accept interface
 - go.mod - Add kelindar/search

 Documentation

 - docs/EMBEDDINGS.md - NEW
 - docs/LLM_DEPENDENCIES.md - Update (embeddings now built-in)
 - README.md - Update features
 - examples/Configuration.md - Add EmbeddingMode examples

 Models

 - Download or bundle nomic-embed-text-v1.5.Q4_K_M.gguf
 - Decision needed: Embed in binary OR lazy download on first use

 Verification Plan

 1. Build Test:
 go build -tags fts5 -o tinymem ./cmd/tinymem
 ls -lh tinymem  # Should be ~150-200 MB
 2. Functional Test (No Ollama):
 # Kill Ollama if running
 killall ollama

 # Enable semantic search
 echo '[recall]\nsemantic_enabled = true' >> .tinyMem/config.toml

 # Test embedding generation
 tinymem write --type note --summary "semantic test"
 tinymem query "semantic"

 # Should work without external service!
 3. Performance Test:
 # Time embedding generation
 time tinymem query "test query"
 # Should be <100ms for first call, <50ms cached
 4. Backward Compatibility:
 [embedding]
 mode = "http"
 base_url = "http://localhost:11434"
 4. Should still work with Ollama

 Open Questions for User

 1. Model bundling strategy:
   - Option A: Embed 90MB model in binary (single file)
   - Option B: Download on first use (smaller initial binary)
   - Recommendation: Embed for true self-containment
 2. Model choice:
   - Option A: nomic-embed-text-v1.5 Q4 (90 MB, current default)
   - Option B: all-MiniLM-L6-v2 Q4 (40 MB, faster)
   - Recommendation: nomic for quality
 3. CoVe default:
   - Keep disabled (current plan)?
   - Or enable now that we have local embeddings?
   - Recommendation: Still disable (separate LLM issue)

 Timeline

 - Library integration: 1 hour
 - Local embedder implementation: 2 hours
 - Configuration changes: 1 hour
 - Testing: 2 hours
 - Documentation: 2 hours
 - Total: ~8 hours work

 Success Criteria

 ✅ tinyMem binary includes embedding model
 ✅ Semantic search works without Ollama
 ✅ Performance acceptable (<100ms per embedding)
 ✅ Backward compatible with HTTP mode
 ✅ Documentation clear and complete
 ✅ Binary size acceptable (~150-200 MB)
