# Contributing to CommitLint.ai

Thank you for your interest in contributing to **CommitLint.ai**! We welcome contributions from developers of all skill levels.

---

## 🛠️ Development Setup

1. **Prerequisites:**
   - Go 1.21 or higher installed (`go version`)
   - Git installed (`git --version`)
   - A free Gemini API key from [Google AI Studio](https://aistudio.google.com/apikey)

2. **Fork & Clone:**
   ```bash
   git clone https://github.com/07Durgesh/commitlint-ai.git
   cd commitlint-ai
   ```

3. **Configure Environment:**
   ```bash
   cp .env.example .env
   # Add your GEMINI_API_KEY inside .env
   ```

4. **Run Unit Tests:**
   ```bash
   cd backend
   go test -v ./...
   ```

5. **Start Local Development Server:**
   ```bash
   go run cmd/server/main.go
   ```

---

## 📐 Coding Conventions

- **Standard Library First:** We prioritize standard Go idioms and standard library (`net/http`) design patterns over heavy external frameworks.
- **Error Handling:** Always wrap errors with context using `fmt.Errorf("...: %w", err)`.
- **Testing:** Any new handler or middleware must be accompanied by comprehensive `httptest` unit tests.
- **Git Commits:** Follow the **Conventional Commits** specification (`feat:`, `fix:`, `docs:`, `refactor:`, `perf:`, `test:`, `ci:`).

---

## 📬 Submitting a Pull Request

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   ```
2. Commit your changes with a clear commit message.
3. Ensure all tests pass: `go test -v ./...`
4. Push to your fork and submit a Pull Request.
5. GitHub Actions CI will automatically run tests and static analysis.
