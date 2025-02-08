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

func main() {
	// Open SQLite database or create it if it doesn't exist
	var err error
	db, err = sql.Open("sqlite3", "./users.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create the table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password TEXT,
			visited_count INTEGER DEFAULT 0
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Add a sample user if the database is empty (for testing purposes)
	// Only run this if there is no user already in the database.
	_, err = db.Exec(`INSERT OR IGNORE INTO users (username, password) VALUES (?, ?)`, "testuser", "password123")
	if err != nil {
		log.Fatal(err)
	}

	// Set up routes
	http.HandleFunc("/", loginPage)
	http.HandleFunc("/login", loginHandler)

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
		</body>
		</html>
	`))

	tmpl.Execute(w, nil)
}

// loginHandler handles the login and increments the visit counter
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

		// Show the welcome page with the counter
		tmpl := template.Must(template.New("welcome").Parse(`
			<html>
			<body>
				<h2>Welcome {{.Username}}!</h2>
				<p>You have visited this page {{.VisitedCount}} times.</p>
			</body>
			</html>
		`))

		// Passing the updated user data to template
		tmpl.Execute(w, User{
			Username:     username,
			VisitedCount: visitedCount,
		})
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
