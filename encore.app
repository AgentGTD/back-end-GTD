{
  "id": "backend-gtd-h6hi",
  "api": {
    "test": {
      "handlers": ["test.go"]
    },
    "health": {
      "handlers": ["health.go"]
    },
    "auth": {
      "handlers": ["auth.go"]
    },
    "tasks": {
      "handlers": ["tasks.go"]
    },
    "projects": {
      "handlers": ["projects.go"]
    },
    "nextactions": {
      "handlers": ["nextactions.go"]
    }
  },
  "secrets": {
    "GROQ_API_KEY": {
      "description": "API key for Groq AI service"
    },
    "MONGODB_URI": {
      "description": "MongoDB connection string"
    },
    "FirebaseServiceAccount": {
      "description": "Firebase service account JSON credentials"
    }
  }
}
