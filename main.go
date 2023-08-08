package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"html/template"
	"k8s.io/kubernetes/pkg/kubelet/util/store"
	"log"
	"net/http"
	"strconv"
	"time"
)

var db *gorm.DB // 全局变量用于存储数据库连接
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

// Article 文章模型
type Article struct {
	ID        uint `gorm:"primaryKey"`
	Title     string
	Content   string
	Author    string
	CreatedAt time.Time
	Comments  []Comment
}

// Comment 评论模型
type Comment struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	ArticleID uint
	Content   string
	CreatedAt time.Time
	Author    string
}

//type Comment struct {
//	ID        uint `gorm:"primaryKey;autoIncrement"`
//	ArticleID uint
//	Content   string
//	CreatedAt time.Time
//}

func main() {
	dsn := "root:123456@tcp(192.168.0.113:3306)/aoms?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 自动迁移数据库
	err = db.AutoMigrate(&Article{}, &Comment{})
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()

	router.HandleFunc("/", HomeHandler).Methods("GET")
	router.HandleFunc("/articles", ArticlesHandler).Methods("GET")
	router.HandleFunc("/articles/{id}", GetArticleHandler).Methods("GET")
	router.HandleFunc("/articles", CreateArticleHandler).Methods("POST")
	router.HandleFunc("/articles/{id}", UpdateArticleHandler).Methods("PUT")
	router.HandleFunc("/articles/{id}", DeleteArticleHandler).Methods("DELETE")
	router.HandleFunc("/new-article", authMiddleware(newArticleHandler)).Methods("GET", "POST")
	router.HandleFunc("/edit-article/{id}", authMiddleware(editArticleHandler)).Methods("GET", "POST")
	router.HandleFunc("/delete-article/{id}", authMiddleware(deleteArticleHandler)).Methods("POST")
	router.HandleFunc("/new-comment/{id}", newCommentHandler).Methods("POST")
	router.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	router.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	router.HandleFunc("/logout", logoutHandler).Methods("GET")

	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func editArticleHandler(w http.ResponseWriter, r *http.Request) {
	// ...
	var article Article // 添加这行

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
	var article Article // 添加这行

	// Delete the article and its related comments from the database
	db.Delete(&article)
	db.Delete(&Comment{}, "article_id = ?", article.ID)

	// ...
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
func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
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
func getArticleByID(id int) Article {
	var article Article
	db.Preload("Comments").First(&article, id)
	return article
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
		ArticleID: article.ID,
		Content:   content,
		Author:    author,
		CreatedAt: time.Now(),
	}

	db.Create(&comment)

	redirectURL := fmt.Sprintf("/article/%d", article.ID)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Blog!")
}

func ArticlesHandler(w http.ResponseWriter, r *http.Request) {
	var articles []Article
	var count int64

	// 获取分页参数
	page, _ := strconv.Atoi(r.FormValue("page"))
	pageSize, _ := strconv.Atoi(r.FormValue("page_size"))

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询文章列表和总记录数
	db.Preload("Comments").Limit(pageSize).Offset(offset).Find(&articles)
	db.Model(&Article{}).Count(&count)

	// 构建文章列表视图模型
	type ArticleViewModel struct {
		ID           uint
		Title        string
		Content      string
		Author       string
		CreatedAt    time.Time
		CommentCount int
	}

	var articleVMs []ArticleViewModel
	for _, article := range articles {
		articleVM := ArticleViewModel{
			ID:           article.ID,
			Title:        article.Title,
			Content:      article.Content,
			Author:       article.Author,
			CreatedAt:    article.CreatedAt,
			CommentCount: len(article.Comments),
		}
		articleVMs = append(articleVMs, articleVM)
	}

	// 构建文章列表页面模板
	tmpl := template.Must(template.ParseFiles("articles.html"))
	data := struct {
		Articles    []ArticleViewModel
		TotalCount  int64
		CurrentPage int
		HasPrevious bool
		HasNext     bool
	}{
		Articles:    articleVMs,
		TotalCount:  count,
		CurrentPage: page,
		HasPrevious: page > 1,
		HasNext:     int(count)/pageSize >= page,
	}

	// 渲染模板并返回给客户端
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func GetArticleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var article Article
	db.Preload("Comments").First(&article, id)

	// 检查文章是否存在
	if article.ID == 0 {
		http.NotFound(w, r)
		return
	}

	// 构建文章详情页面模板
	tmpl := template.Must(template.ParseFiles("article.html"))
	err := tmpl.Execute(w, article)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func CreateArticleHandler(w http.ResponseWriter, r *http.Request) {
	// 解析请求数据
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 提取表单数据
	title := r.FormValue("title")
	content := r.FormValue("content")
	author := r.FormValue("author")

	// 创建文章
	article := Article{
		Title:   title,
		Content: content,
		Author:  author,
	}
	db.Create(&article)

	// 返回成功响应
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Article created successfully")
}

func UpdateArticleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// 查询文章
	var article Article
	db.First(&article, id)

	// 检查文章是否存在
	if article.ID == 0 {
		http.NotFound(w, r)
		return
	}

	// 解析请求数据
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 更新文章
	article.Title = r.FormValue("title")
	article.Content = r.FormValue("content")
	article.Author = r.FormValue("author")
	db.Save(&article)

	// 返回成功响应
	fmt.Fprintf(w, "Article updated successfully")
}

func DeleteArticleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// 查询文章
	var article Article
	db.First(&article, id)

	// 检查文章是否存在
	if article.ID == 0 {
		http.NotFound(w, r)
		return
	}

	// 删除文章
	db.Delete(&article)

	// 返回成功响应
	fmt.Fprintf(w, "Article deleted successfully")
}
