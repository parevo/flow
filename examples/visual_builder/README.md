# 🌌 Parevo Flow Visual Studio

**Interactive drag-and-drop workflow builder with live execution tracking**

A production-ready visual workflow editor that demonstrates all node types in Parevo Flow through a comprehensive **Employee Onboarding Journey** example.

---

## 🎯 What's This?

This is a **full-featured visual workflow builder** that lets you:
- ✅ **Design workflows visually** with drag-and-drop
- ✅ **Test all 8 node types** in a real scenario
- ✅ **Track execution live** with color-coded status indicators
- ✅ **Interact with running workflows** via Signal nodes
- ✅ **Inspect database state** through built-in API endpoints

---

## 🚀 Quick Start

### 1. Start the Server

```bash
cd examples/visual_builder
go run main.go
```

Expected output:
```
📦 In-memory SQLite database initialized with seed data.
🌟 Parevo Flow Visual Studio (SQL Edition) is running!
👉 Open: http://localhost:8080
🗄️  DB Inspect: http://localhost:8080/api/db?table=users
📋 Tables: users | badges | documents | invoices | audit_log
```

### 2. Open Your Browser

Navigate to: **http://localhost:8080**

---

## 🧪 Testing the Employee Onboarding Workflow

The pre-loaded workflow tests **all node types** in a realistic employee onboarding scenario.

### Step-by-Step Test Guide

#### 1️⃣ Deploy the Workflow

Click **▶ Deploy & Run** button

- Initial input: `user_id=2` (Zeynep Yılmaz) and `action=onboarding_started`
- Watch as nodes light up in real-time!

#### 2️⃣ First Signal: Document Upload (⏱️ Wait 3s)

After a short wait, you'll see a **yellow pulsing node** (Signal #1).

**Actions:**
- Check the **Live Execution Signals** panel (bottom right)
- Select the signal node from dropdown
- Leave Key/Value **empty**
- Click **⚡ Trigger Signal**

**What happens:** 
- Node turns green ✅
- Workflow proceeds through HTTP API call
- External API verification completes

#### 3️⃣ Second Signal: Admin Approval (Critical Decision Point)

Another **yellow pulsing node** appears.

**For SUCCESS path:**
```
Key:   status
Value: verified
```
Click **⚡ Trigger Signal**

**Result:** ✅ **Green branch** (TRUE condition)
- Employee gets verified
- Database status → `verified`
- Training phase starts
- Third signal appears for training completion

**For FAILURE path:**
```
Key:   status  
Value: rejected
```
(or any value except "verified")

**Result:** ❌ **Red branch** (FALSE condition)
- Employee suspended
- Database status → `suspended`
- Progress drops to 0%
- Workflow terminates

#### 4️⃣ Third Signal: Training Completion (Success Path Only)

If you took the success