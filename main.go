package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Article represents a blog article
type Article struct {
	ID        int
	Title     string
	Content   string
	Author    string
	CreatedAt time.Time
}

// Comment represents a comment on a blog article
type Comment struct {
	ID        int
	ArticleID int
	Content   string
	Author    string
	CreatedAt time.Time
}

var (
	articles  []Article
	comments  []Comment
	articleID int
	commentID int
	templates *template.Template
)

func main() {
	// Initialize templates
	templates = template.Must(template.ParseGlob("templates/*.html"))

	// Create some initial articles
	articles = append(articles,
		Article{ID: getNextArticleID(), Title: "First Article", Content: "This is the first article.", Author: "John Doe", CreatedAt: time.Now()},
		Article{ID: getNextArticleID(), Title: "Second Article", Content: "This is the second article.", Author: "Jane Smith", CreatedAt: time.Now()},
	)

	// Create a router
	router := mux.NewRouter()

	// Define routes
	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/article/{id}", articleHandler).Methods("GET")
	router.HandleFunc("/new-article", newArticleHandler).Methods("GET", "POST")
	router.HandleFunc("/edit-article/{id}", editArticleHandler).Methods("GET", "POST")
	router.HandleFunc("/delete-article/{id}", deleteArticleHandler).Methods("POST")
	router.HandleFunc("/new-comment/{id}", newCommentHandler).Methods("POST")

	// Start the server
	log.Println("Server started on http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", router))
}

func getNextArticleID() int {
	articleID++
	return articleID
}

func getNextCommentID() int {
	commentID++
	return commentID
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Articles []Article
	}{
		Articles: articles,
	}

	renderTemplate(w, "home.html", data)
}

func articleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var article Article
	for _, a := range articles {
		if a.ID == id {
			article = a
			break
		}
	}

	if article.ID == 0 {
		http.NotFound(w, r)
		return
	}

	comments := getCommentsByArticleID(article.ID)

	data := struct {
		Article  Article
		Comments []Comment
	}{
		Article:  article,
		Comments: comments,
	}

	renderTemplate(w, "article.html", data)
}

func newArticleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		title := r.FormValue("title")
		content := r.FormValue("content")
		author := r.FormValue("author")

		article := Article{
			ID:        getNextArticleID(),
			Title:     title,
			Content:   content,
			Author:    author,
			CreatedAt: time.Now(),
		}

		articles = append(articles, article)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	renderTemplate(w, "new_article.html", nil)
}

func newCommentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	article := getArticleByID(id)
	if article.ID == 0 {
		http.NotFound(w, r)
		return
	}

	content := r.FormValue("content")
	author := r.FormValue("author")

	comment := Comment{
		ID:        getNextCommentID(),
		ArticleID: article.ID,
		Content:   content,
		Author:    author,
		CreatedAt: time.Now(),
	}

	comments = append(comments, comment)

	redirectURL := fmt.Sprintf("/article/%d", article.ID)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func getArticleByID(id int) Article {
	for _, a := range articles {
		if a.ID == id {
			return a
		}
	}
	return Article{}
}

