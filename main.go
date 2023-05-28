package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// user represents
type User struct {
	Username string
	Email    string
	Password []byte
}

// Post represents a user post
type Post struct {
	ID           int
	Title        string
	Content      string
	CategoryID   int
	Comments     []Comment
	LikeCount    int
	DislikeCount int
}

// category represnet structure
type Category struct {
	ID   int
	Name string
}

// postcategoryAssociation represent structure
type PostCategoryAssociation struct {
	PostID     int64 `db:"post_id"`
	CategoryID int64 `db:"category_id"`
}

// Comment represents a user comment on a post
type Comment struct {
	ID      int
	PostID  int
	Content string
}

// Create a struct to represent the Like entry in the database
type Likes struct {
	ID     int
	PostID int
}

// Create a struct to represent the Dislike entry in the database
type Dislikes struct {
	ID     int
	PostID int
}

// database connection
var db *sql.DB

// main function to setup server and handles http requests
func main() {
	var err error
	db, err = sql.Open("sqlite3", "./data/forum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// called the created table
	err = createTables(db)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/login", login(db))
	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/readonly", readOnly)
	http.HandleFunc("/landingPageHandler", landingPageHandler)
	http.HandleFunc("/servicePageHandler", servicePageHandler)
	http.HandleFunc("/contactPageHandler", contactPageHandler)

	http.HandleFunc("/create", createPostHandler)
	http.HandleFunc("/comment", commentHandler)
	http.HandleFunc("/like", likeHandler)
	http.HandleFunc("/dislike", dislikeHandler)

	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/success", successHandler)
	log.Fatal(http.ListenAndServe(":5799", nil))
}

// function for the creation of table
func createTables(db *sql.DB) error {
	// Create user table if it does not exist
	createUserTable := `
		CREATE TABLE IF NOT EXISTS user (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uname TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		);
	`

	// Create posts table if it does not exist
	createPostsTable := `
	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		likes INTEGER DEFAULT 0,
		dislikes INTEGER DEFAULT 0
	);
`

	//Create comments table if it does not exist
	createCommentsTable := `
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			post_id INTEGER NOT NULL,
		FOREIGN KEY(post_id) REFERENCES posts(id)
		);
	`

	// Create likes_dislikes table if it does not exist
	createLikesDislikesTable := `
CREATE TABLE IF NOT EXISTS likes_dislikes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	post_id INTEGER NOT NULL,
	liked BOOLEAN NOT NULL
);
`

	//if error occur return error or  return nil if statement successful
	_, err := db.Exec(createUserTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createPostsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createCommentsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createLikesDislikesTable)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

//login function handler

func login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			// if not a POST request, render the login form
			t, err := template.ParseFiles("login.html")
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			err = t.Execute(w, nil)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			return
		}

		// parse form data
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := getUser(db, email)
		if err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword(user.Password, []byte(password))
		if err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// create session and set cookie
		expiration := time.Now().Add(24 * time.Hour)
		cookie := http.Cookie{
			Name:    "session",
			Value:   fmt.Sprintf("%s|%s", user.Username, user.Email),
			Expires: expiration,
			Path:    "/",
		}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/welcome", http.StatusSeeOther)

	}
}

//function to retrieves user from databse

