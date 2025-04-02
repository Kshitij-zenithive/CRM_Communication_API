# CRM Communication API

This repository contains the CRM Communication API built using Go. This API facilitates communication management within a CRM system.

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
- [Endpoints](#endpoints)
- [Configuration](#configuration)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- Send and receive messages
- Manage communication channels
- Track communication history
- Integrate with third-party services

---

## Requirements

- Go 1.16 or higher
- Docker (optional, for containerization)
- PostgreSQL (or any supported database)

---

## Installation

### 1️⃣ Clone the Repository

```bash
git clone https://github.com/Kshitij-zenithive/CRM_Communication_API.git
cd CRM_Communication_API
```

### 2️⃣ Install Dependencies

```bash
go mod tidy
```

### 3️⃣ Setup Database

Ensure your database is running and accessible. Update the configuration file with your database credentials.

### 4️⃣ Run the Application

```bash
go run main.go
```

---

## Usage

### Running the API

Start the server using the following command:

```bash
go run main.go
```

The API will be available at `http://localhost:8080`.

---

## Endpoints

### 🔐 Authentication

- `POST /auth/login` - Login to the system
- `POST /auth/register` - Register a new user

### ✉️ Messages

- `GET /messages` - Retrieve all messages
- `POST /messages` - Send a new message

### 📢 Channels

- `GET /channels` - Retrieve all channels
- `POST /channels` - Create a new channel

---

## Configuration

Configuration settings can be found in the `config.yaml` file. Update this file with your environment-specific settings.

### Environment Variables

| Variable       | Description               |
|---------------|---------------------------|
| `DB_HOST`     | Database host              |
| `DB_USER`     | Database username          |
| `DB_PASSWORD` | Database password          |
| `DB_NAME`     | Database name              |
| `PORT`        | API server port            |

---

## Testing

### Running Tests

```bash
go test ./...
```

### Using Docker (Optional)

Build and run the application using Docker:

```bash
docker build -t crm_communication_api .
docker run -p 8080:8080 crm_communication_api
```

---

## Contributing

Contributions are welcome! Please follow these steps to contribute:

1. Fork the repository
2. Create a new branch (`git checkout -b feature-branch`)
3. Make your changes
4. Commit your changes (`git commit -am 'Add new feature'`)
5. Push to the branch (`git push origin feature-branch`)
6. Create a new Pull Request

---


🔥 Happy coding! 🚀