func getCommentsByArticleID(articleID int) []Comment {
	var result []Comment
	for _, c := range comments {
		if c.ArticleID == articleID {
			result = append(result, c)
		}
	}
	return result
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func editArticleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	article := getArticleByID(id)
	if article.ID == 0 {
		http.NotFound(w, r)
		return
	}

	if r.Method == "POST" {
		title := r.FormValue("title")
		content := r.FormValue("content")
		author := r.FormValue("author")

		article.Title = title
		article.Content = content
		article.Author = author

		redirectURL := fmt.Sprintf("/article/%d", article.ID)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	data := struct {
		Article Article
	}{
		Article: article,
	}

	renderTemplate(w, "edit_article.html", data)
}

func deleteArticleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	article := getArticleByID(id)
	if article.ID == 0 {
		http.NotFound(w, r)
		return
	}

	// Delete the article
	for i, a := range articles {
		if a.ID == id {
			articles = append(articles[:i], articles[i+1:]...)
			break
		}
	}

	// Delete related comments
	for i, c := range comments {
		if c.ArticleID == id {
			comments = append(comments[:i], comments[i+1:]...)
		}
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

type Article struct {
	gorm.Model
	Title    string
	Content  string
	Author   string
	Comments []Comment
}

type Comment struct {
	gorm.Model
	ArticleID uint
	Content   string
	Author    string
}

var db *gorm.DB

func main() {
	// ...

	// Connect to the database
	dsn := "user:password@tcp(localhost:3306)/blog_db?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Auto migrate the models to create corresponding tables
	err = db.AutoMigrate(&Article{}, &Comment{})
	if err != nil {
		log.Fatal(err)
	}

	// ...

	// Define routes
	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/article/{id}", articleHandler).Methods("GET")
	router.HandleFunc("/new-article", newArticleHandler).Methods("GET", "POST")
	router.HandleFunc("/edit-article/{id}", editArticleHandler).Methods("GET", "POST")
	router.HandleFunc("/delete-article/{id}", deleteArticleHandler).Methods("POST")
	router.HandleFunc("/new-comment/{id}", newCommentHandler).Methods("POST")

	// ...
}

func getArticleByID(id int) Article {
	var article Article
	db.First(&article, id)
	return article
}

func getCommentsByArticleID(articleID int) []Comment {
	var comments []Comment
	db.Where("article_id = ?", articleID).Find(&comments)
	return comments
}

func newArticleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		title := r.FormValue("title")
		content := r.FormValue("content")
		author := r.FormValue("author")

		article := Article{
			Title:   title,
			Content: content,
			Author:  author,
		}

		// Create the article in the database
		db.Create(&article)

		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	renderTemplate(w, "new_article.html", nil)
}

func editArticleHandler(w http.ResponseWriter, r *http.Request) {
	// ...

	if r.Method == "POST" {
		// ...

		// Update the article in the database
		db.Save(&article)

		redirectURL := fmt.Sprintf("/article/%d", article.ID)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// ...
}

func deleteArticleHandler(w http.ResponseWriter, r *http.Request) {
	// ...

	// Delete the article and its related comments from the database
	db.Delete(&article)
	db.Delete(&Comment{}, "article_id = ?", article.ID)

	// ...
}

type User struct {
	gorm.Model
	Username string `gorm:"unique"`
	Password string
}

type Pagination struct {
	TotalPages   int
	CurrentPage  int
	PageSize     int
	TotalRecords int
}

const pageSize = 10

// ...

func main() {
	// ...

	// Auto migrate the models to create corresponding tables
	err = db.AutoMigrate(&Article{}, &Comment{}, &User{})
	if err != nil {
		log.Fatal(err)
	}

	// ...

	// Define routes
	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/article/{id}", articleHandler).Methods("GET")
	router.HandleFunc("/new-article", authMiddleware(newArticleHandler)).Methods("GET", "POST")
	router.HandleFunc("/edit-article/{id}", authMiddleware(editArticleHandler)).Methods("GET", "POST")
	router.HandleFunc("/delete-article/{id}", authMiddleware(deleteArticleHandler)).Methods("POST")
	router.HandleFunc("/new-comment/{id}", newCommentHandler).Methods("POST")
	router.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	router.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	router.HandleFunc("/logout", logoutHandler).Methods("GET")

	// ...
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to register", http.StatusInternalServerError)
			return
		}

		user := User{
			Username: username,
			Password: string(hashedPassword),
		}

		// Create the user in the database
		db.Create(&user)

		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	renderTemplate(w, "register.html", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		var user User
		db.Where("username = ?", username).First(&user)

		if user.ID == 0 {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Compare the provided password with the stored hashed password
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Set a session cookie to indicate successful login
		session, _ := store.Get(r, "session")
		session.Values["authenticated"] = true
		session.Values["username"] = user.Username
		session.Save(r, w)

		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	renderTemplate(w, "login.html", nil)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["authenticated"] = false
	session.Values["username"] = ""
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		if session.Values["authenticated"] != true {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next(w, r)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	page := getPageNumber(r)

	var articles []Article
	db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&articles)

	var totalRecords int64
	db.Model(&Article{}).Count(&totalRecords)

	pagination := calculatePagination(page, int(totalRecords))

	data := struct {
		Articles   []Article
		Pagination Pagination
	}{
		Articles:   articles,
		Pagination: pagination,
	}

	renderTemplate(w, "home.html", data)
}

func calculatePagination(page, totalRecords int) Pagination {
	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	return Pagination{
		TotalPages:   totalPages,
		CurrentPage:  page,
		PageSize:     pageSize,
		TotalRecords: totalRecords,
	}
}

func getPageNumber(r *http.Request) int {
	page := 1

	pageParam := r.URL.Query().Get("page")
	if pageParam != "" {
		p, err := strconv.Atoi(pageParam)
		if err == nil && p > 0 {
			page = p
		}
	}

	return page
}