func getUser(db *sql.DB, email string) (*User, error) {
	row := db.QueryRow("SELECT uname, email, password FROM user WHERE email = ?", email)

	var user User
	err := row.Scan(&user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// it is like the main page of the forum with all element needed
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("welcome.html")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Extract username and email from the session cookie value
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	values := strings.Split(cookie.Value, "|")
	username := values[0]
	email := values[1]

	// Connect to the SQLite database
	db, err := sql.Open("sqlite3", "./data/forum.db")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Retrieve posts from the "posts" table
	rows, err := db.Query(`
	SELECT posts.id, posts.title, posts.content, posts.like_count, posts.dislike_count, comments.id, comments.content
	FROM posts
	LEFT JOIN comments ON posts.id = comments.post_id
	ORDER BY posts.id DESC
`)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	posts := []Post{}
	currentPost := Post{}
	for rows.Next() {
		postID := 0
		postTitle := ""
		postContent := ""
		likeCount := 0
		dislikeCount := 0
		commentID := sql.NullInt64{}
		commentContent := sql.NullString{}

		err := rows.Scan(&postID, &postTitle, &postContent, &likeCount, &dislikeCount, &commentID, &commentContent)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Check if the post has changed
		if currentPost.ID != postID {
			// Append the current post to the posts slice if it's not the first post
			if currentPost.ID != 0 {
				posts = append(posts, currentPost)
			}
			// Create a new current post
			currentPost = Post{
				ID:           postID,
				Title:        postTitle,
				Content:      postContent,
				LikeCount:    likeCount,
				DislikeCount: dislikeCount,
			}
		}
		// Append the comment to the current post's comments slice if it exists
		if commentID.Valid && commentContent.Valid {
			currentPost.Comments = append(currentPost.Comments, Comment{
				ID:      int(commentID.Int64),
				Content: commentContent.String,
			})
		}
	}

	// Append the last post to the posts slice
	if currentPost.ID != 0 {
		posts = append(posts, currentPost)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create a context with the necessary data for the template
	context := struct {
		Username string
		Email    string
		Posts    []Post
	}{
		Username: username,
		Email:    email,
		Posts:    posts,
	}

	err = t.Execute(w, context)
	if err != nil {
		handleError(w, err)
		return
	}
}

func handleError(w http.ResponseWriter, err error) {
	http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
}

// it is like the main page of the forum with all element needed
func readOnly(w http.ResponseWriter, r *http.Request) {
	// Connect to the SQLite database
	db, err := sql.Open("sqlite3", "./data/forum.db")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Retrieve posts from the "posts" table
	rows, err := db.Query(`
	SELECT posts.id, posts.title, posts.content, posts.like_count, posts.dislike_count, comments.id, comments.content
	FROM posts
	LEFT JOIN comments ON posts.id = comments.post_id
	ORDER BY posts.id DESC
`)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	posts := []Post{}
	currentPost := Post{}
	for rows.Next() {
		var postID int
		var postTitle, postContent string
		var likeCount, dislikeCount int
		var commentID sql.NullInt64
		var commentContent sql.NullString

		err := rows.Scan(&postID, &postTitle, &postContent, &likeCount, &dislikeCount, &commentID, &commentContent)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if currentPost.ID != postID {
			if currentPost.ID != 0 {
				posts = append(posts, currentPost)
			}
			currentPost = Post{
				ID:           postID,
				Title:        postTitle,
				Content:      postContent,
				LikeCount:    likeCount,
				DislikeCount: dislikeCount,
			}
		}
		if commentID.Valid && commentContent.Valid {
			currentPost.Comments = append(currentPost.Comments, Comment{
				ID:      int(commentID.Int64),
				Content: commentContent.String,
			})
		}
	}

	if currentPost.ID != 0 {
		posts = append(posts, currentPost)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Parse the template file
	t, err := template.ParseFiles("readonly.html")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create a context with the necessary data for the template
	context := struct {
		Posts []Post
	}{
		Posts: posts,
	}

	// Execute the template with the provided context
	err = t.Execute(w, context)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func likeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	postID := r.Form.Get("post_id")

	// Connect to the SQLite database
	db, err := sql.Open("sqlite3", "./data/forum.db")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Update the like count in the "posts" table
	_, err = db.Exec("UPDATE posts SET like_count = like_count + 1 WHERE id = ?", postID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect the user back to the welcome page or show a success message
	http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

func dislikeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	postID := r.Form.Get("post_id")

	// Connect to the SQLite database
	db, err := sql.Open("sqlite3", "./data/forum.db")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Update the dislike count in the "posts" table
	_, err = db.Exec("UPDATE posts SET dislike_count = dislike_count + 1 WHERE id = ?", postID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect the user back to the welcome page or show a success message
	http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

//implement the function to enable comment on each post

func commentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	postID := r.Form.Get("post_id")
	commentContent := r.Form.Get("content")

	// Connect to the SQLite database
	db, err := sql.Open("sqlite3", "./data/forum.db")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Insert the comment into the "comments" table
	stmt, err := db.Prepare("INSERT INTO comments (post_id, content) VALUES (?, ?)")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(postID, commentContent)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect the user back to the welcome page or show a success message
	http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

//implemet function to create post by users

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// If not a POST request, render create post form
		t, err := template.ParseFiles("create_post.html")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		err = t.Execute(w, nil)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	}

	// Retrieve the form data
	title := r.FormValue("title")
	content := r.FormValue("content")
	categoryID := r.FormValue("category")

	// Insert the post into the database
	db, err := sql.Open("sqlite3", "./data/forum.db")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Insert the post into the "posts" table
	stmt, err := db.Prepare("INSERT INTO posts (title, content) VALUES (?, ?)")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(title, content)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	postID, err := res.LastInsertId()
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check if a category was selected
	if categoryID != "" {
		// Insert the association into the "post_category_associations" table
		stmt, err = db.Prepare("INSERT INTO post_category_associations (post_id, category_id) VALUES (?, ?)")
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		_, err := stmt.Exec(postID, categoryID)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// Redirect the user to the welcome page or show a success message
	http.Redirect(w, r, "/welcome", http.StatusSeeOther)
}

// implement the function for the logout session
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:    "session",
		Value:   "",
		Expires: time.Now().Add(-time.Hour), // Expire the cookie immediately
		Path:    "/",
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

//implement the signup functionality

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// if not a POST request, render the signup form
		t, err := template.ParseFiles("signup.html")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		err = t.Execute(w, nil)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	}

	// parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// get form values
	uname := r.FormValue("uname")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// check if email already exists in database
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM user WHERE email = ?", email)
	err = row.Scan(&count)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		// email already exists, render error message
		t, err := template.ParseFiles("signup.html")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		err = t.Execute(w, "Email already taken")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	}

	// encrypt password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// insert new user into database
	stmt, err := db.Prepare("INSERT INTO user (uname, email, password) VALUES (?, ?, ?)")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(uname, email, string(hashedPassword))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// render success message
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

//implement function for suceess message after registration

func successHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Signup successful!")
}

func landingPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("landingpage.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func servicePageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("service.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func contactPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("contact.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
