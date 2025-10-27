# 🚀 Postman Automation Guide for BoardSync API

**Made for Non-Technical Users** ✨

---

## 📥 **STEP 1: Import the Collection into Postman**

### A) Open Postman
1. Download and install Postman from: https://www.postman.com/downloads/
2. Open Postman (you can skip signup)

### B) Import the Collection File
1. Click **"Import"** button (top left corner)
2. Click **"Upload Files"**
3. Navigate to: `D:\Practice\BoardSyncAPI\BoardSyncv2\`
4. Select: `BoardSync_Postman_Collection.json`
5. Click **"Open"**
6. Click **"Import"**

✅ **Done!** You should now see **"BoardSync API - Complete Collection"** in your Collections tab.

---

## 🎯 **STEP 2: Understanding the Collection**

Your collection has **14 pre-configured requests**:

| # | Request Name | What It Does |
|---|--------------|--------------|
| 1 | Health Check | Check if API is running |
| 2 | Register User | Create a new account |
| 3 | Login | Get access token (auto-saves!) |
| 4 | Get User Profile | View your account info |
| 5 | Get Settings | View current settings |
| 6 | Save Settings | Save Asana/YouTrack credentials |
| 7 | Analyze Tickets | See what needs syncing |
| 8 | Create Missing Tickets | Create tickets in YouTrack |
| 9 | Sync Mismatched Tickets | Sync changed tickets |
| 10 | Get Ticket Mappings | View all mapped tickets |
| 11 | Get Sync History | View past sync operations |
| 12 | Get Auto-Sync Status | Check auto-sync status |
| 13 | Start Auto-Sync | Enable automatic syncing |
| 14 | Stop Auto-Sync | Disable automatic syncing |

---

## 🔑 **STEP 3: How Authorization Token Works (Automatic!)**

### The Magic: Auto-Saving Token

When you run **"3. Login"**, the collection automatically:
1. Sends your email/password
2. Receives a token from the API
3. **Saves it to `{{token}}` variable** automatically
4. Uses it for all protected requests

### How It Works Behind the Scenes:

```javascript
// This script runs automatically after login
var jsonData = pm.response.json();
if (jsonData.data && jsonData.data.token) {
    pm.collectionVariables.set("token", jsonData.data.token);
    console.log('Token saved!');
}
```

### Where Is the Token Used?

All protected requests (4-14) automatically use:
```
Authorization: Bearer {{token}}
```

**You don't need to copy/paste anything!** 🎉

---

## ⚡ **STEP 4: Quick Start - First Time Setup**

### Before You Start:
1. Make sure your API server is running:
   ```bash
   cd D:\Practice\BoardSyncAPI\BoardSyncv2\backend
   asana-youtrack-sync.exe
   ```
2. Keep the server window open!

### Run These in Order:

#### 1️⃣ Health Check
- Click on **"1. Health Check"**
- Click **"Send"**
- ✅ Should see: `"status": "success"`

#### 2️⃣ Register or Login
**First Time? Register:**
- Click **"2. Register User"**
- Edit the Body if you want different credentials
- Click **"Send"**
- ✅ Token auto-saved!

**Already Registered? Login:**
- Click **"3. Login"**
- Click **"Send"**
- ✅ Token auto-saved!

#### 3️⃣ Save Your Settings
- Click **"6. Save Settings"**
- **IMPORTANT:** Edit the Body with your actual API tokens:
  ```json
  {
    "asana_pat": "YOUR_ACTUAL_ASANA_TOKEN",
    "youtrack_base_url": "https://simran.youtrack.cloud",
    "youtrack_token": "YOUR_ACTUAL_YOUTRACK_TOKEN",
    "asana_project_id": "1211341745333034",
    "youtrack_project_id": "ARD",
    "youtrack_board_id": "YOUR_BOARD_ID"
  }
  ```
- Click **"Send"**
- ✅ Settings saved!

#### 4️⃣ Test It Works
- Click **"7. Analyze Tickets"**
- Click **"Send"**
- ✅ Should see list of tickets!

---

## 🤖 **STEP 5: AUTOMATION - Run All Requests Automatically**

### Method 1: Collection Runner (Simple)

1. Click **"..."** next to collection name
2. Select **"Run collection"**
3. You'll see all 14 requests listed
4. **Uncheck requests you don't want** (like Register if already registered)
5. Set options:
   - **Iterations:** `1` (how many times to run the whole collection)
   - **Delay:** `1000` (1 second delay between requests)
6. Click **"Run BoardSync API"**

**Recommended Order for Daily Sync:**
```
✓ Health Check
✓ Login (to refresh token)
✓ Get Settings (verify settings)
✓ Analyze Tickets (see what needs sync)
✓ Create Missing Tickets (create new ones)
✓ Sync Mismatched Tickets (sync changed ones)
```

### Method 2: Run Individual Folder

You can organize requests into folders:
1. Right-click collection → **"Add folder"**
2. Name it: "Daily Sync Workflow"
3. Drag these requests into it:
   - Login
   - Analyze Tickets
   - Create Missing Tickets
   - Sync Mismatched Tickets
4. Right-click folder → **"Run folder"**

---

## 🔄 **STEP 6: AUTOMATION - Postman Monitor (Cloud Automation)**

**This runs your API calls automatically every hour/day - even when your computer is off!**

### A) Upgrade to Postman Account (Free)
1. Sign up for free at: https://www.postman.com/
2. Login to Postman app

### B) Create a Monitor

1. Click on your collection
2. Click **"..."** → **"Monitor Collection"**
3. Fill in:
   - **Monitor Name:** "BoardSync Daily Sync"
   - **Environment:** None (we're using collection variables)
   - **Frequency:**
     - Every hour
     - Every 6 hours
     - Every 12 hours
     - **Every day at specific time** ← RECOMMENDED
   - **Timezone:** Select your timezone
4. Click **"Create Monitor"**

### C) Monitor Settings

**Important:** Monitors run in the cloud, so your local API at `localhost:8080` won't work!

**Two Solutions:**

#### Solution 1: Deploy Your API to Cloud (Recommended)
1. Deploy your backend to Render.com / Railway / Heroku
2. Get your cloud URL (e.g., `https://boardsync.onrender.com`)
3. Update collection variable:
   - `base_url` → `https://boardsync.onrender.com`

