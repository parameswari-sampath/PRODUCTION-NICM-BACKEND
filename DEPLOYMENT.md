# SmartMCQ Deployment Guide

Complete guide to deploy SmartMCQ backend with automatic SSL certificates.

## 🚀 Features

- ✅ Automatic SSL certificates via Let's Encrypt
- ✅ Auto-renewal of certificates
- ✅ PostgreSQL database included
- ✅ Nginx reverse proxy
- ✅ Production-ready setup

## 📋 Prerequisites

1. **VPS Server** with:
   - Ubuntu 20.04+ / Debian 11+
   - Docker & Docker Compose installed
   - Ports 80 and 443 open

2. **Domain DNS Setup**:
   - Point `api.smart-mcq.com` A record to your VPS IP
   - Point `nicm.smart-mcq.com` A record to your frontend server IP

## 🛠️ Installation Steps

### 1. Install Docker & Docker Compose

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo apt install docker-compose-plugin -y

# Add user to docker group
sudo usermod -aG docker $USER

# Logout and login again for group changes to take effect
```

### 2. Clone/Upload Your Project

```bash
# Create project directory
mkdir -p ~/smartmcq
cd ~/smartmcq

# Upload your project files here
# Or use git clone if your repo is ready
```

### 3. Configure Environment Variables

```bash
# Copy production env file
cp .env.production .env

# Edit the .env file with your actual values
nano .env
```

**IMPORTANT: Update these values in `.env`:**

```env
# PostgreSQL - Change this password!
POSTGRES_USER=smartmcq_user
POSTGRES_PASSWORD=YOUR_SECURE_PASSWORD_HERE
POSTGRES_DB=smartmcq

# Email (keep your ZeptoMail credentials)
ZEPTO_API_KEY=your_actual_key
ZEPTO_FROM_EMAIL=no-reply@smart-mcq.com
ZEPTO_FROM_NAME=SmartMCQ

# URLs (already configured)
FRONTEND_URL=https://nicm.smart-mcq.com
BASE_URL=https://api.smart-mcq.com
```

### 4. Update docker-compose.yml

Edit `docker-compose.yml` and update the email address:

```yaml
environment:
  - LETSENCRYPT_EMAIL=your-email@example.com  # Change this to your real email!
```

And in the `acme-companion` service:

```yaml
environment:
  - DEFAULT_EMAIL=your-email@example.com  # Change this to your real email!
```

### 5. Verify DNS is Propagated

```bash
# Check if DNS is pointing to your server
ping api.smart-mcq.com

# Should show your VPS IP address
```

**⚠️ IMPORTANT:** DNS must be pointing to your VPS before starting. Let's Encrypt will fail if DNS is not configured.

### 6. Deploy the Application

```bash
# Make sure you're in the project directory
cd ~/smartmcq

# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f
```

### 7. Verify SSL Certificate

Wait 2-3 minutes for Let's Encrypt to issue certificates. Then check:

```bash
# Check if certificates are created
docker-compose exec acme-companion ls -la /etc/nginx/certs/

# You should see files like:
# api.smart-mcq.com.crt
# api.smart-mcq.com.key

# Test the endpoint
curl https://api.smart-mcq.com/health
```

### 8. Check Database

```bash
# Access PostgreSQL
docker-compose exec postgres psql -U smartmcq_user -d smartmcq

# Check if migrations ran
\dt

# Exit psql
\q
```

## 📊 Management Commands

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f backend
docker-compose logs -f nginx-proxy
docker-compose logs -f postgres
```

### Restart Services

```bash
# Restart all
docker-compose restart

# Restart specific service
docker-compose restart backend
```

### Stop/Start

```bash
# Stop all
docker-compose down

# Start all
docker-compose up -d
```

### Database Backup

```bash
# Create backup
docker-compose exec postgres pg_dump -U smartmcq_user smartmcq > backup_$(date +%Y%m%d).sql

# Restore backup
docker-compose exec -T postgres psql -U smartmcq_user smartmcq < backup_20251006.sql
```

### Update Application

```bash
# Pull latest changes
git pull

# Rebuild and restart
docker-compose up -d --build backend
```

## 🔒 SSL Certificate Renewal

Certificates auto-renew via acme-companion. To manually check:

```bash
# Check certificate expiry
docker-compose exec nginx-proxy openssl x509 -in /etc/nginx/certs/api.smart-mcq.com.crt -noout -dates

# Force renewal (if needed)
docker-compose restart acme-companion
```

## 🐛 Troubleshooting

### SSL Certificate Not Issued

```bash
# Check acme-companion logs
docker-compose logs acme-companion

# Common issues:
# 1. DNS not pointing to server - verify with: dig api.smart-mcq.com
# 2. Ports 80/443 blocked - check firewall
# 3. Rate limit - Let's Encrypt has limits, wait 1 hour
```

### Backend Not Responding

```bash
# Check backend logs
docker-compose logs backend

# Check if backend is running
docker-compose ps

# Check database connection
docker-compose exec backend env | grep DATABASE_URL
```

### Database Connection Issues

```bash
# Check if postgres is healthy
docker-compose ps postgres

# Check postgres logs
docker-compose logs postgres

# Test connection
docker-compose exec postgres pg_isready -U smartmcq_user
```

## 🔥 Firewall Configuration

```bash
# Allow required ports
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 22/tcp  # SSH
sudo ufw enable
```

## 📈 Monitoring

### Check Service Health

```bash
# Quick status check
docker-compose ps

# Resource usage
docker stats
```

### Check API Health

```bash
curl https://api.smart-mcq.com/health
```

## 🔄 Migration from Neon DB

If you want to migrate from your current Neon DB to local PostgreSQL:

```bash
# 1. Export from Neon (on your local machine)
pg_dump "postgresql://neondb_owner:npg_xxx@ep-winter-bar-xxx.neon.tech/neondb" > neon_export.sql

# 2. Upload to VPS
scp neon_export.sql user@your-vps-ip:~/smartmcq/

# 3. Import to local PostgreSQL
docker-compose exec -T postgres psql -U smartmcq_user smartmcq < neon_export.sql
```

## 📝 Important Notes

1. **Email Address**: Update `LETSENCRYPT_EMAIL` in docker-compose.yml
2. **Database Password**: Change the default password in `.env`
3. **DNS First**: Ensure DNS is configured before deployment
4. **Backup**: Regular database backups recommended
5. **Security**: Keep your `.env` file secure and never commit to git

## 🎉 Success!

Your application should now be running at:
- **Backend API**: https://api.smart-mcq.com
- **Frontend**: https://nicm.smart-mcq.com (deploy separately)

SSL certificates will auto-renew every 90 days automatically! 🔒
