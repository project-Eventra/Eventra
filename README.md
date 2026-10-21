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

---

## 🔑 Pre-Configured Demo Credentials

Seed data includes active accounts ready for immediate testing:

| Role | Email | Password | Access Level |
|---|---|---|---|
| **Super Admin** | `admin@eventra.edu` | `Admin@123456` | Full University Admin Console |
| **Robotics Club** | `robotics@eventra.edu` | `Club@123456` | Club Management Portal |
| **IEEE Student Branch** | `ieee@eventra.edu` | `Club@123456` | Club Management Portal |
| **Drama Guild** | `drama@eventra.edu` | `Club@123456` | Club Management Portal |
| **Student** | `student@eventra.edu` | `Student@123456` | Student Ticket Portal |

---

## 📁 Directory Structure

```
eventra/
├── render.yaml                    # Render deployment blueprint
├── .env.example                   # Environment configuration template
├── README.md                      # Complete documentation
│
├── backend/                       # Go Backend
│   ├── go.mod                     # Go module definitions
│   ├── go.sum                     # Checksums
│   ├── cmd/
│   │   └── server/
│   │       └── main.go            # Application entrypoint & HTTP server
│   ├── internal/
│   │   ├── auth/                  # JWT generation/validation & Firebase Auth
│   │   ├── config/                # Environment configuration loader
│   │   ├── database/              # PostgreSQL pgx pool & migration runner
│   │   ├── handlers/              # Chi HTTP request handlers
│   │   ├── middleware/            # Auth, CORS, RateLimiting, SecurityHeaders
│   │   ├── models/                # Domain models, requests & responses
│   │   ├── repositories/          # PostgreSQL data access layer
│   │   └── validation/            # Request validators
│   └── migrations/
│       ├── 001_init.up.sql        # Tables, foreign keys, and indexes
│       ├── 001_init.down.sql      # Schema rollback
│       ├── 002_seed.up.sql        # Demo clubs, accounts, events, and passes
│       └── 002_seed.down.sql      # Seed data rollback
│
└── frontend/                      # Vanilla Web Application
    ├── index.html                 # Landing page & orbital universe model
    ├── events.html                # Events catalog, search & timeline view
    ├── event-detail.html          # Details, dynamic fields & group registration
    ├── clubs.html                 # Certified university societies directory
    ├── auth.html                  # Unified login, student & club sign up
    ├── dashboard-student.html     # Student portal with QR passes
    ├── dashboard-club.html        # Club admin event wizard & attendee roster
    ├── dashboard-admin.html       # Super admin analytics & club approvals
    ├── css/
    │   ├── main.css               # Design system, glassmorphism & buttons
    │   ├── animations.css         # Planetary orbit rotation, glows & tickers
    │   └── dashboard.css          # Tables, stats, and ticket pass styling
    └── js/
        ├── api.js                 # Central API client & toast notifications
        ├── navbar.js              # Role-aware navigation controller
        ├── events.js              # Event loaders, search & countdown clocks
        └── feedback.js            # Feedback modal & rating submission
```

---

## 🏃 Running Locally

### Prerequisites
1. **Go 1.21+** installed (`go version`)
2. **PostgreSQL** running locally or in the cloud (e.g. Neon, Supabase, Render)

### 1. Configure Environment
Copy `.env.example` to `.env` or set environment variables:
```bash
cp .env.example .env
```

### 2. Run the Application
From the `backend/` directory:
```bash
cd backend
go run ./cmd/server
```
The server will automatically:
- Connect to PostgreSQL
- Execute any pending migrations (`001_init.up.sql`, `002_seed.up.sql`)
- Serve the REST API on `http://localhost:8080/api/v1`
- Automatically serve frontend files from `http://localhost:8080`

### 3. Open in Browser
Open `http://localhost:8080` in your web browser.

---

## 🌐 Deploying to Render

1. Push this repository to GitHub or GitLab.
2. Log in to [Render](https://render.com).
3. Click **New +** -> **Blueprint**.
4. Connect this repository. Render will automatically detect `render.yaml`, provision a managed PostgreSQL database, build the Go application, run migrations, and launch Eventra live!
