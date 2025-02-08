package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// User struct for template
type User struct {
	Username     string
	Password     string
	VisitedCount int
}

// Message struct to display chat messages with timestamp
type Message struct {
	Username  string
	Message   string
	CreatedAt string
}

func main() {
	// Open SQLite database or create it if it doesn't exist
	var err error
	db, err = sql.Open("sqlite3", "./users.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create the users and messages table if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password TEXT,
			visited_count INTEGER DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			message TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Add a sample user if the database is empty (for testing purposes)
	_, err = db.Exec(`INSERT OR IGNORE INTO users (username, password) VALUES (?, ?)`, "testuser", "password123")
	if err != nil {
		log.Fatal(err)
	}

	// Set up routes
	http.HandleFunc("/", loginPage)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerPage)
	http.HandleFunc("/register-user", registerHandler)
	http.HandleFunc("/send-message", sendMessageHandler)

	// Start the web server
	fmt.Println("Server started at :9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}

// loginPage serves the login form
func loginPage(w http.ResponseWriter, r *http.Request) {
	// Serve login form
	tmpl := template.Must(template.New("login").Parse(`
		<html>
		<body>
			<h2>Login</h2>
			<form method="post" action="/login">
				<label for="username">Username: </label>
				<input type="text" id="username" name="username" required><br><br>
				<label for="password">Password: </label>
				<input type="password" id="password" name="password" required><br><br>
				<button type="submit">Login</button>
			</form>
			<p>Don't have an account? <a href="/register">Register here</a></p>
		</body>
		</html>
	`))

	tmpl.Execute(w, nil)
}

// loginHandler handles the login and shows the public chat
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Validate user credentials
		var storedPassword string
		var visitedCount int
		err := db.QueryRow("SELECT password, visited_count FROM users WHERE username = ?", username).Scan(&storedPassword, &visitedCount)
		if err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Check if password is correct
		if storedPassword != password {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Increment the visited counter
		visitedCount++
		_, err = db.Exec("UPDATE users SET visited_count = ? WHERE username = ?", visitedCount, username)
		if err != nil {
			log.Println("Error updating visited count:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Fetch all messages
		rows, err := db.Query("SELECT username, message, created_at FROM messages ORDER BY created_at DESC")
		if err != nil {
			http.Error(w, "Error fetching messages", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var messages []Message

		for rows.Next() {
			var msg Message
			err := rows.Scan(&msg.Username, &msg.Message, &msg.CreatedAt)
			if err != nil {
				http.Error(w, "Error scanning messages", http.StatusInternalServerError)
				return
			}
			messages = append(messages, msg)
		}

		// Serve the welcome page with the chat messages
		tmpl := template.Must(template.New("welcome").Parse(`
			<html>
			<body>
				<h2>Welcome {{.Username}}!</h2>
				<p>You have visited this page {{.VisitedCount}} times.</p>

				<h3>Public Chat</h3>
				<div>
					{{range .Messages}}
						<p><strong>{{.Username}}:</strong> {{.Message}} <em>({{.CreatedAt}})</em></p>
					{{end}}
				</div>

				<form method="post" action="/send-message">
					<textarea name="message" required></textarea><br>
					<button type="submit">Send</button>
				</form>
			</body>
			</html>
		`))

		// Passing the updated user data and messages to template
		tmpl.Execute(w, struct {
			Username     string
			VisitedCount int
			Messages     []Message
		}{
			Username:     username,
			VisitedCount: visitedCount,
			Messages:     messages,
		})
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

// registerPage serves the registration form
func registerPage(w http.ResponseWriter, r *http.Request) {
	// Serve registration form
	tmpl := template.Must(template.New("register").Parse(`
		<html>
		<body>
			<h2>Register</h2>
			<form method="post" action="/register-user">
				<label for="username">Username: </label>
				<input type="text" id="username" name="username" required><br><br>
				<label for="password">Password: </label>
				<input type="password" id="password" name="password" required><br><br>
				<button type="submit">Register</button>
			</form>
		</body>
		</html>
	`))

	tmpl.Execute(w, nil)
}

// registerHandler handles the registration form submission
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Check if the username already exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if count > 0 {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}

		// Insert the new user into the database
		_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
		if err != nil {
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}

		// Redirect to login page after successful registration
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

// sendMessageHandler handles the posting of new messages
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username") // We assume the user is logged in
		message := r.FormValue("message")

		// Insert the new message into the database
		_, err := db.Exec("INSERT INTO messages (username, message) VALUES (?, ?)", username, message)
		if err != nil {
			http.Error(w, "Error sending message", http.StatusInternalServerError)
			return
		}

		// Redirect to the same page (login page) after sending the message to display it immediately
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
