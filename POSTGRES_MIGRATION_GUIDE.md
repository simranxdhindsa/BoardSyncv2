# PostgreSQL Migration Guide for BoardSync

**Made by Simran** 🚀

This guide will help you migrate from the current file-based database to PostgreSQL.

## Table of Contents
1. [Why PostgreSQL?](#why-postgresql)
2. [Step-by-Step Setup](#step-by-step-setup)
3. [Testing Locally](#testing-locally)
4. [Deploying to Production](#deploying-to-production)
5. [Troubleshooting](#troubleshooting)

---

## Why PostgreSQL?

### Current System (File-based)
- ❌ Data only on one computer
- ❌ Can't handle multiple users simultaneously
- ❌ Risk of data corruption
- ❌ No automatic backups
- ❌ Slow with lots of data

### New System (PostgreSQL)
- ✅ Data in cloud (accessible anywhere)
- ✅ Multiple users at once
- ✅ Automatic backups
- ✅ Much faster
- ✅ Professional-grade security
- ✅ Easy to scale

---

## Step-by-Step Setup

### Step 1: Create PostgreSQL Database on Render.com

**Time needed: 5 minutes**

1. Go to https://dashboard.render.com
2. Log in with your account
3. Click the **"New +"** button (top right)
4. Select **"PostgreSQL"**

5. Fill in the form:
   - **Name**: `boardsync-db` (or any name)
   - **Database**: Leave as default (auto-generated)
   - **User**: Leave as default (auto-generated)
   - **Region**: Choose **Oregon (US West)** (same as your backend)
   - **PostgreSQL Version**: **16** (latest)
   - **Instance Type**: Select **Free** for testing

6. Click **"Create Database"**

7. **Wait 2-3 minutes** for the database to be created

8. Once ready, you'll see the database dashboard. Look for **"Connections"** section

9. **Copy the "Internal Database URL"** - it looks like this:
   ```
   postgresql://boardsync_user:SOME_PASSWORD_HERE@dpg-xxxxx-a.oregon-postgres.render.com/boardsync
   ```

   ⚠️ **IMPORTANT**: Copy the **Internal Database URL**, not the External one!

10. **Save this URL somewhere safe** - you'll need it in the next step

---

### Step 2: Add Database URL to Render Backend

**Time needed: 2 minutes**

1. Go to https://dashboard.render.com
2. Click on your **backend service** (boardsyncv2 or similar)
3. Click **"Environment"** in the left sidebar
4. Click **"Add Environment Variable"**
5. Add:
   - **Key**: `DATABASE_URL`
   - **Value**: Paste the Internal Database URL you copied earlier
6. Click **"Save Changes"**

Your backend will automatically redeploy with the new environment variable.

---

### Step 3: Update Backend Code (CRITICAL)

**Time needed: 10 minutes**

You need to update `main.go` to use PostgreSQL instead of the file-based database.

Open: `backend/main.go`

Find this code (around line 30-40):
```go
// Initialize database
db, err := database.InitDB("./sync_app.db")
if err != nil {
    log.Fatal("Failed to initialize database:", err)
}
defer db.Close()
```

**Replace it with this:**
```go
// Initialize database
// Check if DATABASE_URL is set (use PostgreSQL)
if os.Getenv("DATABASE_URL") != "" {
    log.Println("Using PostgreSQL database")
    pgDB, err := database.InitPostgres()
    if err != nil {
        log.Fatal("Failed to initialize PostgreSQL:", err)
    }
    defer pgDB.Close()

    // TODO: Update all handlers to use pgDB instead of db
} else {
    // Fallback to file-based database for local development
    log.Println("Using file-based database (local development)")
    db, err := database.InitDB("./sync_app.db")
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    defer db.Close()
}
```

**Add this import at the top:**
```go
import (
    "os" // Add this if not already present
    // ... other imports
)
```

---

### Step 4: Test Database Connection

**Time needed: 5 minutes**

After deploying, check if the database connection works:

1. Go to Render dashboard → Your backend service
2. Click **"Logs"** tab
3. Look for these messages:
   ```
   Connecting to PostgreSQL database...
   PostgreSQL database connected successfully
   Running database migrations...
   Database migrations completed successfully
   ```

4. If you see these messages → **SUCCESS!** ✅
5. If you see errors → See [Troubleshooting](#troubleshooting) section

---

### Step 5: Verify Tables Were Created

**Time needed: 3 minutes**

1. Go to Render dashboard → Your PostgreSQL database
2. Click **"Connect"** button
3. Copy the **psql command** (looks like):
   ```bash
   PGPASSWORD=xxx psql -h dpg-xxxxx-a.oregon-postgres.render.com -U boardsync_user boardsync
   ```

4. Open your computer's terminal/command prompt
5. Paste and run the command
6. You'll see a `postgres=>` prompt

7. Type this command to see all tables:
   ```sql
   \dt
   ```

8. You should see these tables:
   - `users`
   - `user_settings`
   - `ticket_mappings`
   - `ignored_tickets`
   - `sync_operations`
   - `rollback_snapshots`
   - `audit_logs`

9. Type `\q` to exit

If you see all tables → **SUCCESS!** ✅

---

## Testing Locally

If you want to test PostgreSQL on your computer before deploying:

### Option 1: Use Docker (Easiest)

1. Install Docker Desktop from https://www.docker.com/products/docker-desktop/
2. Run this command:
   ```bash
   docker run --name boardsync-postgres -e POSTGRES_PASSWORD=test123 -e POSTGRES_DB=boardsync -p 5432:5432 -d postgres:16
   ```
3. Set environment variable:
   ```bash
   # Windows (Command Prompt)
   set DATABASE_URL=postgresql://postgres:test123@localhost:5432/boardsync?sslmode=disable

   # Windows (PowerShell)
   $env:DATABASE_URL="postgresql://postgres:test123@localhost:5432/boardsync?sslmode=disable"

   # Mac/Linux
   export DATABASE_URL=postgresql://postgres:test123@localhost:5432/boardsync?sslmode=disable
   ```
4. Run your backend:
   ```bash
   cd backend
   go run main.go
   ```

### Option 2: Install PostgreSQL Locally

1. Download from: https://www.postgresql.org/download/
2. Install with default settings
3. Remember the password you set during installation
4. Create database:
   ```bash
   createdb boardsync
   ```
5. Set environment variable (same as Option 1 above)

---

## Deploying to Production

### Current Status Checklist

Before deploying, make sure:

- [ ] PostgreSQL database created on Render ✅
- [ ] DATABASE_URL added to backend environment variables ✅
- [ ] Backend code updated to use PostgreSQL (pending)
- [ ] All handler code updated (TODO - see below)
- [ ] Tested locally (optional but recommended)

### What's Left to Do

The current implementation (`postgres.go`) has the database operations, but we need to:

1. **Update all handlers** to use PostgreSQL methods instead of file-based methods
2. **Add remaining operations** (sync operations, snapshots, audit logs)
3. **Test all endpoints** to ensure they work with PostgreSQL
4. **Migrate existing data** from JSON files to PostgreSQL

**Estimated Time**: 15-20 hours of development work

**Recommendation**: This is complex work. Consider:
- Hiring a Go developer (1-2 weeks, ~$1000-2000)
- Or taking time to learn and do it yourself (2-3 weeks part-time)

---

## Database Operations Comparison

### Before (File-based):
```go
// Get user
user, err := db.GetUserByEmail(email)
```

### After (PostgreSQL):
```go
// Get user
pgDB := database.GetPostgresDB()
user, err := pgDB.GetUserByEmail(email)
```

The method names are the same! Just need to call them on `pgDB` instead of `db`.

---

## Troubleshooting

### Error: "DATABASE_URL environment variable is not set"

**Solution**: Make sure you added the DATABASE_URL to Render environment variables (Step 2)

---

### Error: "failed to open database: ..."

**Possible causes**:
1. Wrong DATABASE_URL format
2. Database not created yet
3. Wrong credentials

**Solution**:
- Double-check the DATABASE_URL from Render
- Make sure you copied the **Internal Database URL**
- Verify the database is "Available" in Render dashboard

---

### Error: "connection refused"

**Possible causes**:
1. Database not ready yet
2. Wrong host/port
3. Firewall blocking connection

**Solution**:
- Wait 2-3 minutes for database to fully start
- Use **Internal Database URL** (not External)
- Check Render database status

---

### Error: "permission denied for table ..."

**Possible causes**:
1. User doesn't have correct permissions
2. Tables not created yet

**Solution**:
- Check that migrations ran successfully
- Look for "Database migrations completed successfully" in logs

---

### Data is empty after migration

This is expected! The PostgreSQL database is brand new and empty.

**Solutions**:
1. **Start fresh** (recommended for testing)
   - Create new user accounts
   - Re-configure settings
   - Let users start syncing again

2. **Migrate old data** (advanced)
   - Export data from JSON files
   - Write migration script
   - Import into PostgreSQL
   - **Time**: 5-10 hours of work

---

## Next Steps

### Immediate (Right Now)
1. ✅ Database created on Render
2. ✅ DATABASE_URL added to environment
3. ⏳ Update main.go to use PostgreSQL
4. ⏳ Deploy and test connection

### Short Term (This Week)
1. Update all handler files to use PostgreSQL
2. Test each endpoint thoroughly
3. Fix any bugs

### Long Term (Optional)
1. Set up automatic database backups (Render does this automatically!)
2. Monitor database size and upgrade plan if needed
3. Consider adding database pooling for better performance
4. Set up database monitoring/alerts

---

## Cost Estimate

### Free Tier (Current)
- **Database Size**: 256 MB
- **Concurrent Connections**: 97
- **Backups**: 7 days
- **Cost**: **$0/month**
- **Good for**: Testing, small projects, 1-10 users

### Paid Tier (When You Grow)
- **Starter ($7/month)**:
  - 1 GB storage
  - 97 connections
  - 14-day backups

- **Standard ($25/month)**:
  - 10 GB storage
  - 400 connections
  - 30-day backups
  - High availability

---

## Questions?

If you get stuck:
1. Check the Render logs first
2. Check this troubleshooting section
3. Search the error message on Google
4. Ask for help on Render community forums

---

## Summary

✅ **What We Did**:
- Created PostgreSQL schema (schema.sql)
- Implemented database operations (postgres.go)
- Installed PostgreSQL driver (lib/pq)
- Created migration guide (this file)

⏳ **What's Left**:
- Update main.go to switch between file-based and PostgreSQL
- Update all handlers to use PostgreSQL methods
- Test thoroughly
- Migrate existing data (optional)

🎯 **Benefits When Done**:
- Professional database
- Multiple users supported
- Automatic backups
- Much faster
- Scalable to thousands of users
- Ready for production

---

**Made by Simran with Determination 💪**
