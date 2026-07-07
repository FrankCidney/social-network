# Social Network Platform

A robust, full-stack social networking application built with Go and React. This platform supports real-time communication, group management, and a dynamic feed.

## 🚀 Tech Stack

### Backend
- **Language:** Go (Golang)
- **Database:** PostgreSQL
- **Migrations:** golang-migrate
- **Communication:** Gorilla WebSocket for real-time features
- **Security:** Bcrypt for password hashing, UUID for unique identifiers

### Frontend
- **Framework:** Next.js with React
- **Language:** TypeScript
- **Styling:** Tailwind CSS
- **Routing:** React Router
- **Icons:** Lucide React
- **Animations:** Framer Motion

### DevOps & Tools
- **Containerization:** Docker & Docker Compose
- **Package Management:** NPM (Frontend), Go Modules (Backend)

## ✨ Key Features

- **Authentication:** Secure registration and login with session management.
- **Social Graph:** Follow/unfollow system with privacy controls.
- **Content:** Create posts with image support, comment on posts, and interact with the feed.
- **Groups:** Create and join groups, invite members, and organize group events.
- **Real-time Messaging:** Private and group chats using WebSockets.
- **Notifications:** Real-time alerts for follow requests, group invites, and events.
- **Profiles:** Customizable user profiles with public/private visibility toggles.

## 🛠️ Getting Started

### Prerequisites
- Docker and Docker Compose
- Go (optional, for local development)
- Node.js (optional, for local development)

### Running with Docker

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd social-network
   ```

2. Start the services:
   ```bash
   docker-compose up --build
   ```

3. Access the application:
   - Frontend: `http://localhost:3000`
   - Backend API: `http://localhost:8080`

## 📂 Project Structure

- `cmd/`: Entry point for the Go server.
- `internal/`: Core backend logic, including database handlers and models.
- `frontend/`: React application.
- `docs/`: Project documentation and architecture details.
- `docker-compose.yml`: Docker configuration for the entire stack.

## 📄 License

This project is licensed under the MIT License.
