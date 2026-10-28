// Automatically resolve API base URL:
// - Standalone local file (file://) or dev servers -> use http://localhost:8080
// - Production (Vercel) -> use relative path '' (same-origin, zero CORS issues!)
const API_BASE = (window.location.protocol === 'file:' || window.location.port === '3000' || window.location.port === '5500')
  ? 'http://localhost:8080'
  : '';

const SAMPLE_DIFF = `diff --git a/backend/internal/middleware/ratelimit.go b/backend/internal/middleware/ratelimit.go
new file mode 100644
index 0000000..7a8b9c1
--- /dev/null
+++ b/backend/internal/middleware/ratelimit.go
@@ -0,0 +1,45 @@
+package middleware
+
+import (
+	"net/http"
+	"sync"
+	"time"
+)
+
+type RateLimiter struct {
+	mu       sync.Mutex
+	visitors map[string]*visitor
+	rate     float64
+	capacity float64
+}
+
+func NewRateLimiter(rate, capacity float64) *RateLimiter {
+	return &RateLimiter{
+		visitors: make(map[string]*visitor),
+		rate:     rate,
+		capacity: capacity,
+	}
+}
+
+func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
+	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		// Enforce token bucket rate limiting per IP
+		next.ServeHTTP(w, r)
+	})
+}`;

document.addEventListener('DOMContentLoaded', () => {
  const diffInput = document.getElementById('diff-input');
  const contextInput = document.getElementById('context-input');
  const styleSelect = document.getElementById('style-select');
  const analyzeBtn = document.getElementById('analyze-btn');
  const sampleDiffBtn = document.getElementById('sample-diff-btn');
  const errorBanner = document.getElementById('error-banner');
  const resultsSection = document.getElementById('results-section');
  const healthStatus = document.getElementById('health-status');

  const commitSubject = document.getElementById('commit-subject');
  const commitBody = document.getElementById('commit-body');
  const commitBodyWrapper = document.getElementById('commit-body-wrapper');
  const commitFull = document.getElementById('commit-full');
  const prFull = document.getElementById('pr-full');
  const observationsList = document.getElementById('observations-list');
  const observationsCard = document.getElementById('observations-card');

  // 1. Check backend health
  async function checkHealth() {
    try {
      const res = await fetch(`${API_BASE}/api/health`);
      if (res.ok) {
        const data = await res.json();
        healthStatus.textContent = `🟢 Backend connected (v${data.version || '0.1.0'})`;
        healthStatus.style.color = '#7ee787';
      } else {
        throw new Error();
      }
    } catch {
      healthStatus.textContent = '🔴 Backend offline (start server at :8080)';
      healthStatus.style.color = '#f85149';
    }
  }
  checkHealth();

  // 2. Sample Diff loader
  sampleDiffBtn.addEventListener('click', () => {
    diffInput.value = SAMPLE_DIFF;
    contextInput.value = 'Add concurrent token-bucket rate limiter middleware to protect upstream API';
    clearError();
  });

  // 3. Analyze click handler
  analyzeBtn.addEventListener('click', async () => {
    const diff = diffInput.value.trim();
    const context = contextInput.value.trim();
    const style = styleSelect.value;

    if (!diff) {
      showError('Please paste a git diff or click "Load Sample Diff" first.');
      return;
    }

    setLoading(true);
    clearError();

    try {
      const response = await fetch(`${API_BASE}/api/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ diff, context, style }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || `Server responded with status ${response.status}`);
      }

      renderResults(data);
    } catch (err) {
      showError(err.message || 'An unexpected error occurred while communicating with the backend.');
    } finally {
      setLoading(false);
    }
  });

  // 4. Render results to DOM
  function renderResults(data) {
    // Commit message
    if (data.commit_message) {
      commitSubject.textContent = data.commit_message.subject || 'N/A';
      if (data.commit_message.body) {
        commitBody.textContent = data.commit_message.body;
        commitBodyWrapper.hidden = false;
      } else {
        commitBodyWrapper.hidden = true;
      }
      commitFull.value = data.commit_message.full || `${data.commit_message.subject}\n\n${data.commit_message.body || ''}`;
    }

    // PR Description
    if (data.pr_description) {
      prFull.textContent = data.pr_description.full_markdown ||
        `## What\n${data.pr_description.what}\n\n## Why\n${data.pr_description.why}\n\n## How\n${data.pr_description.how}`;
    }

    // Observations
    observationsList.innerHTML = '';
    const obs = data.observations || [];
    if (obs.length > 0) {
      obs.forEach(item => {
        const div = document.createElement('div');
        div.className = `obs-item obs-${item.type || 'suggestion'}`;
        const icon = item.type === 'warning' ? '⚠️' : item.type === 'issue' ? '🛑' : '💡';
        div.innerHTML = `<span>${icon}</span> <span>${escapeHtml(item.message)}</span>`;
        observationsList.appendChild(div);
      });
      observationsCard.hidden = false;
    } else {
      observationsCard.hidden = true;
    }

    resultsSection.hidden = false;
    resultsSection.scrollIntoView({ behavior: 'smooth' });
  }

  // 5. Copy to clipboard
  document.querySelectorAll('.btn-copy').forEach(btn => {
    btn.addEventListener('click', async () => {
      const targetId = btn.getAttribute('data-target');
      const targetEl = document.getElementById(targetId);
      const textToCopy = targetEl.value !== undefined ? targetEl.value : targetEl.textContent;

      if (!textToCopy) return;

      try {
        await navigator.clipboard.writeText(textToCopy);
        const originalText = btn.textContent;
        btn.textContent = 'Copied! ✓';
        btn.style.color = '#7ee787';
        setTimeout(() => {
          btn.textContent = originalText;
          btn.style.color = '';
        }, 1800);
      } catch (err) {
        alert('Failed to copy to clipboard.');
      }
    });
  });

  function setLoading(isLoading) {
    analyzeBtn.disabled = isLoading;
    const textSpan = analyzeBtn.querySelector('.btn-text-content');
    const spinner = analyzeBtn.querySelector('.spinner');

    if (isLoading) {
      textSpan.textContent = 'Analyzing Diff...';
      spinner.hidden = false;
    } else {
      textSpan.textContent = '✨ Generate Commit & PR';
      spinner.hidden = true;
    }
  }

  function showError(msg) {
    errorBanner.textContent = msg;
    errorBanner.hidden = false;
  }

  function clearError() {
    errorBanner.textContent = '';
    errorBanner.hidden = true;
  }

  function escapeHtml(str) {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }
});

