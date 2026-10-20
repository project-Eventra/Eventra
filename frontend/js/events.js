/**
 * EVENRTRA Events Engine
 * Catalog, Search, Filtering, Dynamic Countdowns, and Timeline Display
 */

// Global active intervals for countdown tickers
const countdownTimers = new Map();

/**
 * Render a single event card
 */
function createEventCard(event) {
  const isExpired = new Date(event.registration_deadline) <= new Date();
  const title = event.title || event.name || 'Campus Event';
  const capacity = event.max_capacity || event.max_participants || 100;
  const current = event.current_registrations || event.registration_count || 0;
  const percentFilled = Math.min(100, Math.round((current / capacity) * 100));
  const participationType = event.registration_type || event.participation_type || 'individual';

  let badgeClass = 'badge-cyan';
  let badgeText = 'Open for Registration';
  if (isExpired) {
    badgeClass = 'badge-rose';
    badgeText = 'Registration Closed';
  } else if (percentFilled >= 90) {
    badgeClass = 'badge-amber';
    badgeText = 'Filling Fast!';
  }

  const defaultBanner = 'https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=800&auto=format&fit=crop&q=80';
  const bannerUrl = event.banner_url || defaultBanner;
  const clubName = event.club_name || 'University Club';

  const rawDate = event.start_time || event.event_date || new Date();
  const eventDate = new Date(rawDate).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric'
  });
  const eventTime = new Date(rawDate).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit'
  });

  const countdownId = `countdown-${event.id}`;

  return `
    <div class="glass-card event-card" data-event-id="${event.id}">
      <div style="position:relative; height:180px; margin:-1.75rem -1.75rem 1.25rem; overflow:hidden; border-radius:var(--radius-lg) var(--radius-lg) 0 0;">
        <img src="${bannerUrl}" alt="${escapeHtml(title)}" style="width:100%; height:100%; object-fit:cover; transition:transform 0.4s ease;" class="event-banner-img" onerror="this.src='${defaultBanner}'">
        <div style="position:absolute; inset:0; background:linear-gradient(180deg, rgba(7,9,19,0.1) 0%, rgba(7,9,19,0.85) 100%);"></div>
        <div style="position:absolute; top:1rem; right:1rem;">
          <span class="badge ${badgeClass}">${badgeText}</span>
        </div>
        <div style="position:absolute; bottom:0.85rem; left:1rem; display:flex; align-items:center; gap:0.5rem;">
          <span class="badge badge-purple">${escapeHtml(event.category || 'General')}</span>
          ${participationType === 'group' ? '<span class="badge badge-emerald">Team / Group</span>' : '<span class="badge badge-cyan">Individual</span>'}
        </div>
      </div>

      <div style="display:flex; align-items:center; gap:0.5rem; margin-bottom:0.5rem;">
        <span style="font-size:0.82rem; font-weight:600; color:var(--accent-cyan);">${escapeHtml(clubName)}</span>
      </div>

      <h3 style="font-size:1.25rem; font-weight:700; margin-bottom:0.6rem; line-height:1.3; min-height:2.6rem;">
        <a href="event-detail.html?id=${event.id}" style="color:var(--text-main); hover:color:var(--accent-cyan);">${escapeHtml(title)}</a>
      </h3>

      <div style="display:flex; flex-direction:column; gap:0.35rem; font-size:0.85rem; color:var(--text-muted); margin-bottom:1.1rem;">
        <div style="display:flex; align-items:center; gap:0.5rem;">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="18" y2="10"></line></svg>
          <span>${eventDate} at ${eventTime}</span>
        </div>
        <div style="display:flex; align-items:center; gap:0.5rem;">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"></path><circle cx="12" cy="10" r="3"></circle></svg>
          <span>${escapeHtml(event.venue || 'Campus Auditorium')}</span>
        </div>
      </div>

      <!-- Capacity Bar -->
      <div style="margin-bottom:1.25rem;">
        <div style="display:flex; justify-content:space-between; font-size:0.78rem; font-weight:600; margin-bottom:0.35rem;">
          <span style="color:var(--text-dim);">Spots Taken</span>
          <span style="color:var(--text-main); font-family:var(--font-mono);">${current} / ${capacity} (${percentFilled}%)</span>
        </div>
        <div style="width:100%; height:6px; background:rgba(255,255,255,0.08); border-radius:var(--radius-full); overflow:hidden;">
          <div style="width:${percentFilled}%; height:100%; background:var(--grad-primary); border-radius:var(--radius-full); transition:width 0.5s ease;"></div>
        </div>
      </div>

      <!-- Countdown Bar -->
      <div style="background:rgba(9,26,17,0.85); border:1px solid var(--border-white-layer); border-radius:var(--radius-md); padding:0.65rem 0.85rem; margin-bottom:1.25rem; display:flex; align-items:center; justify-content:space-between;">
        <span style="font-size:0.75rem; color:var(--text-dim); text-transform:uppercase; font-weight:600;">Deadline</span>
        <div id="${countdownId}" class="countdown-ticker" data-deadline="${event.registration_deadline}" style="font-family:var(--font-mono); font-size:0.85rem; font-weight:700; color:var(--accent-mint);">
          Calculating...
        </div>
      </div>

      <div style="display:flex; gap:0.75rem;">
        <a href="event-detail.html?id=${event.id}" class="btn btn-sm btn-glass" style="flex:1;">View Details</a>
        ${isExpired ? `
          <button class="btn btn-sm btn-glass" disabled style="opacity:0.5; flex:1;">Closed</button>
        ` : event.is_registered ? `
          <span class="btn btn-sm btn-success" style="flex:1;">Registered ✓</span>
        ` : `
          <a href="event-detail.html?id=${event.id}#register" class="btn btn-sm btn-primary" style="flex:1;">Register</a>
        `}
      </div>
    </div>
  `;
}