#### Solution 2: Use Postman CLI (Local)
1. Install Postman CLI: https://learning.postman.com/docs/postman-cli/postman-cli-overview/
2. Run on your local machine:
   ```bash
   postman collection run "BoardSync API" --schedule "0 9 * * *"
   ```
   This runs daily at 9 AM.

---

## ⏰ **STEP 7: AUTOMATION - Windows Task Scheduler (Local Automation)**

**Run Postman automatically on your Windows PC at scheduled times.**

### A) Create a Batch Script

1. Create a new file: `D:\Practice\BoardSyncAPI\BoardSyncv2\run_postman_sync.bat`
2. Add this content:
   ```batch
   @echo off
   echo Starting BoardSync API...
   start /B "BoardSync API" "D:\Practice\BoardSyncAPI\BoardSyncv2\backend\asana-youtrack-sync.exe"

   echo Waiting for API to start...
   timeout /t 5 /nobreak

   echo Running Postman collection...
   newman run "D:\Practice\BoardSyncAPI\BoardSyncv2\BoardSync_Postman_Collection.json" --reporters cli,json

   echo Sync complete!
   pause
   ```

### B) Install Newman (Postman CLI)

1. Install Node.js from: https://nodejs.org/
2. Open Command Prompt
3. Run:
   ```bash
   npm install -g newman
   ```

### C) Setup Windows Task Scheduler

1. Press `Win + R`, type `taskschd.msc`, press Enter
2. Click **"Create Basic Task"**
3. **Name:** "BoardSync Daily Sync"
4. **Trigger:** Daily
5. **Time:** 9:00 AM (or your preferred time)
6. **Action:** Start a program
7. **Program:** `D:\Practice\BoardSyncAPI\BoardSyncv2\run_postman_sync.bat`
8. Click **"Finish"**

✅ **Done!** Your sync will run automatically every day at 9 AM!

---

## 📊 **STEP 8: View Automation Results**

