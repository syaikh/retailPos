# Retail POS System

A modern, real-time Point of Sale system built with Go backend, Svelte frontend, and PostgreSQL database. Features JWT authentication, WebSocket real-time updates, role-based access control (RBAC), and thermal receipt printing.

## 🚀 Features

### Core Functionality
- **User Management**: Multi-role user system (Super Admin, Admin, Manager, Cashier)
- **Product Inventory**: SKU-based product management with categories
- **Sales Transactions**: Real-time POS with cart management and receipt generation
- **Dashboard Analytics**: Sales statistics and inventory insights
- **Audit Logging**: Complete transaction and user activity tracking

### Technical Features
- **JWT Authentication**: Secure token-based authentication with refresh tokens
- **Role-Based Access Control**: Granular permissions system
- **Real-time Updates**: WebSocket broadcasting for live inventory and sales updates
- **Thermal Receipt Printing**: Browser-based receipt generation (58mm thermal format)
- **RESTful API**: Clean, documented API endpoints
- **Database Migrations**: Automated schema management
- **Comprehensive Testing**: 30+ unit and integration tests

### Security
- Password hashing with bcrypt
- CORS protection
- Rate limiting on WebSocket connections
- Origin validation for WebSocket upgrades
- Connection limits per user
- SQL injection prevention

## 🏗️ Architecture

The application follows Clean Architecture principles:

```
cmd/                    # Application entrypoints
├── server/            # Main HTTP server
├── migrate/           # Database migration tool
└── seed/              # Database seeding tool

internal/
├── domain/            # Business entities and models
├── repository/        # Data access layer (PostgreSQL)
├── auth/              # Authentication & authorization
├── middleware/        # HTTP middleware (auth, CORS)
└── delivery/http/     # HTTP handlers and routing

pkg/websocket/         # Real-time WebSocket hub
web/                   # Svelte frontend application
database/              # SQL migrations and seed data
```

## 📋 Prerequisites

- **Go 1.21+** (tested with 1.21)
- **PostgreSQL 15+** (tested with PostgreSQL 18)
- **Node.js 18+** (for frontend development)
- **Podman/Docker** (for database container, optional)

## 🚀 Quick Start

### Option 1: One-Command Setup (Recommended)

```bash
# Start PostgreSQL in Podman
podman run -d --name postgres18 -p 5432:5432 \
  -e POSTGRES_USER=devuser \
  -e POSTGRES_PASSWORD=devuser123 \
  -e POSTGRES_DB=retailpos \
  postgres:18

# Wait for database to be ready
sleep 10

# Clone and setup project
cd /path/to/projects
git clone <repository-url> retail-pos-system
cd retail-pos-system

# Build and run
go build -o retailpos ./cmd/server
go build -o seeder ./cmd/seeder
./seeder

# Set environment and run
export DB_HOST=localhost DB_PORT=5432 DB_USER=devuser \
       DB_PASSWORD=devuser123 DB_NAME=retailpos \
       JWT_SECRET=your-secret-key-change-in-production \
       PORT=8080
./retailpos
```

### Option 2: Manual Setup

#### 1. Start PostgreSQL

```bash
# Using Podman
podman run -d --name postgres18 -p 5432:5432 \
  -e POSTGRES_USER=devuser \
  -e POSTGRES_PASSWORD=devuser123 \
  -e POSTGRES_DB=retailpos \
  postgres:18

# Or using Docker
docker run -d --name postgres18 -p 5432:5432 \
  -e POSTGRES_USER=devuser \
  -e POSTGRES_PASSWORD=devuser123 \
  -e POSTGRES_DB=retailpos \
  postgres:18

# Or local PostgreSQL installation
# Create database and user manually
```

#### 2. Build the Application

```bash
# Navigate to project directory
cd retail-pos-system

# Download dependencies
go mod tidy

# Build server binary
go build -o retailpos ./cmd/server

# Build utility tools
go build -o seeder ./cmd/seeder
go build -o migrate ./cmd/migrate
```

#### 3. Initialize Database

```bash
# Run database migrations and seed data
./seeder
```

#### 4. Configure Environment

```bash
# Set required environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=devuser
export DB_PASSWORD=devuser123
export DB_NAME=retailpos
export JWT_SECRET=your-secret-key-change-in-production
export PORT=8080
export FRONTEND_URL=http://localhost:5173  # For CORS
```

#### 5. Run the Application

```bash
# Start the server
./retailpos

# Server will start on http://localhost:8080
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `devuser` | Database username |
| `DB_PASSWORD` | `devuser123` | Database password |
| `DB_NAME` | `retailpos` | Database name |
| `JWT_SECRET` | `your-secret-key-change-in-production` | JWT signing secret |
| `PORT` | `8080` | HTTP server port |
| `FRONTEND_URL` | `http://localhost:5173` | Frontend URL for CORS |

## 📖 API Documentation

### Authentication

#### Login
```http
POST /api/login
Content-Type: application/json

{
  "username": "superadmin",
  "password": "admin123"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "username": "superadmin",
    "email": "superadmin@retailpos.local",
    "role_id": 1,
    "is_active": true
  }
}
```

