# EVENTRA

> **"All clubs/events revolving around one platform"**  
> *Innovation, collaboration and leadership*

Eventra is a university club and event management web application. It centralizes student societies, digital event registrations, attendee passes with QR verification, waitlists, custom application questions, and administration moderation into a unified web application.

---

## 🛠️ Technology Stack

| Layer | Technologies |
|---|---|
| **Frontend** | HTML5, CSS3 (Custom Glassmorphism Design System, CSS Animations), Vanilla JavaScript (ES6+ Modules) |
| **Backend** | Go (Golang 1.21+), `go-chi/chi/v5` Router |
| **Database** | PostgreSQL with `jackc/pgx/v5` Connection Pooling |
| **Authentication** | Custom HMAC-SHA256 JWT sessions (HttpOnly Secure Cookies + Bearer fallback) & Firebase Admin SDK |
| **Deployment** | Render (`render.yaml` Blueprint for Go Web Service + Managed PostgreSQL) |

---

## 🚀 Key Features

1. **Revolving Campus Orbital Showcase**:
   - Dynamic 3D CSS planetarium / orbit visualization representing university clubs rotating around the central Eventra Core.
2. **Event Registration Engine**:
   - Individual & group/team registration support.
   - Dynamic custom registration questions builder (text, numbers, emails, selects).
   - Real-time countdown tickers to registration deadlines.
   - Capacity tracking and automatic cutoff enforcement.
3. **Student Portal (`dashboard-student.html`)**:
   - Digital event tickets with tear notches, registration identifiers, and entry QR codes.
   - My Registrations tracking (upcoming, confirmed, attended).
4. **Club Admin Console (`dashboard-club.html`)**:
   - Event creation wizard with custom field designer.
   - Deadline extension modal with audit logging.
   - Live participant roster viewer with search and status checks.
5. **Super Admin Console (`dashboard-admin.html`)**:
   - Global university statistics (total clubs, students, events, registrations).
   - Club certification queue (Approve or Reject with reason).
   - Student registry and event moderation.
   - Campus feedback and incident review inbox.
6. **Campus Feedback System**:
   - Integrated modal for submitting complaints, bug reports, and suggestions with star ratings.
