# BoardSync - Asana ↔ YouTrack Sync Platform

A powerful, feature-rich synchronization platform that bridges Asana and YouTrack project management tools. Built with Go backend and React frontend, featuring real-time sync, advanced analysis, rollback capabilities, and comprehensive audit logging.

![Version](https://img.shields.io/badge/version-4.1-blue)
![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go)
![React](https://img.shields.io/badge/React-18.2-61DAFB?logo=react)

---

## Features

### Core Synchronization
- **Bidirectional Sync**: Seamlessly sync tasks between Asana and YouTrack
- **Auto-Sync & Auto-Create**: Automated synchronization with customizable intervals (15 seconds to hours)
- **Smart Ticket Mapping**: Intelligent mapping that handles ticket recreation and prevents duplicates
- **Column/State Mapping**: Customizable mapping between Asana sections and YouTrack states
- **Change Detection**: Detects and highlights changes before syncing
- **HTML to Markdown Conversion**: Preserves formatting when syncing descriptions

### Advanced Analysis
- **Enhanced Analysis Engine**: Analyzes tickets across multiple columns with filtering and sorting
- **Ticket Comparison**: Compare Asana and YouTrack tickets side-by-side
- **Conflict Detection**: Identifies differences in titles, descriptions, tags, and priorities
- **Custom Field Mapping**: Map tags, priorities, and custom fields between platforms

### History & Rollback
- **Sync History**: Complete history of all sync operations with timestamps
- **Audit Logs**: Detailed audit trail with user actions and ticket changes
- **Rollback/Restore**: Restore tickets to previous states (15 snapshots, 24h retention)
- **Snapshot Management**: Automatic snapshots before major operations

### User Experience
- **Glass Morphism UI**: Modern, beautiful interface with fluid animations
- **Real-time Updates**: WebSocket-based live updates
- **Responsive Design**: Works seamlessly on desktop and mobile
- **Background Link Opening**: External links open in background tabs
- **Custom Interval Editor**: Visual interval editor with time unit selection

---

## Tech Stack

**Backend:** Go 1.21, Gorilla Mux/WebSocket, JWT Auth, JSON Database
**Frontend:** React 18.2, Lucide Icons, Tailwind CSS 4.1, Three.js

---

## Installation & Setup

### Prerequisites
- Go 1.21 or higher
- Node.js 16.x or higher
- npm 8.x or higher

### Backend Setup

```bash
cd backend
go mod download
go run main.go
```
Backend runs on `http://localhost:8080`

### Frontend Setup

```bash
cd frontend
npm install
npm start          # Development server on http://localhost:3000
npm run build      # Production build
```

---

## Configuration

### Initial Setup

1. **Create an account** at `http://localhost:3000`
2. **Configure API credentials** in Settings:
   - Add Asana Personal Access Token (PAT)
   - Add YouTrack base URL and token
   - Select Asana project and YouTrack project
   - Select YouTrack agile board
3. **Configure column mappings** (optional) in Settings → Column Mapping

---

## Usage

### Dashboard Controls

**Auto-Sync:** Automatically sync existing tickets at regular intervals
- Click play to start
- Click edit icon to customize interval (seconds/minutes/hours)
- Minimum interval: 15 seconds

**Auto-Create:** Automatically create new tickets from Asana to YouTrack

**Column Selection:** Choose which Asana section to analyze (Backlog, In Progress, DEV, STAGE, etc.)

### Sync Workflow

1. Select a column from the dashboard
2. Click "Analyze" to fetch and compare tickets
3. Review analysis results (matches, differences, conflicts)
4. Select tickets to sync
5. Click "Sync Selected"

### Rollback & Restore

1. Go to Sync History
2. Find the operation to rollback
3. Click "Rollback" and confirm
4. Tickets restored to previous state

**Limitations:** Max 15 snapshots per ticket, 24h expiration

---

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login user
- `GET /api/auth/me` - Get current user

### Settings
- `GET /api/settings` - Get user settings
- `PUT /api/settings` - Update settings
- `GET /api/settings/asana/projects` - List Asana projects
- `GET /api/settings/youtrack/projects` - List YouTrack projects
- `GET /api/settings/youtrack/boards` - List YouTrack boards

### Sync Operations
- `POST /api/legacy/analyze` - Analyze tickets
- `POST /api/legacy/sync` - Sync selected tickets
- `POST /api/legacy/auto-sync/start` - Start auto-sync
- `GET /api/legacy/auto-sync/status` - Get auto-sync status

### History & Rollback
- `GET /api/sync/history` - Get sync history
- `POST /api/rollback/restore` - Restore ticket from snapshot
- `GET /api/audit/logs` - Get audit logs

---

## Key Features Explained

### Smart Ticket Mapping
- Prevents duplicates by checking existing mappings
- Updates mappings when YouTrack tickets are recreated
- Detailed logging for all mapping operations

### HTML to Markdown Conversion
Automatically converts Asana's HTML to YouTrack's Markdown:
- Headings, bold, italic, strikethrough, code blocks
- Ordered and unordered lists
- Hyperlinks and blockquotes

### Custom Interval Editor
Visual editor for auto-sync intervals with:
- Input field for interval value
- Time unit dropdown (seconds/minutes/hours)
- Live preview with validation
- Prevents intervals < 15 seconds

---

## Troubleshooting

**Backend Issues:**
- Database locked: Ensure only one backend instance is running
- API failures: Verify API credentials and base URLs

**Frontend Issues:**
- Build failures: Clear `node_modules/` and reinstall
- WebSocket issues: Check backend is running on correct port

**Sync Issues:**
- Tickets not syncing: Check column mapping configuration
- Duplicates created: Verify only one backend instance is running

---

## Development

**Run in development:**
```bash
# Backend
cd backend && go run main.go

# Frontend
cd frontend && npm start
```

**Build for production:**
```bash
# Backend
cd backend && go build -o boardsync-api main.go

# Frontend
cd frontend && npm run build
```

---

## Roadmap

- Multi-Project Support: Sync multiple Asana/YouTrack projects simultaneously
- Attachment Syncing: Sync file attachments between platforms
- Advanced Filtering: More granular control over what gets synced
- Export Capabilities: Export sync history and audit logs to CSV
- Scheduled Syncs: Schedule syncs for specific times
- Email Notifications: Get notified when syncs complete or fail

---

## Credits

**Made by Simran** with Frustration 😤
**Version:** 4.1
**Last Updated:** 2025

---

**Making Two Apps Talk to Each Other** 🚀