#### Refresh Token
```http
POST /api/refresh
Authorization: Bearer <refresh_token>
```

### Products

#### Get Products
```http
GET /api/products
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `limit`: Number of items (default: 10)
- `offset`: Pagination offset (default: 0)
- `search`: Search term
- `category_id`: Filter by category

#### Create Product
```http
POST /api/products
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "sku": "PRD-001",
  "name": "Sample Product",
  "category_id": 1,
  "price": 15000,
  "cost": 12000,
  "stock": 100,
  "stock_min": 10,
  "stock_max": 200,
  "is_active": true
}
```

#### Update Product
```http
PUT /api/products/{id}
Authorization: Bearer <access_token>
```

#### Delete Product
```http
DELETE /api/products/{id}
Authorization: Bearer <access_token>
```

### Sales

#### Create Sale
```http
POST /api/sales
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "items": [
    {
      "product_id": 1,
      "quantity": 2,
      "unit_price": 15000
    }
  ],
  "payment_method": "cash"
}
```

#### Get Sales History
```http
GET /api/sales
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `limit`, `offset`: Pagination
- `start_date`, `end_date`: Date range (YYYY-MM-DD)
- `search`: Invoice number or customer info

### Dashboard

#### Get Statistics
```http
GET /api/stats
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "data": {
    "total_sales": 1250000,
    "total_revenue": 1250000,
    "total_products": 45,
    "low_stock_count": 3
  }
}
```

### WebSocket

#### Connect to Real-time Updates
```javascript
const ws = new WebSocket('ws://localhost:8080/api/ws?token=<access_token>');
```

**Events:**
- `product_updated`: Product stock/price changes
- `user_online_count`: User connection count updates
- `sale_created`: New sales notifications

## 👥 Default Accounts

| Username | Password | Role | Permissions |
|----------|----------|------|-------------|
| `superadmin` | `admin123` | Super Admin | All permissions |
| `admin` | `admin123` | Admin | Full store management |
| `manager` | `admin123` | Manager | Reports + inventory adjustment |
| `cashier` | `admin123` | Cashier | POS transactions only |

## 🧪 Testing

### Run All Tests
```bash
# Run all test suites
go test ./...

# Run with coverage
go test -cover ./...

# Run specific packages
go test ./internal/auth/
go test ./internal/repository/
go test ./pkg/websocket/
go test ./internal/delivery/http/handler/
```

### Test Database
Tests automatically create isolated PostgreSQL databases and clean them up after execution.

## 📊 Database Schema

### Core Tables
- `users` - User accounts and authentication
- `roles` - User roles (superadmin, admin, manager, cashier)
- `permissions` - Granular permission definitions
- `role_permissions` - Role-permission mappings
- `products` - Product inventory
- `categories` - Product categories
- `sales` - Sales transactions
- `sale_items` - Individual sale line items
- `inventory_movements` - Stock adjustment history
- `audit_logs` - System activity logging

## 🔒 Security Features

- **JWT Authentication**: Stateless token-based auth with refresh tokens
- **Password Security**: bcrypt hashing with salt
- **Role-Based Access**: Granular permission system
- **Rate Limiting**: WebSocket connection rate limiting
- **CORS Protection**: Configurable origin validation
- **Connection Limits**: Maximum concurrent connections per user
- **Audit Logging**: Complete activity tracking
- **SQL Injection Prevention**: Parameterized queries

## 🚀 Production Deployment

### Build for Production
```bash
# Build optimized binary
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o retailpos ./cmd/server

# Build with version info
go build -ldflags "-X main.version=1.0.0 -X main.build=$(date -u +%Y%m%d_%H%M%S)" -o retailpos ./cmd/server
```

### Docker Deployment
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o retailpos ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/retailpos .
CMD ["./retailpos"]
```

### Systemd Service
```ini
[Unit]
Description=Retail POS System
After=network.target postgresql.service

[Service]
Type=simple
User=retailpos
WorkingDirectory=/opt/retail-pos
ExecStart=/opt/retail-pos/retailpos
Restart=always
RestartSec=5
Environment=DB_HOST=localhost
Environment=DB_PORT=5432
Environment=DB_USER=retailpos
Environment=DB_PASSWORD=secure_password
Environment=DB_NAME=retailpos
Environment=JWT_SECRET=production_secret_key
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go naming conventions
- Add tests for new features
- Update documentation
- Ensure all tests pass
- Use meaningful commit messages

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

For support and questions:
- Create an issue in the GitHub repository
- Check the documentation
- Review the test cases for usage examples

## 🔄 Version History

- **v1.0.0** - Complete POS system with authentication, real-time updates, and testing
  - JWT authentication with refresh tokens
  - Role-based access control
  - WebSocket real-time communication
  - Comprehensive test coverage
  - Thermal receipt printing
  - Audit logging

---

Built with ❤️ using Go, Svelte, PostgreSQL, and modern web technologies.</content>
<parameter name="filePath">/home/my-excellency/Projects/retail-pos-system/README.md