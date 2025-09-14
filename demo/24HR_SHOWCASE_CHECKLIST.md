# Helios Demo - 24-Hour Showcase Checklist

> Complete step-by-step guide to deploy and showcase the Helios Parallel Universe Engine demo within 24 hours

## ⏱️ Timeline Overview

**Hours 0-2: Environment Setup**
**Hours 2-6: Local Deployment & Testing**
**Hours 6-24: Public Exposure & Presentation Prep**

---

## 📋 Hour-by-Hour Action Plan

### Hours 0-2: Environment Setup ✅

#### Prerequisites Check (30 minutes)
- [ ] **Docker Installation**
  ```bash
  docker --version  # Should be 20.10+
  docker ps         # Should show running containers list
  ```

- [ ] **System Resources**
  ```bash
  free -h          # Check available RAM (need 2GB+)
  df -h            # Check disk space (need 2GB+)
  ```

- [ ] **Port Availability**
  ```bash
  lsof -i :3000    # Should show nothing (port available)
  lsof -i :3001    # Should show nothing (port available)
  ```

#### Repository Setup (30 minutes)
```bash
# Clone the repository
git clone https://github.com/your-org/helios-demo
cd helios-demo

# Verify all scripts are executable
ls -la scripts/   # Should show executable permissions
```

---

### Hours 2-6: Local Deployment & Testing ✅

#### Core Deployment (5 minutes)
```bash
# One-command deployment
./scripts/run.sh
```

**Expected Output:**
```
🌌 Starting Helios Parallel Universe Engine...
✅ Backend is healthy!
🎉 Helios Demo is running!
🌐 Open your browser and visit: http://localhost:3000
```

#### Verification Tests (15 minutes)

- [ ] **Backend Health Check**
  ```bash
  curl http://localhost:3001/api/stats
  ```
  **Expected:** JSON with `universeCount: 0` and system metrics

- [ ] **Frontend Loading**
  - Open http://localhost:3000
  - Should show Helios demo interface
  - No console errors in browser dev tools

- [ ] **Core Functionality Test**
  1. Click "Create 100 Universes"
  2. Watch real-time creation progress
  3. Verify memory usage displays
  4. Check VST commit latency shows <70μs

- [ ] **WebSocket Connection Test**
  - Open browser dev tools → Network → WS
  - Should show active Socket.IO connection
  - Real-time stats should update automatically

#### Performance Validation (30 minutes)

- [ ] **Universe Creation Benchmarks**
  ```bash
  # Test different scales
  # 10 universes: <1 second
  # 100 universes: <5 seconds
  # 1000 universes: <10 seconds
  ```

- [ ] **Memory Efficiency Check**
  ```bash
  docker stats helios-demo
  ```
  - CPU usage should be <50% during creation
  - Memory should grow linearly with actual changes, not universe count

- [ ] **API Response Times**
  ```bash
  # All API endpoints should respond <100ms
  time curl http://localhost:3001/api/stats
  time curl http://localhost:3001/api/universes
  ```

#### Troubleshooting Common Issues

**Issue: Port conflicts**
```bash
# Solution
docker stop $(docker ps -q)  # Stop all containers
sudo pkill -f :3000          # Kill processes on port 3000
./scripts/run.sh              # Restart
```

**Issue: Frontend not loading**
```bash
# Solution
docker logs helios-demo | grep -i error  # Check logs
docker restart helios-demo               # Restart container
```

---

### Hours 6-24: Public Exposure & Presentation Prep ✅

#### Choose Exposure Method (15 minutes)

**For Quick Setup (Recommended):**
```bash
# Install ngrok
curl -s https://ngrok-agent.s3.amazonaws.com/ngrok.asc | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null
echo "deb https://ngrok-agent.s3.amazonaws.com buster main" | sudo tee /etc/apt/sources.list.d/ngrok.list
sudo apt update && sudo apt install ngrok

# Get free account and auth token from https://ngrok.com
ngrok config add-authtoken YOUR_TOKEN

# Expose demo publicly
./scripts/expose.sh ngrok
```

**For Professional URLs:**
```bash
# Use Cloudflare Tunnels
./scripts/expose.sh cloudflare
```

