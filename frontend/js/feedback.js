/**
 * EVENRTRA Feedback Modal & Submission Controller
 */

function initFeedback() {
  if (document.getElementById('feedback-modal-overlay')) return;

  const modalHtml = `
    <div class="modal-overlay" id="feedback-modal-overlay">
      <div class="modal-box">
        <button class="modal-close-btn" onclick="closeFeedbackModal()">&times;</button>
        <div style="margin-bottom:1.5rem;">
          <div class="badge badge-cyan" style="margin-bottom:0.5rem;">Campus Feedback</div>
          <h3 style="font-size:1.4rem;">Share Your Experience</h3>
          <p style="font-size:0.9rem; color:var(--text-muted); margin-top:0.3rem;">
            Help university clubs and student affairs improve future events.
          </p>
        </div>

        <form id="feedback-form" onsubmit="handleFeedbackSubmit(event)">
          <div class="form-group">
            <label class="form-label">Category <span class="required">*</span></label>
            <select class="form-control" id="feedback-category" required>
              <option value="general">General Suggestion</option>
              <option value="complaint">Complaint or Issue</option>
              <option value="suggestion">Club / Feature Idea</option>
              <option value="bug">Bug Report</option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">Rating</label>
            <div style="display:flex; gap:0.5rem;" id="rating-stars">
              <button type="button" class="btn btn-sm btn-glass rating-star-btn" data-val="1">★ 1</button>
              <button type="button" class="btn btn-sm btn-glass rating-star-btn" data-val="2">★ 2</button>
              <button type="button" class="btn btn-sm btn-glass rating-star-btn" data-val="3">★ 3</button>
              <button type="button" class="btn btn-sm btn-glass rating-star-btn" data-val="4">★ 4</button>
              <button type="button" class="btn btn-sm btn-glass rating-star-btn" data-val="5" style="border-color:var(--accent-cyan); color:var(--accent-cyan);">★ 5</button>
            </div>
            <input type="hidden" id="feedback-rating" value="5">
          </div>

          <div class="form-group">
            <label class="form-label">Your Message <span class="required">*</span></label>
            <textarea class="form-control" id="feedback-message" placeholder="Provide detailed feedback, recommendations or describe any problem you encountered..." required></textarea>
          </div>

          <div class="form-group">
            <label class="form-label">Contact Email (Optional)</label>
            <input type="email" class="form-control" id="feedback-email" placeholder="student@university.edu for follow-up">
          </div>

          <div style="display:flex; justify-content:flex-end; gap:0.75rem; margin-top:1.5rem;">
            <button type="button" class="btn btn-glass" onclick="closeFeedbackModal()">Cancel</button>
            <button type="submit" class="btn btn-primary" id="feedback-submit-btn">
              <span>Submit Feedback</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  `;

  document.body.insertAdjacentHTML('beforeend', modalHtml);

  // Bind star buttons
  const starBtns = document.querySelectorAll('.rating-star-btn');
  starBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const val = btn.getAttribute('data-val');
      document.getElementById('feedback-rating').value = val;
      starBtns.forEach(b => {
        if (parseInt(b.getAttribute('data-val')) <= parseInt(val)) {
          b.style.borderColor = 'var(--accent-cyan)';
          b.style.color = 'var(--accent-cyan)';
        } else {
          b.style.borderColor = 'var(--border-glass)';
          b.style.color = 'var(--text-muted)';
        }
      });
    });
  });
}

function openFeedbackModal() {
  initFeedback();
  const overlay = document.getElementById('feedback-modal-overlay');
  if (overlay) {
    const user = window.api.getUser();
    if (user && user.email) {
      const emailField = document.getElementById('feedback-email');
      if (emailField && !emailField.value) emailField.value = user.email;
    }
    overlay.classList.add('active');
  }
}

function closeFeedbackModal() {
  const overlay = document.getElementById('feedback-modal-overlay');
  if (overlay) overlay.classList.remove('active');
}

async function handleFeedbackSubmit(e) {
  e.preventDefault();
  const submitBtn = document.getElementById('feedback-submit-btn');
  submitBtn.disabled = true;
  submitBtn.innerText = 'Submitting...';

  const category = document.getElementById('feedback-category').value;
  const rating = parseInt(document.getElementById('feedback-rating').value) || 5;
  const message = document.getElementById('feedback-message').value.trim();
  const email = document.getElementById('feedback-email').value.trim();

  try {
    const res = await window.api.post('/feedback', {
      category,
      rating,
      message,
      email: email || undefined
    });

    window.showToast('Thank you! Your feedback has been received.', 'success');
    closeFeedbackModal();
    document.getElementById('feedback-form').reset();
  } catch (err) {
    window.showToast(err.message || 'Failed to submit feedback', 'error');
  } finally {
    submitBtn.disabled = false;
    submitBtn.innerHTML = '<span>Submit Feedback</span>';
  }
}

document.addEventListener('DOMContentLoaded', initFeedback);
