# URLInsight — Back-end Requirements

## **Tech Stack & Tooling**

### Core Technologies
- **Go (Golang)** - Primary programming language
- **Gin3. **Current Analysis Features*** - Web framework for building REST APIs
- **GORM ORM** - Object-Relational Mapping for database operations
- **MySQL** - Primary database for data persistence
- **Go Modules** (`go.mod` + `go.sum`) - Dependency management

### Authentication & Security
- **JWT (JSON Web Tokens)** - Stateless authentication
- **bcrypt** - Password hashing
- **Token Blacklisting** - Secure logout mechanism

### Documentation & Development
- **Swagger/OpenAPI** - API documentation
- **Docker** - Containerization (optional)
- **VS Code Tasks** - Build automation

### Testing & Quality
- **Go Testing** - Unit and integration tests
- **Testify** - Testing assertions and mocks
- **Linting** - Code quality tools

1. **API Endpoints**
   - **Authentication**
     - `POST /auth/register` – register a new user
       - Request: `{username: string, email: string, password: string}`
       - Response: `{token: string, user: User}`
     - `POST /auth/login` – user login
       - Request: `{email: string, password: string}`
       - Response: `{token: string}`
     - `POST /auth/logout` – invalidate JWT token (requires Bearer token)
       - Response: `{message: string}`

   - **Health & Status**
     - `GET /` – home endpoint, server status check
       - Response: `{message: string, service: string, status: string}`
     - `GET /health` – health check including database connectivity
       - Response: `{status: string, database: string, service: string}`

   - **URLs Management** (requires Bearer token authentication)
     - `GET /api/urls` – list all URLs for authenticated user
       - Response: `{message: string, urls: URL[]}`
     - `POST /api/urls` – add a new URL for analysis
       - Request: `{url: string}`
       - Response: `{message: string, id: number}`
     - `GET /api/urls/:id` – get specific URL details by ID
       - Response: `{message: string, id: string, url: string}`
     - `PUT /api/urls/:id` – update a specific URL by ID
       - Request: `{url: string, status: string}`
       - Response: `{message: string, id: string}`

   - **Documentation**
     - `GET /swagger/*any` – Swagger API documentation



### Model Structures & Relationships

#### User Model
| Field     | Type           | JSON             | GORM Tags                                | Description                        |
| --------- | -------------- | ---------------- | ---------------------------------------- | ---------------------------------- |
| ID        | uint           | `id`             | `primaryKey`                             | Primary key                        |
| Username  | string         | `username`       | `type:varchar(255);uniqueIndex;not null` | Unique username                    |
| Email     | string         | `email`          | `type:varchar(255);uniqueIndex;not null` | Unique email address               |
| Password  | string         | `-`              | `type:varchar(255);not null`             | Hashed password (hidden from JSON) |
| CreatedAt | time.Time      | `created_at`     | -                                        | Creation timestamp                 |
| UpdatedAt | time.Time      | `updated_at`     | -                                        | Last update timestamp              |
| DeletedAt | gorm.DeletedAt | `-`              | `index`                                  | Soft delete timestamp              |
| URLs      | []URL          | `urls,omitempty` | `foreignKey:UserID`                      | Related URLs                       |

#### URL Model
| Field           | Type             | JSON                         | GORM Tags                                  | Description                |
| --------------- | ---------------- | ---------------------------- | ------------------------------------------ | -------------------------- |
| ID              | uint             | `id`                         | `primaryKey`                               | Primary key                |
| UserID          | uint             | `user_id`                    | `not null;index`                           | Foreign key to users table |
| URL             | string           | `url`                        | `type:varchar(2048);not null`              | URL to be analyzed         |
| Title           | string           | `title`                      | `type:varchar(500)`                        | Page title                 |
| Status          | string           | `status`                     | `type:varchar(50);default:'pending';index` | Analysis status            |
| CreatedAt       | time.Time        | `created_at`                 | -                                          | Creation timestamp         |
| UpdatedAt       | time.Time        | `updated_at`                 | -                                          | Last update timestamp      |
| DeletedAt       | gorm.DeletedAt   | `deleted_at,omitempty`       | `index`                                    | Soft delete timestamp      |
| User            | User             | `user,omitempty`             | `foreignKey:UserID`                        | Related user               |
| AnalysisResults | []AnalysisResult | `analysis_results,omitempty` | `foreignKey:URLID`                         | Related analysis results   |
| Links           | []Link           | `links,omitempty`            | `foreignKey:URLID`                         | Related links              |