**For No-Signup Option:**
```bash
# Use localtunnel
./scripts/expose.sh localtunnel
```

#### Public Access Verification (10 minutes)

- [ ] **External Access Test**
  - Share the public URL with a colleague/friend
  - Verify they can access the demo
  - Test real-time synchronization (both browsers should update simultaneously)

- [ ] **Mobile Compatibility**
  - Test the public URL on mobile devices
  - Verify responsive design works
  - Check that all controls are accessible

#### Demo Preparation (2-4 hours)

- [ ] **Practice Demo Script**
  - Follow the 5-minute presentation flow in QUICKSTART.md
  - Time each section: Intro (1min) → Traditional (1min) → Helios (2min) → Impact (1min)
  - Practice seamless transitions between sections

- [ ] **Prepare Demo Scenarios**
  - **EDA Scenario**: Create 1000 universes, show timing comparison
  - **Finance Scenario**: 5000 risk scenarios, show memory efficiency
  - **Interactive**: Let audience create universes via shared URL

- [ ] **Backup Preparations**
  ```bash
  # Take screenshots of key demo states
  # Record video of successful demo run
  # Prepare static slides with performance numbers
  ```

#### Presentation Environment Setup

- [ ] **Display Configuration**
  - Primary screen: Demo at public URL
  - Secondary screen: Presentation slides with technical details
  - Have browser bookmarked to public demo URL

- [ ] **Audience Interaction Setup**
  - Share public demo URL via QR code or short link
  - Test multiple simultaneous users
  - Verify real-time synchronization works

- [ ] **Emergency Backup Plans**
  - Local demo screenshots in `/demo/assets/screenshots/`
  - Recorded demo video ready to play
  - Static presentation slides with key metrics

---

## 🎯 Demo Success Metrics

### Technical Performance
- [ ] Universe creation: <10 seconds for 1000 universes
- [ ] VST commit latency: <70μs consistently displayed
- [ ] Memory efficiency: Shows dramatic difference (50GB vs 50TB)
- [ ] Real-time updates: All connected browsers sync within 1 second

### Audience Engagement
- [ ] Public URL accessible from any device
- [ ] Interactive elements working (audience can create universes)
- [ ] Visual impact: Timing comparison shows 83 hours → 5 minutes
- [ ] Professional presentation: No errors or crashes during demo

### Presentation Quality
- [ ] Demo flows smoothly through all 5 phases
- [ ] Key metrics are prominently displayed
- [ ] Audience can interact and see real-time results
- [ ] Technical credibility maintained throughout

---

## 🚨 Final Pre-Demo Checklist (T-15 minutes)

```bash
# 1. Verify local demo is running
curl -s http://localhost:3000 > /dev/null && echo "✅ Local Demo Ready" || echo "❌ Local Demo Failed"

# 2. Verify public URL is accessible
curl -s YOUR_PUBLIC_URL > /dev/null && echo "✅ Public URL Ready" || echo "❌ Public URL Failed"

# 3. Test universe creation
# Open public URL → Create 10 universes → Verify completion

# 4. Check real-time sync
# Open public URL in two browser tabs → Create universes in one → Verify both update

# 5. Verify key metrics display
# VST latency showing <70μs
# Memory usage displaying correctly
# Timing comparison showing dramatic improvement
```

---

## 📞 Emergency Support

**If Docker fails:**
```bash
./scripts/dev.sh  # Fall back to development mode
```

**If public exposure fails:**
- Use screenshots from `/demo/assets/screenshots/`
- Play recorded demo video
- Present slides with static metrics

**If complete failure:**
- Have backup presentation ready
- Focus on conceptual explanation
- Use performance numbers from documentation

---

## 🎉 Success Indicators

**You're ready for showcase when:**
- ✅ Demo loads in <5 seconds from public URL
- ✅ Universe creation works reliably
- ✅ All visualizations render smoothly
- ✅ Key performance metrics display correctly
- ✅ Multiple users can interact simultaneously
- ✅ You can smoothly present the 5-minute demo script

**Total Setup Time: 4-8 hours (including practice)**
**Presentation Time: 5 minutes + Q&A**
**Impact: Transform 83 hours to 5 minutes with live demonstration! 🚀**