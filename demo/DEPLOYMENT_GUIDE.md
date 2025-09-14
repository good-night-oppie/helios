# Helios Demo - 24-Hour Deployment Guide

> Complete setup instructions for showcasing the Helios Parallel Universe Engine within 24 hours

## 🎯 Deployment Timeline

### Hour 0-2: Environment Setup
- [ ] Verify system requirements
- [ ] Install Docker/Node.js
- [ ] Clone repository
- [ ] Test basic connectivity

### Hour 2-6: Build & Test
- [ ] Build Docker image
- [ ] Run local tests
- [ ] Verify all components
- [ ] Performance validation

### Hour 6-24: Production Ready
- [ ] Deploy to presentation environment
- [ ] Load test with demo scenarios
- [ ] Prepare presentation materials
- [ ] Final validation checks

## 🖥️ System Requirements

### Minimum Requirements:
- **OS**: Linux, macOS, or Windows with WSL2
- **RAM**: 4GB available (8GB recommended)
- **Disk**: 2GB free space
- **Ports**: 3000, 3001 must be available
- **Network**: Internet access for initial setup

### Software Prerequisites:
- **Docker**: 20.10+ (recommended)
- **Docker Compose**: 2.0+ (optional but recommended)
- **Git**: Latest version
- **curl**: For health checks

### Alternative (Development):
- **Node.js**: 18.0+
- **npm**: 8.0+

## 🚀 Deployment Methods

### Method 1: Docker (Production Ready)

#### Step 1: Quick Deploy
```bash
# Clone repository
git clone https://github.com/your-org/helios-demo
cd helios-demo

# One-command deployment
./scripts/run.sh
```

#### Step 2: Verify Deployment
```bash
# Check container status
docker ps | grep helios-demo

# Verify health
curl http://localhost:3001/api/stats

# View logs
docker logs -f helios-demo
```

#### Expected Output:
```json
{
  "universeCount": 0,
  "memoryUsed": 53687091200,
  "avgCommitLatency": 0,
  "memoryEfficiency": 1.0,
  "uptime": 1234
}
```

### Method 2: Docker Compose (Development)

#### Step 1: Compose Deployment
```bash
cd docker
docker-compose up -d
```

#### Step 2: Monitor Services
```bash
docker-compose ps
docker-compose logs -f
```

### Method 3: Development Mode

#### Step 1: Manual Setup
```bash
# Backend
cd backend
npm install
npm start &

# Frontend (new terminal)
cd ../frontend
npm install
npm start
```

#### Step 2: Verify Services
- Backend: http://localhost:3001/api/stats
- Frontend: http://localhost:3000

## 🔍 Validation Checklist

### Core Functionality:
- [ ] Frontend loads without errors
- [ ] Backend API responds to health checks
- [ ] WebSocket connection establishes
- [ ] Universe creation works (try 100 universes)
- [ ] Real-time stats update
- [ ] Memory usage displays correctly
- [ ] Garbage collection functions

### Performance Validation:
- [ ] VST commit latency shown (<70μs target)
- [ ] Memory efficiency calculations accurate
- [ ] Timing comparison displays properly
- [ ] Multiple visualization modes work

### User Experience:
- [ ] All control buttons function
- [ ] Responsive design works on different screens
- [ ] Animations and transitions smooth
- [ ] Error states handled gracefully

## 🎪 Demo Scenarios

### Scenario 1: EDA Chip Design (1000 Tests)
```bash
# Via API
curl -X POST http://localhost:3001/api/universes \
  -H "Content-Type: application/json" \
  -d '{"count": 1000}'

# Via UI: Set count to 1000, click "Create Universes"
```

**Expected Results:**
- Creation time: <10 seconds
- Memory usage: ~100GB (vs 50TB traditional)
- Parallel time: ~10 minutes (vs 83 hours serial)

### Scenario 2: Risk Analysis (5000 Scenarios)
```bash
# High-volume test
curl -X POST http://localhost:3001/api/universes \
  -H "Content-Type: application/json" \
  -d '{"count": 5000}'
```

**Expected Results:**
- Memory efficiency: <2x base memory
- Garbage collection: Automatic cleanup
- System stability: No memory leaks

### Scenario 3: Real-time Monitoring
1. Open multiple browser tabs
2. Create universes in one tab
3. Monitor real-time updates in others
4. Verify WebSocket synchronization

## 📊 Performance Benchmarks

### Target Metrics:
- **Container Startup**: <30 seconds
- **First Paint**: <5 seconds
- **Universe Creation**: <100ms per 100 universes
- **Memory Growth**: Linear with actual diffs, not universe count
- **API Response**: <100ms for all endpoints

### Monitoring Commands:
```bash
# System resources
docker stats helios-demo

# Memory usage
docker exec helios-demo ps aux

# Network connections
netstat -tlnp | grep :300[01]

# Disk usage
du -sh helios-demo/
```

## 🔧 Troubleshooting

### Issue: Port Already in Use

**Symptoms:**
- "EADDRINUSE: address already in use"
- Container fails to start

**Solution:**
```bash
# Find process using port
lsof -i :3000
lsof -i :3001

# Kill process
sudo kill -9 PID

# Or use different ports
docker run -p 8000:3000 -p 8001:3001 helios/demo:latest
```

### Issue: Docker Build Fails

**Symptoms:**
- Build timeouts
- "No space left on device"
- Dependency installation errors

**Solution:**
```bash
# Clean Docker cache
docker system prune -a -f

# Free up space
docker volume prune -f

# Rebuild with more memory
docker build --memory=4g -f docker/Dockerfile -t helios/demo:latest .
```

### Issue: Frontend Not Loading

**Symptoms:**
- White screen
- Console errors
- API connection failures

