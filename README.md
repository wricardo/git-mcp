# git-mcp

This project is a **Model Context Protocol (MCP) server** designed to interact with Git repositories. It provides functionality for managing and interacting with Git repositories through MCP tools.

---

## 🚀 Features

✅ **Git Repository Management**: Execute Git operations and manage repositories.  
✅ **Repository History**: View commit history and file changes over time.  
✅ **File Tracking**: Track file changes and view file history.  
✅ **Diff Operations**: View detailed file differences between commits.

---

## 📋 Requirements

- **Go** 1.23.0 or later
- **Git** installed on your system
- **Access to Git repositories**

---

## ⚙️ Setup

### 1️⃣ Install the Package
```bash
go install github.com/wricardo/git-mcp@latest
```

### 2️⃣ Configure MCP Client Settings
Add the following configuration to your MCP settings:
```json
"git-mcp": {
  "command": "git-mcp",
  "env": {
    "WORKDIR": "/path/to/your/git/repository"
  },
  "disabled": false,
  "autoApprove": []
}
```

---

## ▶️ Usage
Run the MCP server:
```bash
git-mcp
```

---

## 🛠️ Tools

### 🔹 **git-log**
Display git commit history with commit hash, author, date, and message.

#### 📌 Parameters:
- `limit` (**optional**): Number of commits to display (default: 10)

#### 📌 Example Response:
```json
[
  {
    "hash": "1234567890abcdef",
    "author": "John Doe",
    "date": "2024-03-20",
    "message": "Initial commit"
  }
]
```

---

### 🔹 **git-changed-files**
List files changed between HEAD and a specified number of commits back.

#### 📌 Parameters:
- `commits_back` (**required**): Number of commits to look back from HEAD

#### 📌 Example Response:
```json
[
  {
    "path": "README.md",
    "changeType": "modified"
  }
]
```

---

### 🔹 **git-file-diff**
View detailed file differences between commits.

#### 📌 Parameters:
- `file_path` (**required**): Path to the file to view differences
- `commits_back` (**required**): Number of commits to look back from HEAD

#### 📌 Example Response:
```json
{
  "path": "README.md",
  "changes": [
    {
      "type": "add",
      "content": "New line added",
      "lineNumber": 42
    }
  ]
}
```

---

### 🔹 **git-file-history**
View the commit history for a specific file.

#### 📌 Parameters:
- `file_path` (**required**): Path to the file to view history
- `limit` (**optional**): Number of commits to display (default: 10)

#### 📌 Example Response:
```json
[
  {
    "hash": "1234567890abcdef",
    "author": "John Doe",
    "date": "2024-03-20",
    "message": "Update README.md"
  }
]
```