/**
 * Start Countdown Ticker for all mounted counters
 */
function startCountdownTickers() {
  const tickers = document.querySelectorAll('.countdown-ticker');
  tickers.forEach(ticker => {
    const deadline = new Date(ticker.getAttribute('data-deadline')).getTime();

    function update() {
      const now = new Date().getTime();
      const distance = deadline - now;

      if (distance <= 0) {
        ticker.innerText = 'Deadline Passed';
        ticker.style.color = 'var(--accent-rose)';
        return;
      }

      const days = Math.floor(distance / (1000 * 60 * 60 * 24));
      const hours = Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
      const minutes = Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((distance % (1000 * 60)) / 1000);

      ticker.innerText = `${days}d ${hours}h ${minutes}m ${seconds}s`;
    }

    update();
    const timerId = setInterval(update, 1000);
    countdownTimers.set(ticker.id, timerId);
  });
}

/**
 * Fetch and Render Events Catalog
 */
async function loadEventsCatalog({ containerId, search = '', category = '', page = 1, limit = 9 }) {
  const container = document.getElementById(containerId);
  if (!container) return;

  container.innerHTML = `
    <div style="grid-column: 1 / -1; text-align: center; padding: 4rem 0;">
      <div style="width: 40px; height: 40px; border: 3px solid rgba(0, 242, 254, 0.2); border-top-color: var(--accent-cyan); border-radius: 50%; animation: rotateRing 1s linear infinite; margin: 0 auto 1rem;"></div>
      <p>Discovering university club events...</p>
    </div>
  `;

  try {
    const res = await window.api.get('/events', {
      search,
      page,
      page_size: limit
    });

    let events = res.data?.items || [];
    if (category) {
      events = events.filter(e => (e.category || '').toLowerCase() === category.toLowerCase());
    }

    if (events.length === 0) {
      const isFiltered = Boolean(search || category);
      container.innerHTML = `
        <div style="grid-column: 1 / -1; text-align: center; padding: 4rem 1.5rem;" class="glass-card">
          <div style="font-size: 3rem; margin-bottom: 0.75rem;">📅</div>
          <h3 style="font-size: 1.4rem; margin-bottom: 0.5rem; color: #ffffff;">
            ${isFiltered ? 'No events match your criteria' : 'No Events Scheduled Yet'}
          </h3>
          <p style="color: var(--text-muted); max-width: 480px; margin: 0 auto 1.5rem; line-height: 1.6;">
            ${isFiltered 
              ? 'Try selecting a different category or clearing your search term to see other events.' 
              : 'No campus events are currently open for registration as no clubs have added events yet. Once registered university clubs publish upcoming events, they will appear here with live countdowns.'}
          </p>
          <div style="display: flex; justify-content: center; gap: 0.75rem; flex-wrap: wrap;">
            ${isFiltered ? `
              <button class="btn btn-sm btn-glass" onclick="document.getElementById('event-search-input').value=''; loadEventsCatalog({ containerId:'${containerId}' });">
                Reset Filters
              </button>
            ` : `
              <a href="auth.html?mode=club" class="btn btn-sm btn-primary">Register Your Club</a>
              <a href="auth.html?mode=register" class="btn btn-sm btn-glass">Student Sign Up</a>
            `}
          </div>
        </div>
      `;
      return;
    }

    // Clear previous timers
    countdownTimers.forEach(id => clearInterval(id));
    countdownTimers.clear();

    container.innerHTML = events.map(createEventCard).join('');
    startCountdownTickers();
  } catch (err) {
    container.innerHTML = `
      <div style="grid-column: 1 / -1; text-align: center; padding: 3rem 1rem;" class="glass-card">
        <p style="color: var(--accent-rose); margin-bottom: 1rem;">Failed to load events: ${err.message}</p>
        <button class="btn btn-sm btn-glass" onclick="loadEventsCatalog({ containerId:'${containerId}' })">Retry</button>
      </div>
    `;
  }
}