**Solution:**
```bash
# Check backend health
curl http://localhost:3001/api/stats

# Verify frontend build
docker exec helios-demo ls -la /app/frontend/build

# Check logs
docker logs helios-demo | grep -i error
```

### Issue: WebSocket Connection Failed

**Symptoms:**
- "Connection failed" in console
- Real-time updates not working

**Solution:**
```bash
# Check Socket.IO endpoint
curl http://localhost:3001/socket.io/?transport=polling

# Verify CORS settings
docker logs helios-demo | grep -i cors

# Test with curl
curl -X POST http://localhost:3001/api/universes \
  -H "Content-Type: application/json" \
  -d '{"count": 10}'
```

## 🌐 Endpoint Exposure for Public Access

### Quick Public Access (For Presentations)

After running the demo locally, expose it publicly using one of these methods:

#### Method 1: ngrok (Recommended for Quick Setup)
```bash
# Install ngrok (one-time setup)
curl -s https://ngrok-agent.s3.amazonaws.com/ngrok.asc | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null
echo "deb https://ngrok-agent.s3.amazonaws.com buster main" | sudo tee /etc/apt/sources.list.d/ngrok.list
sudo apt update && sudo apt install ngrok

# Get auth token from https://ngrok.com and authenticate
ngrok config add-authtoken YOUR_TOKEN

# Use the exposure script
./scripts/expose.sh ngrok
```

#### Method 2: Cloudflare Tunnels (For Professional URLs)
```bash
# Install cloudflared
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb -o cloudflared.deb
sudo dpkg -i cloudflared.deb

# Use the exposure script
./scripts/expose.sh cloudflare
```

#### Method 3: localtunnel (No Signup Required)
```bash
# Use the exposure script (installs automatically)
./scripts/expose.sh localtunnel
```

### Expected Results:
- **ngrok**: Professional random URLs (e.g., https://abc123.ngrok.io)
- **Cloudflare**: Quick tunnels or custom domains
- **localtunnel**: Simple URLs (e.g., https://helios-demo.loca.lt)

## 🌐 Production Deployment

### Cloud Deployment (AWS/GCP/Azure):

#### Step 1: VM Setup
```bash
# Launch VM with Docker pre-installed
# Minimum: 2 vCPU, 4GB RAM, 10GB disk

# Security groups: Allow inbound 3000, 3001
```

#### Step 2: Deploy to Cloud
```bash
# SSH to VM
ssh user@your-vm-ip

# Deploy demo
git clone https://github.com/your-org/helios-demo
cd helios-demo
./scripts/run.sh
```

#### Step 3: Configure Domain (Optional)
```bash
# Using nginx as reverse proxy
sudo apt install nginx

# Configure proxy to ports 3000/3001
# Set up SSL with Let's Encrypt
```

### Kubernetes Deployment:

#### Step 1: Create Manifests
```yaml
# helios-demo-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: helios-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: helios-demo
  template:
    metadata:
      labels:
        app: helios-demo
    spec:
      containers:
      - name: helios-demo
        image: helios/demo:latest
        ports:
        - containerPort: 3000
        - containerPort: 3001
---
apiVersion: v1
kind: Service
metadata:
  name: helios-demo-service
spec:
  selector:
    app: helios-demo
  ports:
  - name: frontend
    port: 3000
    targetPort: 3000
  - name: backend
    port: 3001
    targetPort: 3001
  type: LoadBalancer
```

#### Step 2: Deploy
```bash
kubectl apply -f helios-demo-deployment.yaml
kubectl get pods
kubectl get services
```

## 📋 Pre-Demo Checklist

### T-24 Hours:
- [ ] Build and test Docker image
- [ ] Verify all scripts work
- [ ] Test on target deployment environment
- [ ] Prepare backup deployment method

### T-4 Hours:
- [ ] Deploy to presentation environment
- [ ] Run full demo scenarios
- [ ] Test from audience network/devices
- [ ] Prepare contingency plans

### T-1 Hour:
- [ ] Final health check
- [ ] Clear browser cache
- [ ] Test all demo scenarios
- [ ] Have backup demo ready

### T-0 (Demo Time):
- [ ] Open browser to demo URL
- [ ] Verify connection status is green
- [ ] Test one quick universe creation
- [ ] Ready to present!

## 🎬 Presentation Setup

### Optimal Setup:
1. **Main Display**: Demo running at http://localhost:3000
2. **Secondary Display**: Slides with technical details
3. **Backup**: Screenshots/video of demo in action
4. **Audience Access**: Share demo URL for interactive experience

### Demo Flow:
1. **Introduction** (30s): Problem statement
2. **Traditional Approach** (1m): Serial execution visualization
3. **Helios Solution** (2m): Create 1000 universes live
4. **Performance Comparison** (1m): Metrics and cost analysis
5. **Q&A** (30s): Interactive exploration

## 📞 Support & Resources

### Quick Commands:
```bash
# Status check
./scripts/run.sh && curl http://localhost:3000

# Emergency restart
docker restart helios-demo

# Debug mode
docker logs -f helios-demo

# Performance check
docker stats helios-demo
```

### Emergency Contacts:
- **Technical Issues**: Check GitHub Issues
- **Demo Questions**: Refer to README.md
- **Performance Problems**: Monitor resource usage

### Backup Plans:
1. **Primary**: Docker deployment
2. **Secondary**: Development mode (if Docker fails)
3. **Tertiary**: Static screenshots/video demo
4. **Emergency**: Slides-only presentation with recorded demo

---

**Final Check**: Before your presentation, run:
```bash
curl -s http://localhost:3000 > /dev/null && echo "✅ Demo Ready!" || echo "❌ Demo Failed!"
```

Your Helios demo should now be ready to showcase the transformation from 83-hour serial execution to 5-minute parallel execution! 🚀