#### AnalysisResult Model
| Field        | Type           | JSON                   | GORM Tags           | Description                   |
| ------------ | -------------- | ---------------------- | ------------------- | ----------------------------- |
| ID           | uint           | `id`                   | `primaryKey`        | Primary key                   |
| URLID        | uint           | `url_id`               | `not null;index`    | Foreign key to urls table     |
| Title        | string         | `title`                | `type:varchar(500)` | Page title                    |
| Description  | string         | `description`          | `type:text`         | Meta description              |
| Keywords     | string         | `keywords`             | `type:text`         | Meta keywords                 |
| StatusCode   | int            | `status_code`          | -                   | HTTP status code              |
| ResponseTime | int            | `response_time`        | -                   | Response time in milliseconds |
| ContentType  | string         | `content_type`         | `type:varchar(100)` | Content MIME type             |
| ContentSize  | int            | `content_size`         | -                   | Content size in bytes         |
| CreatedAt    | time.Time      | `created_at`           | -                   | Creation timestamp            |
| UpdatedAt    | time.Time      | `updated_at`           | -                   | Last update timestamp         |
| DeletedAt    | gorm.DeletedAt | `deleted_at,omitempty` | `index`             | Soft delete timestamp         |
| URL          | URL            | `url,omitempty`        | `foreignKey:URLID`  | Related URL                   |

#### Link Model
| Field      | Type           | JSON                   | GORM Tags                     | Description               |
| ---------- | -------------- | ---------------------- | ----------------------------- | ------------------------- |
| ID         | uint           | `id`                   | `primaryKey`                  | Primary key               |
| URLID      | uint           | `url_id`               | `not null;index`              | Foreign key to urls table |
| URL        | string         | `url`                  | `type:varchar(2048);not null` | Link URL                  |
| Text       | string         | `text`                 | `type:varchar(500)`           | Link text/anchor text     |
| IsExternal | bool           | `is_external`          | `default:false;index`         | Whether link is external  |
| IsWorking  | bool           | `is_working`           | `default:true;index`          | Whether link is working   |
| CreatedAt  | time.Time      | `created_at`           | -                             | Creation timestamp        |
| UpdatedAt  | time.Time      | `updated_at`           | -                             | Last update timestamp     |
| DeletedAt  | gorm.DeletedAt | `deleted_at,omitempty` | `index`                       | Soft delete timestamp     |
| ParentURL  | URL            | `parent_url,omitempty` | `foreignKey:URLID`            | Related parent URL        |

#### BlacklistedToken Model
| Field     | Type           | JSON         | GORM Tags                                | Description                    |
| --------- | -------------- | ------------ | ---------------------------------------- | ------------------------------ |
| ID        | uint           | `-`          | `primaryKey`                             | Primary key (hidden from JSON) |
| JTI       | string         | `jti`        | `uniqueIndex;type:varchar(255);not null` | JWT ID (unique)                |
| ExpiresAt | time.Time      | `expires_at` | `index;not null`                         | Token expiration time          |
| CreatedAt | time.Time      | `created_at` | `autoCreateTime`                         | Creation timestamp             |
| DeletedAt | gorm.DeletedAt | `-`          | `index`                                  | Soft delete timestamp          |

3. **Current Analysis Features**
   - User authentication with JWT tokens
   - URL storage and management per user
   - Basic analysis results storage (title, description, keywords, status_code, response_time, content_type, content_size)
   - Link tracking (external/internal classification, working status)
   - Token blacklisting for secure logout

   **Planned Analysis Logic** (to be implemented):
   - Fetch the page HTML
   - Detect HTML version (`<!DOCTYPE html>` vs. HTML4, etc.)
   - Extract `<title>` text
   - Count heading tags (`<h1>`–`<h6>`)
   - Classify and record internal vs. external links
   - Identify broken links (HTTP 4xx/5xx)
   - Detect login forms (`<form>` with `<input type="password">`)

4. **API Endpoints**
   - **Auth Middleware** (Bearer token or JWT)
   - `POST /api/urls` – enqueue a new URL
   - `PATCH /api/urls/:id/start` – start processing
   - `PATCH /api/urls/:id/stop` – stop processing
   - `GET /api/urls` – list URLs with pagination, sorting, filtering
   - `GET /api/urls/:id` – fetch a single URL’s metadata
   - `GET /api/urls/:id/results` – full analysis result (incl. link list)
   - `DELETE /api/urls/:id` – remove a URL and its data

3. **Real-Time Status Updates**
   - Update the URL’s `status` field as it moves through the crawl pipeline
   - Expose via polling or push (SSE/WebSocket)

4. **Error Handling & Reporting**
   - Centralized JSON error responses
   - Graceful panic recovery (Gin’s Recovery middleware)

5. **Reproducible Builds & Migrations**
   - `go.mod` & `go.sum` checked into Git
   - Auto-migrations via GORM or file-based migrations (e.g., `golang-migrate`)
   - Connection pool tuning

6. **Testing**
   - Unit tests for parsing/analysis logic (HTML version detection, link classification)
   - (Optional) Integration tests for API endpoints

7. **Documentation & Setup**
   - `README.md` with:
     1. Clone instructions
     2. Environment variables (`.env.example`)
     3. How to run migrations
     4. How to build & start the server
