/**
 * EVENTRA Dynamic Navbar & Auth State Controller
 */

function initNavbar() {
  const navContainer = document.getElementById('navbar-container');
  if (!navContainer) return;

  const user = window.api.getUser();
  const currentPath = window.location.pathname;

  let portalLink = '';
  let userActions = '';

  if (user) {
    let portalUrl = 'dashboard-student.html';
    let portalLabel = 'Student Portal';
    let roleClass = 'role-student';
    let roleText = 'Student';

    if (user.role === 'club_admin') {
      portalUrl = 'dashboard-club.html';
      portalLabel = 'Club Portal';
      roleClass = 'role-club_admin';
      roleText = 'Club Admin';
    } else if (user.role === 'super_admin') {
      portalUrl = 'dashboard-admin.html';
      portalLabel = 'Admin Console';
      roleClass = 'role-super_admin';
      roleText = 'Super Admin';
    }

    const initials = (user.name || user.email || 'U').charAt(0).toUpperCase();

    userActions = `
      <a href="${portalUrl}" class="btn btn-sm btn-glass">${portalLabel}</a>
      <div class="user-menu-pill" id="user-menu-btn" title="${user.email}">
        <div class="user-avatar-circle">${initials}</div>
        <span style="font-size:0.85rem; font-weight:600;">${escapeHtml(user.name || user.email.split('@')[0])}</span>
        <span class="user-role-badge ${roleClass}">${roleText}</span>
      </div>
      <button class="btn btn-sm btn-glass" onclick="handleLogout()" title="Sign Out">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path><polyline points="16 17 21 12 16 7"></polyline><line x1="21" y1="12" x2="9" y2="12"></line></svg>
      </button>
    `;
  } else {
    userActions = `
      <a href="auth.html?action=login" class="btn btn-sm btn-glass">Log In</a>
      <a href="auth.html?action=signup" class="btn btn-sm btn-primary">Sign Up</a>
    `;
  }

  navContainer.innerHTML = `
    <nav class="navbar" id="main-nav">
      <div class="container navbar-inner">
        <a href="index.html" class="brand-logo">
          <div class="logo-orb"></div>
          <span>Eventra</span>
        </a>

        <ul class="nav-links">
          <li><a href="index.html" class="nav-link ${currentPath.endsWith('index.html') || currentPath.endsWith('/') ? 'active' : ''}">Home</a></li>
          <li><a href="events.html" class="nav-link ${currentPath.includes('event') ? 'active' : ''}">Events</a></li>
          <li><a href="clubs.html" class="nav-link ${currentPath.includes('club') && !currentPath.includes('dashboard') ? 'active' : ''}">Clubs</a></li>
          <li><a href="events.html#timeline" class="nav-link">Timeline</a></li>
          <li><button class="btn btn-sm btn-glass" onclick="openFeedbackModal()" style="padding: 0.35rem 0.8rem; font-size: 0.82rem;">Feedback</button></li>
        </ul>

        <div class="nav-actions">
          ${userActions}
        </div>
      </div>
    </nav>
  `;

  // Scroll effect on navbar
  window.addEventListener('scroll', () => {
    const nav = document.getElementById('main-nav');
    if (nav) {
      if (window.scrollY > 20) {
        nav.classList.add('scrolled');
      } else {
        nav.classList.remove('scrolled');
      }
    }
  });
}

async function handleLogout() {
  try {
    await window.api.post('/auth/logout', {});
  } catch (e) {
    console.log('Logout notice:', e);
  } finally {
    window.api.clearAuth();
    window.showToast('You have been signed out', 'info');
    setTimeout(() => {
      window.location.href = 'index.html';
    }, 600);
  }
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

document.addEventListener('DOMContentLoaded', initNavbar);