/**
 * Load Timeline
 */
async function loadTimeline(containerId) {
  const container = document.getElementById(containerId);
  if (!container) return;

  try {
    const res = await window.api.get('/events/timeline');
    const items = res.data || [];

    if (items.length === 0) {
      container.innerHTML = `
        <div class="glass-card" style="text-align: center; padding: 2.5rem 1.5rem;">
          <div style="font-size: 2.2rem; margin-bottom: 0.5rem;">🗓️</div>
          <h4 style="font-size: 1.15rem; margin-bottom: 0.35rem; color: #ffffff;">No Timeline Events Scheduled Yet</h4>
          <p style="color: var(--text-muted); font-size: 0.9rem; max-width: 440px; margin: 0 auto;">
            The university event timeline will automatically synchronize as soon as registered campus clubs publish their upcoming activities.
          </p>
        </div>
      `;
      return;
    }

    container.innerHTML = items.map(item => {
      const dateStr = new Date(item.date).toLocaleDateString('en-US', {
        weekday: 'short',
        month: 'short',
        day: 'numeric'
      });
      return `
        <div class="timeline-node">
          <div class="timeline-dot"></div>
          <div class="glass-card" style="padding:1.25rem 1.5rem;">
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:0.4rem;">
              <span class="badge badge-cyan">${escapeHtml(item.club_name)}</span>
              <span style="font-family:var(--font-mono); font-size:0.8rem; color:var(--accent-cyan); font-weight:600;">${dateStr}</span>
            </div>
            <h4 style="font-size:1.1rem; margin-bottom:0.35rem;">
              <a href="event-detail.html?id=${item.id}" style="color:var(--text-main);">${escapeHtml(item.title)}</a>
            </h4>
            <div style="display:flex; align-items:center; gap:1.25rem; font-size:0.82rem; color:var(--text-muted);">
              <span>📍 ${escapeHtml(item.venue || 'Main Campus')}</span>
              <span>⏰ ${escapeHtml(item.time || '10:00 AM')}</span>
            </div>
          </div>
        </div>
      `;
    }).join('');
  } catch (e) {
    container.innerHTML = `<p style="color:var(--accent-rose);">Could not load timeline.</p>`;
  }
}

window.loadEventsCatalog = loadEventsCatalog;
window.loadTimeline = loadTimeline;