### In Postman Console:
1. Click **"Console"** button (bottom left)
2. You'll see all logged messages:
   ```
   ✅ API is running!
   ✅ Login successful! Token saved automatically.
   📊 Analysis Results:
     Missing in YouTrack: 5
     Mismatched: 3
   ✅ Tickets created successfully!
   ```

### In Collection Runner:
1. After running, you'll see:
   - ✅ Passed tests (green)
   - ❌ Failed tests (red)
   - Response times
   - Status codes

### In Monitor Dashboard (Cloud):
1. Go to: https://go.postman.co/monitors
2. Click on your monitor
3. View:
   - Run history
   - Success/failure rates
   - Response times
   - Error logs

---

## 🎨 **STEP 9: Customize for Your Workflow**

### Add Environment Variables

Instead of collection variables, you can use environments for multiple setups:

1. Click **"Environments"** (left sidebar)
2. Create **"Development"** environment:
   - `base_url` = `http://localhost:8080`
   - `token` = (leave empty)

3. Create **"Production"** environment:
   - `base_url` = `https://your-production-url.com`
   - `token` = (leave empty)

4. Switch environments from dropdown (top right)

### Create Custom Workflows

**Example: Morning Sync Workflow**
1. Create folder: "Morning Sync"
2. Add:
   - Login
   - Analyze Tickets
   - Create Missing Tickets
   - Sync Mismatched Tickets

**Example: Evening Report Workflow**
1. Create folder: "Evening Report"
2. Add:
   - Login
   - Get Sync History
   - Get Ticket Mappings
   - Get Auto-Sync Status

---

## 🔧 **STEP 10: Troubleshooting**

### ❌ "Error: connect ECONNREFUSED"
**Problem:** API server is not running

**Solution:**
```bash
cd D:\Practice\BoardSyncAPI\BoardSyncv2\backend
asana-youtrack-sync.exe
```

### ❌ "Authentication required"
**Problem:** Token expired or not set

**Solution:**
1. Run **"3. Login"** again
2. Check Console to see if token was saved
3. Verify `{{token}}` variable has a value

### ❌ "Invalid credentials"
**Problem:** Wrong email/password

**Solution:**
1. Check your credentials in Login request body
2. Or run Register to create new account

### ❌ Monitor not working
**Problem:** Monitor runs in cloud, can't reach localhost

**Solution:**
- Deploy API to cloud (Render/Railway)
- Or use Newman + Task Scheduler for local automation

---

## 📝 **STEP 11: Best Practices**

### Daily Sync Routine:
```
1. Health Check      → Verify API is up
2. Login             → Get fresh token
3. Analyze           → See what needs sync
4. Create            → Create missing tickets
5. Sync              → Sync mismatched tickets
6. Get History       → View what was done
```

### Weekly Maintenance:
```
1. Get Ticket Mappings → Review all mappings
2. Get Sync History    → Check for errors
3. Update Settings     → Adjust if needed
```

### Security Tips:
1. **Never share your collection** with tokens in it
2. **Use environment variables** for sensitive data
3. **Rotate tokens regularly** (change passwords)
4. **Don't commit tokens** to Git

---

## 🎯 **Quick Reference**

### Collection Variables:
- `{{base_url}}` → API server URL (http://localhost:8080)
- `{{token}}` → Auto-saved authentication token

### Auto-saved Token After:
- ✅ Registration (Request #2)
- ✅ Login (Request #3)

### Protected Endpoints (Need Token):
- Requests #4 through #14

### Public Endpoints (No Token):
- Request #1 (Health Check)
- Request #2 (Register)
- Request #3 (Login)

---

## 🚀 **Next Steps**

1. ✅ Import collection
2. ✅ Start API server
3. ✅ Run Health Check
4. ✅ Login (token auto-saves!)
5. ✅ Save your settings
6. ✅ Run analyze to test
7. ✅ Set up automation (Collection Runner or Monitor)

---

## 📞 **Need Help?**

- Check Postman Console for detailed logs
- View Collection documentation (click on collection → View documentation)
- Test endpoints one by one to isolate issues

---

**Made with ❤️ for BoardSync**
**Last Updated:** October 2025